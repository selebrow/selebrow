package pw

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/models"
)

var testURL, _ = url.Parse("http://testhost:5678")

func TestPWSessionServiceImpl_CreateSession(t *testing.T) {
	g := NewWithT(t)
	m := new(mocks.BrowserManager)
	d := new(mocks.ContextDialer)
	ss := new(mocks.SessionStorage)
	testTime := time.UnixMilli(123)
	now := func() time.Time { return testTime }
	s := NewPWSessionService(m, ss, d, time.Second, false, now, zaptest.NewLogger(t))

	br := new(mocks.Browser)

	genSessionID = func() string {
		return "12345"
	}
	ss.EXPECT().IsShutdown().Return(false).Once()

	var crCtx context.Context
	m.EXPECT().Allocate(mock.Anything, models.PlaywrightProtocol, &models.PWCapabilities{
		Flavor:  "some",
		Browser: "test",
		Version: "v2",
	}).Run(func(ctx context.Context, _ models.BrowserProtocol, _ capabilities.Capabilities) {
		dl, ok := ctx.Deadline()
		g.Expect(ok).To(BeTrue())
		g.Expect(dl).To(BeTemporally("~", time.Now().Add(time.Second), 100*time.Millisecond))
		crCtx = ctx
	}).Return(br, nil).Once()

	br.EXPECT().GetURL().Return(testURL)
	conn, _ := net.Pipe()
	d.EXPECT().DialContext(mock.Anything, "tcp", "testhost:5678").Run(func(ctx context.Context, _ string, _ string) {
		g.Expect(ctx).To(BeIdenticalTo(crCtx))
	}).Return(conn, nil).Once()

	caps := &models.PWCapabilities{Flavor: "some", Browser: "test", Version: "v2"}
	var savedSess *session.Session
	ss.EXPECT().Add(models.PlaywrightProtocol, mock.Anything).RunAndReturn(func(_ models.BrowserProtocol, sess *session.Session) error {
		g.Expect(sess.ID()).To(Equal("12345"))
		g.Expect(sess.Platform()).To(Equal("LINUX"))
		g.Expect(sess.Browser()).To(BeIdenticalTo(br))
		g.Expect(sess.ReqCaps()).To(BeIdenticalTo(caps))
		g.Expect(sess.Created()).To(Equal(testTime))
		g.Expect(sess.Context()).ToNot(BeNil())
		g.Expect(sess.Cancel()).ToNot(BeNil())
		savedSess = sess
		return nil
	})
	got, err := s.CreateSession(context.TODO(), caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got).To(BeIdenticalTo(savedSess))

	_, err = conn.Read(nil)
	g.Expect(err).To(MatchError(io.ErrClosedPipe))

	ss.AssertExpectations(t)
	m.AssertExpectations(t)
	br.AssertExpectations(t)
	d.AssertExpectations(t)
}

func TestPWSessionServiceImpl_CreateSession_Allocate_Error(t *testing.T) {
	g := NewWithT(t)
	m := new(mocks.BrowserManager)
	d := new(mocks.ContextDialer)
	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(m, ss, d, time.Second, false, nil, zaptest.NewLogger(t))

	ss.EXPECT().IsShutdown().Return(false).Once()

	m.EXPECT().Allocate(mock.Anything, models.PlaywrightProtocol, mock.Anything).Return(nil, errors.New("test error")).Once()

	_, err := s.CreateSession(context.TODO(), &models.PWCapabilities{Flavor: "some", Browser: "test", Version: "v2"})
	g.Expect(err).To(MatchError(MatchRegexp("failed to allocate.*test error")))

	ss.AssertExpectations(t)
	m.AssertExpectations(t)
}

func TestPWServiceImpl_CreateSession_Allocate_Timeout(t *testing.T) {
	g := NewWithT(t)
	m := new(mocks.BrowserManager)
	d := new(mocks.ContextDialer)
	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(m, ss, d, time.Nanosecond, false, nil, zaptest.NewLogger(t))

	ss.EXPECT().IsShutdown().Return(false).Once()

	m.EXPECT().
		Allocate(mock.Anything, models.PlaywrightProtocol, mock.Anything).
		RunAndReturn(func(ctx context.Context, _ models.BrowserProtocol, _ capabilities.Capabilities) (browser.Browser, error) {
			return nil, ctx.Err()
		}).
		Once()

	_, err := s.CreateSession(context.TODO(), &models.PWCapabilities{Flavor: "some", Browser: "test", Version: "v2"})
	g.Expect(err).To(MatchError(context.DeadlineExceeded))
	var e models.ErrorWithCode
	g.Expect(errors.As(err, &e)).To(BeTrue())
	g.Expect(e.Code()).To(Equal(http.StatusGatewayTimeout))

	ss.AssertExpectations(t)
	m.AssertExpectations(t)
}

