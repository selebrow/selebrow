package wdsession_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/internal/services/wdsession"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/models"
)

func TestWDSessionServiceImpl_CreateSession(t *testing.T) {
	g := NewWithT(t)
	client := mocks.NewHTTPClient(t)
	cfg := createCfg(t, time.Second, false)
	mgr := mocks.NewBrowserManager(t)
	ss := mocks.NewSessionStorage(t)
	createTime := time.UnixMilli(123)
	now := func() time.Time { return createTime }
	svc := wdsession.NewWDSessionServiceImpl(mgr, ss, client, cfg, now, 0, zaptest.NewLogger(t))

	caps := mocks.NewCapabilities(t)
	caps.EXPECT().GetPlatform().Return("cp/m")
	caps.EXPECT().GetName().Return("opera")
	caps.EXPECT().GetVersion().Return("123.23")
	caps.EXPECT().GetRawCapabilities().Return([]byte("{}"))

	ss.EXPECT().IsShutdown().Return(false).Once()

	br := mocks.NewBrowser(t)
	expDeadline := time.Now().Add(time.Second)
	mgr.EXPECT().
		Allocate(mock.Anything, models.WebdriverProtocol, caps).
		Run(func(ctx context.Context, _ models.BrowserProtocol, _ capabilities.Capabilities) {
			dl, ok := ctx.Deadline()
			g.Expect(ok).To(BeTrue())
			g.Expect(dl).To(BeTemporally("~", expDeadline, 100*time.Millisecond))
		}).
		Return(br, nil).
		Once()

	u, err := url.Parse("http://host:123")
	g.Expect(err).ToNot(HaveOccurred())
	br.EXPECT().GetURL().Return(u)
	br.EXPECT().GetHost().Return("hst:111")

	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		expUrl := *u
		expUrl.Path = "/status"
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL).To(Equal(&expUrl))
		g.Expect(req.Host).To(Equal("hst:111"))
	}).Return(nil, errors.New("test err")).Once()

	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		expUrl := *u
		expUrl.Path = "/status"
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL).To(Equal(&expUrl))
		g.Expect(req.Host).To(Equal("hst:111"))
	}).Return(&http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader(``))}, nil).Once()

	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		expUrl := *u
		expUrl.Path = "/status"
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL).To(Equal(&expUrl))
		g.Expect(req.Host).To(Equal("hst:111"))
	}).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))}, nil).Once()

	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		expUrl := *u
		expUrl.Path = "/session"
		g.Expect(req.Method).To(Equal(http.MethodPost))
		g.Expect(req.URL).To(Equal(&expUrl))
		g.Expect(req.Host).To(Equal("hst:111"))
		g.Expect(req.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
	}).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"value":{"sessionId":"123"}}`))}, nil).Once()

	var savedSess *session.Session
	ss.EXPECT().Add(models.WebdriverProtocol, mock.Anything).RunAndReturn(func(_ models.BrowserProtocol, sess *session.Session) error {
		g.Expect(sess.ID()).To(Equal("123"))
		g.Expect(sess.Created()).To(Equal(createTime))
		g.Expect(sess.ReqCaps()).To(BeIdenticalTo(caps))
		g.Expect(sess.Browser()).To(BeIdenticalTo(br))
		g.Expect(sess.Platform()).To(Equal("CP/M"))
		g.Expect(sess.Resp()).To(Equal(map[string]interface{}{
			"value": map[string]interface{}{
				"sessionId": "123",
			},
		}))
		g.Expect(sess.Context()).To(BeNil())
		g.Expect(sess.Cancel()).To(BeNil())

		savedSess = sess
		return nil
	})

	sess, err := svc.CreateSession(context.TODO(), caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sess).To(BeIdenticalTo(savedSess))
	g.Expect(sess.LastUsed()).To(Equal(createTime))
}

func TestWDSessionServiceImpl_CreateSessionDefaultPlatform(t *testing.T) {
	g := NewWithT(t)

	client := mocks.NewHTTPClient(t)
	cfg := createCfg(t, time.Second, false)
	mgr := mocks.NewBrowserManager(t)
	ss := mocks.NewSessionStorage(t)
	now := func() time.Time { return time.Time{} }
	svc := wdsession.NewWDSessionServiceImpl(mgr, ss, client, cfg, now, 0, zaptest.NewLogger(t))

	sess, err := createSession(t, g, svc, ss, mgr, client, "", "netscape", "11", "http://host1", "s1", "hst:11111")
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(sess.Platform()).To(Equal(browser.DefaultPlatform))
}

func TestWDSessionServiceImpl_CreateSessionTimeout(t *testing.T) {
	g := NewWithT(t)

	cfg := createCfg(t, time.Nanosecond, false)
	mgr := mocks.NewBrowserManager(t)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(mgr, ss, nil, cfg, nil, 0, zaptest.NewLogger(t))

	ss.EXPECT().IsShutdown().Return(false).Once()

	caps := mocks.NewCapabilities(t)
	caps.EXPECT().GetPlatform().Return("")

	mgr.EXPECT().
		Allocate(mock.Anything, models.WebdriverProtocol, caps).
		RunAndReturn(func(ctx context.Context, _ models.BrowserProtocol, _ capabilities.Capabilities) (browser.Browser, error) {
			return nil, ctx.Err()
		})
	_, err := svc.CreateSession(context.TODO(), caps)
	g.Expect(err).To(MatchError(context.DeadlineExceeded))
	var e models.ErrorWithCode
	g.Expect(errors.As(err, &e)).To(BeTrue())
	g.Expect(e.Code()).To(Equal(http.StatusGatewayTimeout))
}

func TestWDSessionServiceImpl_CreateSession_Shutdown(t *testing.T) {
	g := NewWithT(t)

	cfg := createCfg(t, time.Second, false)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, nil, cfg, nil, 0, zaptest.NewLogger(t))

	ss.EXPECT().IsShutdown().Return(true).Once()
	_, err := svc.CreateSession(context.TODO(), nil)
	g.Expect(err).To(MatchError(session.ErrStorageShutdown))
}

func TestWDSessionServiceImpl_CreateSession_StorageError(t *testing.T) {
	g := NewWithT(t)
	client := mocks.NewHTTPClient(t)
	cfg := createCfg(t, time.Second, false)
	mgr := mocks.NewBrowserManager(t)
	ss := mocks.NewSessionStorage(t)
	now := func() time.Time { return time.UnixMilli(123) }
	svc := wdsession.NewWDSessionServiceImpl(mgr, ss, client, cfg, now, 0, zaptest.NewLogger(t))

	caps := mocks.NewCapabilities(t)
	caps.EXPECT().GetPlatform().Return("cp/m")
	caps.EXPECT().GetRawCapabilities().Return([]byte("{}"))

	ss.EXPECT().IsShutdown().Return(false).Once()

	br := mocks.NewBrowser(t)
	mgr.EXPECT().Allocate(mock.Anything, models.WebdriverProtocol, caps).Return(br, nil).Once()

	u, err := url.Parse("http://host:123")
	g.Expect(err).ToNot(HaveOccurred())
	br.EXPECT().GetURL().Return(u)
	br.EXPECT().GetHost().Return("hst:111")

	client.EXPECT().Do(mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))}, nil).Once()

	client.EXPECT().Do(mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"value":{"sessionId":"123"}}`))}, nil).
		Once()

	ss.EXPECT().Add(models.WebdriverProtocol, mock.Anything).Return(errors.New("test error"))
	br.EXPECT().Close(context.Background(), true).Once()
	_, err = svc.CreateSession(context.TODO(), caps)
	g.Expect(err).To(MatchError(MatchRegexp("failed to store.*test error")))
}

