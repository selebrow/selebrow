package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/dto"
	evmodels "github.com/selebrow/selebrow/pkg/event/models"
	"github.com/selebrow/selebrow/pkg/models"
)

var caps1 = `
{
  "capabilities": {
    "alwaysMatch": {
      "browserName": "chrome",
      "browserVersion": "102.0",
      "acceptInsecureCerts": true,
      "selenoid:options": {}
    },
    "firstMatch": [ {} ]
  }
}`

func TestWDSessionController_CreateSession(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	sc := NewWDSessionController(srv, eb, now.Execute, zaptest.NewLogger(t))
	now.EXPECT().Execute().Return(time.UnixMilli(123)).Once()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sess", strings.NewReader(caps1))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	expResp := map[string]interface{}{
		"value": map[string]interface{}{
			"sessionId": "123",
		},
	}

	srv.EXPECT().CreateSession(ctx.Request().Context(), mock.Anything).RunAndReturn(
		func(_ context.Context, caps capabilities.Capabilities) (*session.Session, error) {
			g.Expect(caps.GetRawCapabilities()).To(Equal([]byte(caps1)))
			sess := session.NewSession("123", "", nil, caps, expResp, time.UnixMilli(456), nil, nil)
			return sess, nil
		}).Once()
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(Equal(evmodels.SessionRequested{
			Protocol:       "webdriver",
			BrowserName:    "chrome",
			BrowserVersion: "102.0",
			StartDuration:  333 * time.Millisecond,
			Error:          nil,
		}))
	}).Once()

	err := sc.CreateSession(ctx)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	var gotResp map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&gotResp)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(gotResp).To(Equal(expResp))

	srv.AssertExpectations(t)
	eb.AssertExpectations(t)
}

var caps2 = `{notAJsonAtAll}`

func TestWDSessionController_CreateSessionInvalidCaps(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	sc := NewWDSessionController(srv, eb, nil, zaptest.NewLogger(t))

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sess", strings.NewReader(caps2))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Protocol": Equal(models.BrowserProtocol("webdriver")),
			"Error":    HaveOccurred(),
		}))
	}).Once()

	err := sc.CreateSession(ctx)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(*models.W3CError).Value.Error).To(Equal(models.BadSessionParametersErr))

	srv.AssertExpectations(t)
	eb.AssertExpectations(t)
}

func TestWDSessionController_CreateSessionFailed(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	sc := NewWDSessionController(srv, eb, now.Execute, zaptest.NewLogger(t))
	now.EXPECT().Execute().Return(time.Time{})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sess", strings.NewReader(caps1))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	sessErr := errors.New("test session failed")
	srv.EXPECT().CreateSession(ctx.Request().Context(), mock.Anything).Return(nil, sessErr).Once()
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Protocol":       Equal(models.BrowserProtocol("webdriver")),
			"BrowserName":    Equal("chrome"),
			"BrowserVersion": Equal("102.0"),
			"Error":          MatchError("test session failed"),
		}))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes.Error.(models.ErrorWithCode).Code()).
			To(Equal(http.StatusInternalServerError))
	}).Once()

	err := sc.CreateSession(ctx)

	g.Expect(err).To(MatchError(sessErr))
	g.Expect(err.(*models.W3CError).Code()).To(Equal(http.StatusInternalServerError))
	g.Expect(err.(*models.W3CError).Value.Error).To(Equal("session not created"))
	g.Expect(err.(*models.W3CError).Value.Message).To(Equal(sessErr.Error()))

	srv.AssertExpectations(t)
	eb.AssertExpectations(t)
}

func TestWDSessionController_CreateSessionCancelled(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	sc := NewWDSessionController(srv, eb, now.Execute, zaptest.NewLogger(t))
	now.EXPECT().Execute().Return(time.Time{})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sess", strings.NewReader(caps1))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	sessErr := errors.Wrap(context.Canceled, "error")
	srv.EXPECT().CreateSession(ctx.Request().Context(), mock.Anything).Return(nil, sessErr).Once()
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Protocol":       Equal(models.BrowserProtocol("webdriver")),
			"BrowserName":    Equal("chrome"),
			"BrowserVersion": Equal("102.0"),
			"Error":          MatchError(sessErr),
		}))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes.Error.(models.ErrorWithCode).Code()).To(Equal(499))
	}).Once()

	err := sc.CreateSession(ctx)

	g.Expect(err).To(MatchError(sessErr))
	g.Expect(err.(*models.W3CError).Code()).To(Equal(499))
	g.Expect(err.(*models.W3CError).Value.Error).To(Equal("session not created"))
	g.Expect(err.(*models.W3CError).Value.Message).To(Equal(sessErr.Error()))

	srv.AssertExpectations(t)
	eb.AssertExpectations(t)
}

