package pw

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"

	"github.com/selebrow/selebrow/internal/common/clock"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/models"
)

const tcpConnectionReadTimeout = 20 * time.Millisecond

var (
	genSessionID = uuid.NewString
)

type PWSessionService struct {
	mgr           browser.BrowserManager
	createTimeout time.Duration
	d             proxy.ContextDialer
	l             *zap.SugaredLogger
	checkConn     bool
	now           clock.NowFunc
	sStorage      session.SessionStorage
}

func NewPWSessionService(
	mgr browser.BrowserManager,
	sStorage session.SessionStorage,
	d proxy.ContextDialer,
	createTimeout time.Duration,
	checkConn bool,
	now clock.NowFunc,
	l *zap.Logger,
) *PWSessionService {
	return &PWSessionService{
		mgr:           mgr,
		createTimeout: createTimeout,
		d:             d,
		checkConn:     checkConn,
		l:             l.Sugar(),
		sStorage:      sStorage,
		now:           now,
	}
}

func (s *PWSessionService) CreateSession(ctx context.Context, caps capabilities.Capabilities) (*session.Session, error) {
	if s.sStorage.IsShutdown() {
		return nil, session.ErrStorageShutdown
	}

	start := time.Now()
	br, err := s.createBrowser(ctx, caps)
	if err != nil {
		return nil, err
	}

	id := genSessionID()
	sCtx, cancel := context.WithCancel(ctx)
	sess := session.NewSession(id, browser.DefaultPlatform, br, caps, nil, s.now(), sCtx, cancel)
	if err := s.sStorage.Add(models.PlaywrightProtocol, sess); err != nil {
		br.Close(context.Background(), true)
		return nil, errors.Wrap(err, "failed to store session")
	}

	s.l.With(zap.String("session_id", id),
		zap.String("browser_name", caps.GetName()),
		zap.String("browser_version", caps.GetVersion()),
		zap.String("url", br.GetURL().String())).
		Infof("Playwright session is ready in %v", time.Since(start))
	return sess, err
}

func (s *PWSessionService) FindSession(id string) (*session.Session, error) {
	// not used currently
	sess, ok := s.sStorage.Get(models.PlaywrightProtocol, id)
	if !ok {
		return nil, fmt.Errorf("session %s doesn't exist", id)
	}
	return sess, nil
}

func (s *PWSessionService) ListSessions() []*session.Session {
	return s.sStorage.List(models.PlaywrightProtocol)
}

func (s *PWSessionService) DeleteSession(sess *session.Session) {
	if !s.sStorage.Delete(models.PlaywrightProtocol, sess.ID()) {
		return
	}

	sess.Cancel()() // cancel context to reset any active connections
	sess.Browser().Close(context.Background(), false)
	s.l.Infow("Playwright session has been deleted", zap.String("session_id", sess.ID()))
}

func (s *PWSessionService) createBrowser(ctx context.Context, caps capabilities.Capabilities) (browser.Browser, error) {
	ctx, cancel := context.WithTimeout(ctx, s.createTimeout)
	defer cancel()
	br, err := s.mgr.Allocate(ctx, models.PlaywrightProtocol, caps)
	if err != nil {
		return nil, models.WrapTimeoutErr(err, "failed to allocate playwright browser")
	}

	err = s.waitBrowserServerStarted(ctx, br.GetURL().Host)
	if err != nil {
		br.Close(context.Background(), true)
		return nil, models.WrapTimeoutErr(err, "browser server did not get ready within configured timeout")
	}

	return br, nil
}

func (s *PWSessionService) waitBrowserServerStarted(ctx context.Context, hostport string) error {
	var (
		err  error
		conn net.Conn
	)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	buf := make([]byte, 1)
	for {
		conn, err = s.d.DialContext(ctx, "tcp", hostport)
		if err == nil {
			if !s.checkConn {
				_ = conn.Close()
				return nil
			}
			// XXX must go away when we have browserserver with healthcheck
			if err := conn.SetReadDeadline(time.Now().Add(tcpConnectionReadTimeout)); err != nil {
				_ = conn.Close()
				return errors.Wrap(err, "failed to set read timeout on test connection")
			}

			// if process is listening in container, connection would block on read, otherwise we would get
			// io.EOF or any other errors (fix for docker forwarded connections)
			_, err = conn.Read(buf)
			_ = conn.Close()
			if err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return errors.Wrapf(ctx.Err(), "last error was: %s", err.Error())
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