func TestWDSessionServiceImpl_ListSessions(t *testing.T) {
	g := NewWithT(t)
	cfg := createCfg(t, time.Second, false)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, nil, cfg, nil, 0, zaptest.NewLogger(t))

	testSessList := []*session.Session{{}, {}}

	ss.EXPECT().List(models.WebdriverProtocol).Return(testSessList).Once()
	sessList := svc.ListSessions()
	g.Expect(sessList).To(Equal(testSessList))
}

func TestWDSessionServiceImpl_DeleteSession(t *testing.T) {
	cfg := createCfg(t, time.Second, false)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, nil, cfg, nil, 0, zaptest.NewLogger(t))

	br1 := mocks.NewBrowser(t)
	s1 := session.NewSession("12345", "", br1, nil, nil, time.Time{}, nil, nil)

	ss.EXPECT().Delete(models.WebdriverProtocol, "12345").Return(true).Once()
	br1.EXPECT().Close(context.Background(), true).Once()
	svc.DeleteSession(s1)
}

func TestWDSessionServiceImpl_DeleteSessionProxy(t *testing.T) {
	g := NewWithT(t)

	client := mocks.NewHTTPClient(t)
	cfg := createCfg(t, time.Second, true)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, client, cfg, nil, 0, zaptest.NewLogger(t))

	u, err := url.Parse("http://host1")
	g.Expect(err).ToNot(HaveOccurred())

	br1 := mocks.NewBrowser(t)
	s1 := session.NewSession("s1", "", br1, nil, nil, time.Time{}, nil, nil)

	ss.EXPECT().Delete(models.WebdriverProtocol, "s1").Return(true).Once()
	br1.EXPECT().GetURL().Return(u)
	br1.EXPECT().GetHost().Return("hst:11111")

	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		expUrl := *u
		expUrl.Path = "/session/s1"
		g.Expect(req.Method).To(Equal(http.MethodDelete))
		g.Expect(req.URL).To(Equal(&expUrl))
		g.Expect(req.Host).To(Equal("hst:11111"))
	}).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))}, nil).Once()

	br1.EXPECT().GetHostPort(models.FileserverPort).Return("host1:3322").Once()
	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL.String()).To(Equal("http://host1:3322?json=true"))
	}).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[ "file1.txt" ]`))}, nil).Once()

	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		g.Expect(req.Method).To(Equal(http.MethodDelete))
		g.Expect(req.URL.String()).To(Equal("http://host1:3322/file1.txt"))
	}).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))}, nil).Once()
	br1.EXPECT().Close(context.Background(), false).Once()

	svc.DeleteSession(s1)
}

func TestWDSessionServiceImpl_DeleteSessionTrash(t *testing.T) {
	g := NewWithT(t)

	client := mocks.NewHTTPClient(t)
	cfg := createCfg(t, time.Second, true)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, client, cfg, nil, 0, zaptest.NewLogger(t))

	u, err := url.Parse("http://host1")
	g.Expect(err).ToNot(HaveOccurred())

	br1 := mocks.NewBrowser(t)
	s1 := session.NewSession("s1", "", br1, nil, nil, time.Time{}, nil, nil)

	ss.EXPECT().Delete(models.WebdriverProtocol, "s1").Return(true).Once()
	br1.EXPECT().GetURL().Return(u)
	br1.EXPECT().GetHost().Return("hst:11111")
	br1.EXPECT().Close(context.Background(), true).Once()
	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		expUrl := *u
		expUrl.Path = "/session/s1"
		g.Expect(req.Method).To(Equal(http.MethodDelete))
		g.Expect(req.URL).To(Equal(&expUrl))
		g.Expect(req.Host).To(Equal("hst:11111"))
	}).Return(&http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader(``))}, nil)
	svc.DeleteSession(s1)
}

func TestWDSessionServiceImpl_DeleteSession_CleanupTrash(t *testing.T) {
	g := NewWithT(t)

	client := mocks.NewHTTPClient(t)
	cfg := createCfg(t, time.Second, true)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, client, cfg, nil, 0, zaptest.NewLogger(t))

	u, err := url.Parse("http://host1")
	g.Expect(err).ToNot(HaveOccurred())

	br1 := mocks.NewBrowser(t)
	s1 := session.NewSession("s1", "", br1, nil, nil, time.Time{}, nil, nil)

	ss.EXPECT().Delete(models.WebdriverProtocol, "s1").Return(true).Once()
	br1.EXPECT().GetURL().Return(u)
	br1.EXPECT().GetHost().Return("hst:11111")

	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		expUrl := *u
		expUrl.Path = "/session/s1"
		g.Expect(req.Method).To(Equal(http.MethodDelete))
		g.Expect(req.URL).To(Equal(&expUrl))
		g.Expect(req.Host).To(Equal("hst:11111"))
	}).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))}, nil).Once()

	br1.EXPECT().GetHostPort(models.FileserverPort).Return("host1:3322").Once()
	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL.String()).To(Equal("http://host1:3322?json=true"))
	}).Return(nil, errors.New("test error")).Once()
	br1.EXPECT().Close(context.Background(), true).Once()

	svc.DeleteSession(s1)
}

func TestWDSessionServiceImpl_DeleteSession_AlreadyDeleted(t *testing.T) {
	cfg := createCfg(t, time.Second, false)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, nil, cfg, nil, 0, zaptest.NewLogger(t))

	s1 := session.NewSession("12345", "", nil, nil, nil, time.Time{}, nil, nil)

	ss.EXPECT().Delete(models.WebdriverProtocol, "12345").Return(false)
	svc.DeleteSession(s1)
}

func TestWDSessionServiceImpl_FindSession(t *testing.T) {
	g := NewWithT(t)

	cfg := createCfg(t, time.Second, false)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, nil, cfg, nil, 0, zaptest.NewLogger(t))

	s1 := session.NewSession("12345", "", nil, nil, nil, time.Time{}, nil, nil)
	ss.EXPECT().Get(models.WebdriverProtocol, "12345").Return(s1, true).Once()
	got, err := svc.FindSession("12345")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got).To(BeIdenticalTo(s1))
}

func TestWDSessionServiceImpl_FindSession_NotFound(t *testing.T) {
	g := NewWithT(t)

	cfg := createCfg(t, time.Second, false)
	ss := mocks.NewSessionStorage(t)
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, nil, cfg, nil, 0, zaptest.NewLogger(t))

	ss.EXPECT().Get(models.WebdriverProtocol, "12345").Return(nil, false).Once()
	_, err := svc.FindSession("12345")
	g.Expect(err).To(MatchError(MatchRegexp("doesn't exist")))
}

func TestWDSessionServiceImpl_Shutdown_NoOp(t *testing.T) {
	g := NewWithT(t)

	cfg := createCfg(t, time.Second, false)
	svc := wdsession.NewWDSessionServiceImpl(nil, nil, nil, cfg, nil, 0, zaptest.NewLogger(t))

	err := svc.Shutdown(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
}

func TestWDSessionServiceImpl_CleanupSessions(t *testing.T) {
	g := NewWithT(t)

	cfg := createCfg(t, time.Second, false)
	cfg.EXPECT().DefaultSessionTimeout().Return(time.Second).Once()
	cfg.EXPECT().MaxSessionTimeout().Return(50 * time.Millisecond).Once()

	ss := mocks.NewSessionStorage(t)
	now := func() time.Time { return time.UnixMilli(123) }

	br1 := mocks.NewBrowser(t)
	caps := mocks.NewCapabilities(t)
	caps.EXPECT().GetTimeout().Return(0).Once()
	s1 := session.NewSession("12345", "", br1, caps, nil, time.Time{}, nil, nil)
	s1.SetLastUsed(time.UnixMilli(70))

	ss.EXPECT().List(models.WebdriverProtocol).Return([]*session.Session{s1}).Once()
	ss.EXPECT().List(models.WebdriverProtocol).Return([]*session.Session{})
	ch := make(chan struct{})
	ss.EXPECT().Delete(models.WebdriverProtocol, "12345").
		RunAndReturn(func(protocol models.BrowserProtocol, id string) bool {
			close(ch)
			return true
		}).Once()
	br1.EXPECT().Close(context.Background(), true).Once()
	svc := wdsession.NewWDSessionServiceImpl(nil, ss, nil, cfg, now, 10*time.Millisecond, zaptest.NewLogger(t))

	g.Eventually(ch).Should(BeClosed())

	err := svc.Shutdown(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
}

func createCfg(t *testing.T, timeout time.Duration, proxyDelete bool) *mocks.WDSessionConfig {
	cfg := mocks.NewWDSessionConfig(t)
	cfg.EXPECT().CreateTimeout().Return(timeout)
	cfg.EXPECT().ProxyDelete().Return(proxyDelete)
	return cfg
}

func createSession(
	t *testing.T,
	g *WithT,
	svc *wdsession.WDSessionService,
	ss *mocks.SessionStorage,
	mgr *mocks.BrowserManager,
	client *mocks.HTTPClient,
	platform, browserName, version, driverUrl, sessId, host string,
) (*session.Session, error) {
	ss.EXPECT().IsShutdown().Return(false).Once()
	ss.EXPECT().Add(models.WebdriverProtocol, mock.Anything).Return(nil).Once()

	caps := mocks.NewCapabilities(t)
	caps.EXPECT().GetPlatform().Return(platform)
	caps.EXPECT().GetName().Return(browserName)
	caps.EXPECT().GetVersion().Return(version)
	caps.EXPECT().GetRawCapabilities().Return([]byte("{}"))

	u, err := url.Parse(driverUrl)
	g.Expect(err).ToNot(HaveOccurred())

	br := mocks.NewBrowser(t)
	mgr.EXPECT().Allocate(mock.Anything, models.WebdriverProtocol, caps).Return(br, nil).Once()
	br.EXPECT().GetURL().Return(u)
	br.EXPECT().GetHost().Return(host)

	client.EXPECT().
		Do(mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))}, nil).
		Once()

	client.EXPECT().
		Do(mock.Anything).
		Return(
			&http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(`{"value":{"sessionId":%q}}`, sessId)),
				),
			},
			nil,
		).Once()

	sess, err := svc.CreateSession(context.TODO(), caps)
	return sess, err
}
