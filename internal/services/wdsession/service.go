package wdsession

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/selebrow/selebrow/internal/common/client"
	"github.com/selebrow/selebrow/internal/common/clock"
	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/config"
	"github.com/selebrow/selebrow/pkg/models"
)

type WDSessionService struct {
	mgr           browser.BrowserManager
	client        client.HTTPClient
	createTimeout time.Duration
	proxyDelete   bool
	sStorage      session.SessionStorage
	now           clock.NowFunc
	l             *zap.SugaredLogger
}

func NewWDSessionServiceImpl(
	mgr browser.BrowserManager,
	sStorage session.SessionStorage,
	hc client.HTTPClient,
	cfg config.WDSessionConfig,
	now clock.NowFunc,
	l *zap.Logger,
) *WDSessionService {
	return &WDSessionService{
		mgr:           mgr,
		client:        hc,
		createTimeout: cfg.CreateTimeout(),
		proxyDelete:   cfg.ProxyDelete(),
		sStorage:      sStorage,
		now:           now,
		l:             l.Sugar(),
	}
}

func (s *WDSessionService) ListSessions() []*session.Session {
	return s.sStorage.List(models.WebdriverProtocol)
}

func (s *WDSessionService) FindSession(id string) (*session.Session, error) {
	sess, ok := s.sStorage.Get(models.WebdriverProtocol, id)
	if !ok {
		return nil, fmt.Errorf("session %s doesn't exist", id)
	}
	return sess, nil
}

func (s *WDSessionService) DeleteSession(sess *session.Session) {
	if !s.sStorage.Delete(models.WebdriverProtocol, sess.ID()) {
		return
	}

	trash := !s.proxyDelete || !s.doDeleteSession(*sess.Browser().GetURL(), sess.Browser().GetHost(), sess.ID())
	if !trash {
		err := s.cleanupSession(context.Background(), sess)
		if err != nil {
			s.l.Warnw("failed to cleanup Webdriver Session", zap.Error(err))
			trash = true
		}
	}
	sess.Browser().Close(context.Background(), trash)
	s.l.Infow("Webdriver session has been deleted", zap.String("session_id", sess.ID()))
}

func (s *WDSessionService) CreateSession(ctx context.Context, reqCaps capabilities.Capabilities) (*session.Session, error) {
	if s.sStorage.IsShutdown() {
		return nil, session.ErrStorageShutdown
	}

	platform := normalizePlatform(reqCaps.GetPlatform())

	ctx, cancel := context.WithTimeout(ctx, s.createTimeout)
	defer cancel()
	start := time.Now()
	br, err := s.mgr.Allocate(ctx, models.WebdriverProtocol, reqCaps)
	if err != nil {
		return nil, models.WrapTimeoutErr(err, "failed to allocate webdriver")
	}

	err = s.waitWebdriverStarted(ctx, *br.GetURL(), br.GetHost())
	if err != nil {
		br.Close(context.Background(), true)
		return nil, models.WrapTimeoutErr(err, "webdriver did not get ready within configured timeout")
	}

	res, err := s.proxyCreateSession(ctx, *br.GetURL(), br.GetHost(), reqCaps)
	if err != nil {
		br.Close(context.Background(), true)
		return nil, models.WrapTimeoutErr(err, "failed to proxy create session request")
	}

	id, err := extractSessionID(res)
	if err != nil {
		br.Close(context.Background(), true)
		return nil, errors.Wrap(err, "failed to parse create session response")
	}

	sess := session.NewSession(id, platform, br, reqCaps, res, s.now(), nil, nil)
	if err := s.sStorage.Add(models.WebdriverProtocol, sess); err != nil {
		br.Close(context.Background(), true)
		return nil, errors.Wrap(err, "failed to store session")
	}

	s.l.With(
		zap.String("session_id", id),
		zap.String("browser_name", reqCaps.GetName()),
		zap.String("browser_version", reqCaps.GetVersion()),
		zap.String("url", br.GetURL().String()),
	).Infof("Webdriver session is ready in %v", time.Since(start))

	return sess, nil
}