func TestPWServiceImpl_CreateSession_Start_Failed(t *testing.T) {
	g := NewWithT(t)
	m := new(mocks.BrowserManager)
	d := new(mocks.ContextDialer)
	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(m, ss, d, 500*time.Millisecond, false, nil, zaptest.NewLogger(t))

	ss.EXPECT().IsShutdown().Return(false).Once()

	br := new(mocks.Browser)
	m.EXPECT().Allocate(mock.Anything, models.PlaywrightProtocol, mock.Anything).Return(br, nil).Once()

	br.EXPECT().GetURL().Return(testURL)
	d.EXPECT().DialContext(mock.Anything, "tcp", "testhost:5678").Return(nil, errors.New("test error"))
	br.EXPECT().Close(context.Background(), true)

	_, err := s.CreateSession(context.TODO(), &models.PWCapabilities{Flavor: "some", Browser: "test", Version: "v2"})
	g.Expect(err).To(MatchError(MatchRegexp("test error")))

	ss.AssertExpectations(t)
	m.AssertExpectations(t)
	br.AssertExpectations(t)
	d.AssertNumberOfCalls(t, "DialContext", 3)
}

func TestPWSessionServiceImpl_CreateSession_StorageError(t *testing.T) {
	g := NewWithT(t)
	m := new(mocks.BrowserManager)
	d := new(mocks.ContextDialer)
	ss := new(mocks.SessionStorage)
	now := func() time.Time { return time.UnixMilli(123) }
	s := NewPWSessionService(m, ss, d, time.Second, false, now, zaptest.NewLogger(t))

	br := new(mocks.Browser)

	ss.EXPECT().IsShutdown().Return(false).Once()

	caps := &models.PWCapabilities{Flavor: "some", Browser: "test", Version: "v2"}
	m.EXPECT().Allocate(mock.Anything, models.PlaywrightProtocol, caps).Return(br, nil).Once()

	br.EXPECT().GetURL().Return(testURL)
	conn, _ := net.Pipe()
	d.EXPECT().DialContext(mock.Anything, "tcp", "testhost:5678").Return(conn, nil).Once()

	br.EXPECT().Close(context.Background(), true)
	ss.EXPECT().Add(models.PlaywrightProtocol, mock.Anything).Return(errors.New("test error"))
	_, err := s.CreateSession(context.TODO(), caps)
	g.Expect(err).To(MatchError(MatchRegexp("failed to store.*test error")))

	ss.AssertExpectations(t)
	m.AssertExpectations(t)
	br.AssertExpectations(t)
	d.AssertExpectations(t)
}

func TestPWSessionServiceImpl_CreateSession_Shutdown(t *testing.T) {
	g := NewWithT(t)
	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(nil, ss, nil, time.Second, false, nil, zaptest.NewLogger(t))

	ss.EXPECT().IsShutdown().Return(true).Once()

	_, err := s.CreateSession(context.TODO(), &models.PWCapabilities{Flavor: "some", Browser: "test", Version: "v2"})
	g.Expect(err).To(MatchError(session.ErrStorageShutdown))

	ss.AssertExpectations(t)
}

func TestPWSessionServiceImpl_ListSessions(t *testing.T) {
	g := NewWithT(t)
	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(nil, ss, nil, time.Second, false, nil, zaptest.NewLogger(t))

	testSessList := []*session.Session{{}, {}}

	ss.EXPECT().List(models.PlaywrightProtocol).Return(testSessList).Once()
	sessList := s.ListSessions()
	g.Expect(sessList).To(Equal(testSessList))

	ss.AssertExpectations(t)
}

func TestPWSessionServiceImpl_DeleteSession(t *testing.T) {
	g := NewWithT(t)

	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(nil, ss, nil, time.Second, false, nil, zaptest.NewLogger(t))

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	br := new(mocks.Browser)
	s1 := session.NewSession("12345", "", br, nil, nil, time.Time{}, ctx, cancel)

	ss.EXPECT().Delete(models.PlaywrightProtocol, "12345").Return(true)
	br.EXPECT().Close(context.Background(), false)
	s.DeleteSession(s1)
	g.Expect(ctx.Done()).To(BeClosed())

	ss.AssertExpectations(t)
}

func TestWDSessionServiceImpl_DeleteSession_AlreadyDeleted(t *testing.T) {
	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(nil, ss, nil, time.Second, false, nil, zaptest.NewLogger(t))

	s1 := session.NewSession("12345", "", nil, nil, nil, time.Time{}, nil, nil)

	ss.EXPECT().Delete(models.PlaywrightProtocol, "12345").Return(false)
	s.DeleteSession(s1)

	ss.AssertExpectations(t)
}

func TestWDSessionServiceImpl_FindSession(t *testing.T) {
	g := NewWithT(t)

	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(nil, ss, nil, time.Second, false, nil, zaptest.NewLogger(t))

	s1 := session.NewSession("12345", "", nil, nil, nil, time.Time{}, nil, nil)
	ss.EXPECT().Get(models.PlaywrightProtocol, "12345").Return(s1, true).Once()
	got, err := s.FindSession("12345")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got).To(BeIdenticalTo(s1))

	ss.AssertExpectations(t)
}

func TestWDSessionServiceImpl_FindSession_NotFound(t *testing.T) {
	g := NewWithT(t)

	ss := new(mocks.SessionStorage)
	s := NewPWSessionService(nil, ss, nil, time.Second, false, nil, zaptest.NewLogger(t))

	ss.EXPECT().Get(models.PlaywrightProtocol, "12345").Return(nil, false).Once()
	_, err := s.FindSession("12345")
	g.Expect(err).To(MatchError(MatchRegexp("doesn't exist")))

	ss.AssertExpectations(t)
}