func TestWDSessionController_CreateSessionPanic(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	sc := NewWDSessionController(srv, eb, now.Execute, zaptest.NewLogger(t))
	now.EXPECT().Execute().Return(time.Time{})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sess", strings.NewReader(caps1))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	srv.EXPECT().CreateSession(ctx.Request().Context(), mock.Anything).Run(func(_ context.Context, _ capabilities.Capabilities) {
		panic("test")
	}).Once()
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Protocol":       Equal(models.BrowserProtocol("webdriver")),
			"BrowserName":    Equal("chrome"),
			"BrowserVersion": Equal("102.0"),
			"Error":          MatchError("panic: test"),
		}))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes.Error.(models.ErrorWithCode).Code()).
			To(Equal(http.StatusInternalServerError))
	}).Once()

	g.Expect(func() {
		_ = sc.CreateSession(ctx)
	}).To(PanicWith("test"))

	srv.AssertExpectations(t)
	eb.AssertExpectations(t)
}

func TestWDSessionController_ValidateSession(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	sc := NewWDSessionController(srv, nil, nil, zaptest.NewLogger(t))

	s := &session.Session{}
	srv.EXPECT().FindSession("s1").Return(s, nil).Once()
	ctx, rec := getSessionContext(router.SessRoute("/sess/:%s"), "/sess/s1", "s1")
	err := sc.ValidateSession(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	g.Expect(ctx.Get(SessionKey)).To(BeIdenticalTo(s))
	srv.AssertExpectations(t)
}

func TestWDSessionController_ValidateSessionNotFound(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	sc := NewWDSessionController(srv, nil, nil, zaptest.NewLogger(t))

	srv.EXPECT().FindSession("s2").Return(nil, errors.New("test session not found")).Once()
	ctx, _ := getSessionContext(router.SessRoute("/sess/:%s"), "/sess/s1", "s2")
	err := sc.ValidateSession(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(*models.W3CError).Code()).To(Equal(http.StatusNotFound))
	g.Expect(err.(*models.W3CError).Value.StackTrace).To(MatchRegexp(".*test session not found.*"))

	srv.AssertExpectations(t)
}

func TestWDSessionController_DeleteSession(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	caps := new(mocks.Capabilities)
	now := new(mocks.NowFunc)
	sc := NewWDSessionController(srv, eb, now.Execute, zaptest.NewLogger(t))
	now.EXPECT().Execute().Return(time.UnixMilli(333)).Once()

	s := session.NewSession("", "", nil, caps, nil, time.UnixMilli(111), nil, nil)
	caps.EXPECT().GetName().Return("Test")
	caps.EXPECT().GetVersion().Return("dev")
	srv.EXPECT().DeleteSession(s).Once()
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.(*evmodels.Event[evmodels.SessionReleased]).Attributes).To(Equal(evmodels.SessionReleased{
			Protocol:        "webdriver",
			BrowserName:     "Test",
			BrowserVersion:  "dev",
			SessionDuration: 222 * time.Millisecond,
		}))
	}).Once()

	ctx, rec := getSessionContext(router.SessRoute("/sess/:%s"), "/sess/s1", "s1")
	ctx.Set(SessionKey, s)

	err := sc.DeleteSession(ctx)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	srv.AssertExpectations(t)
	eb.AssertExpectations(t)
}

func TestWDSessionController_Status(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	sc := NewWDSessionController(srv, nil, nil, zaptest.NewLogger(t))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/status", strings.NewReader(""))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	expResp := dto.Status{
		Total: 3,
		Sessions: map[string][]dto.SessionStatus{
			"LINUX": {
				{ID: "s1", URL: "http://host1:123"},
				{ID: "s2", URL: "http://host2:123"},
			},
			"WIN3.1": {
				{ID: "s3", URL: "http://host3:123"},
			},
		},
	}
	srv.EXPECT().ListSessions().Return([]*session.Session{
		session.NewSession("s1", "LINUX", getWebdriverMock(g, "http://host1:123", "hst1:111"), nil, nil, time.Now(), nil, nil),
		session.NewSession("s2", "LINUX", getWebdriverMock(g, "http://host2:123", "hst2:222"), nil, nil, time.Now(), nil, nil),
		session.NewSession("s3", "WIN3.1", getWebdriverMock(g, "http://host3:123", "hst3:333"), nil, nil, time.Now(), nil, nil),
	}).Once()
	err := sc.Status(ctx)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	var gotResp dto.Status
	err = json.NewDecoder(rec.Body).Decode(&gotResp)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(gotResp).To(Equal(expResp))

	srv.AssertExpectations(t)
}

func getWebdriverMock(g *WithT, wdUrl string, host string) *mocks.Browser {
	u, err := url.Parse(wdUrl)
	g.Expect(err).ToNot(HaveOccurred())

	wd := new(mocks.Browser)
	wd.EXPECT().GetURL().Return(u)
	wd.EXPECT().GetHost().Return(host)
	return wd
}

func getSessionContext(route, path string, sess string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, path, strings.NewReader(""))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames(router.SessionParam)
	c.SetParamValues(sess)
	c.SetPath(route)
	return c, rec
}