func normalizePlatform(platform string) string {
	if platform == "" {
		platform = browser.DefaultPlatform
	}
	return strings.ToUpper(platform)
}

func (s *WDSessionService) waitWebdriverStarted(ctx context.Context, u url.URL, host string) error {
	var err error
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	u.Path = path.Join(u.Path, "status")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return err
	}
	req.Host = host
	for {
		var res *http.Response
		res, err = s.client.Do(req)
		if err == nil {
			_ = res.Body.Close()
			if res.StatusCode < http.StatusBadRequest {
				return nil
			}
			err = fmt.Errorf("request %s failed with code %d", u.String(), res.StatusCode)
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

func (s *WDSessionService) proxyCreateSession(
	ctx context.Context,
	u url.URL,
	host string,
	caps capabilities.Capabilities,
) (map[string]interface{}, error) {
	u.Path = path.Join(u.Path, router.SessionPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(caps.GetRawCapabilities()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Host = host

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("Create request failed on %s with HTTP code %d: %s", req.URL, resp.StatusCode, string(respBody))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res, nil
}

func extractSessionID(resp map[string]interface{}) (string, error) {
	id, ok := resp["sessionId"]
	if ok {
		sess, ok := id.(string)
		if !ok {
			return "", errors.New("failed to cast sessionId to string")
		}

		return sess, nil
	}

	value, ok := resp["value"]
	if !ok {
		return "", errors.New("wrong response structure")
	}

	sessIDMap, ok := value.(map[string]interface{})
	if !ok {
		return "", errors.New("failed to cast value to map")
	}

	id, ok = sessIDMap["sessionId"]
	if !ok {
		return "", errors.New("wrong response structure")
	}

	sess, ok := id.(string)
	if !ok {
		return "", errors.New("failed to cast sessionId to string")
	}

	return sess, nil
}

func (s *WDSessionService) doDeleteSession(u url.URL, host, id string) bool {
	u.Path = path.Join(u.Path, router.SessionPath, id)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, u.String(), http.NoBody)
	req.Host = host
	resp, err := s.client.Do(req)
	if err != nil {
		s.l.Errorw("failed to close webdriver session", zap.String("url", u.String()), zap.Error(err))
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.l.Errorw("unexpected HTTP response code when closing webdriver session",
			zap.String("url", u.String()), zap.Int("status", resp.StatusCode))
		return false
	}

	return true
}

func (s *WDSessionService) cleanupSession(ctx context.Context, sess *session.Session) error {
	hp := sess.Browser().GetHostPort(models.FileserverPort)
	if hp == "" {
		return nil
	}
	baseURL, err := url.Parse("http://" + hp)
	if err != nil {
		return err
	}

	files, err := s.getFiles(ctx, baseURL)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := s.deleteFile(ctx, baseURL, file); err != nil {
			return err
		}
	}

	if l := len(files); l > 0 {
		s.l.With(zap.String("session_id", sess.ID())).Infof("Webdriver session cleanup: %d files have been deleted", l)
	}
	return nil
}

func (s *WDSessionService) getFiles(ctx context.Context, baseURL *url.URL) ([]string, error) {
	u := *baseURL
	q := make(url.Values)
	q.Set("json", "true")
	u.RawQuery = q.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res []string
	err = json.NewDecoder(resp.Body).Decode(&res)
	return res, err
}

func (s *WDSessionService) deleteFile(ctx context.Context, baseURL *url.URL, file string) error {
	u := *baseURL
	u.Path = path.Join(u.Path, file)
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), http.NoBody)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 && resp.StatusCode != http.StatusNotFound {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			s.l.Errorw("failed to read the response", zap.Error(err))
		}

		return errors.Errorf("%s %s failed with code %d: %s", r.Method, r.URL.String(), resp.StatusCode, string(b))
	}

	return nil
}
