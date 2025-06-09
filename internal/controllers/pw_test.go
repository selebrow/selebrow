package controllers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/capabilities"
	evmodels "github.com/selebrow/selebrow/pkg/event/models"
	"github.com/selebrow/selebrow/pkg/models"
)

func TestPWController_CreateSession_BadParameters(t *testing.T) {
	tests := []struct {
		name     string
		params   url.Values
		errMatch types.GomegaMatcher
	}{
		{
			name:     "Bad headless",
			params:   url.Values{"headless": []string{"aaa"}},
			errMatch: MatchError(MatchRegexp(`.*bad headless.*`)),
		},
		{
			name:     "Bad vnc",
			params:   url.Values{"vnc": []string{"111"}},
			errMatch: MatchError(MatchRegexp(`.*bad vnc.*`)),
		},
		{
			name:     "Bad resolution",
			params:   url.Values{"resolution": []string{"3x5"}},
			errMatch: MatchError(MatchRegexp(`.*incorrect resolution.*`)),
		},
		{
			name:     "Bad env",
			params:   url.Values{"env": []string{"qqqq"}},
			errMatch: MatchError(MatchRegexp(`.*malformed env param.*`)),
		},
		{
			name:     "Bad env name",
			params:   url.Values{"env": []string{"qqq/q=wwww"}},
			errMatch: MatchError(MatchRegexp(`.*invalid env name.*`)),
		},
		{
			name:     "Bad label",
			params:   url.Values{"label": []string{"qqqq<wwww"}},
			errMatch: MatchError(MatchRegexp(`.*bad label.*`)),
		},
		{
			name:     "Bad firefoxUserPref",
			params:   url.Values{"firefoxUserPref": []string{"qqqq"}},
			errMatch: MatchError(MatchRegexp(`.*bad firefoxUserPrefs.*`)),
		},
		{
			name:     "Bad launch options",
			params:   url.Values{"launch-options": []string{"qqqq"}},
			errMatch: MatchError(MatchRegexp(`.*malformed launch-options.*`)),
		},
		{
			name:     "Bad launch options firefoxUserPrefs (object)",
			params:   url.Values{"launch-options": []string{`{"firefoxUserPrefs": {"test": {"key": 1234}}}`}},
			errMatch: MatchError(MatchRegexp(`.*bad launch options.*invalid firefoxUserPref.*`)),
		},
		{
			name:     "Bad launch options firefoxUserPrefs (array)",
			params:   url.Values{"launch-options": []string{`{"firefoxUserPrefs": {"test": [1234]}}`}},
			errMatch: MatchError(MatchRegexp(`.*bad launch options.*invalid firefoxUserPref.*`)),
		},
		{
			name:     "Bad launch options firefoxUserPrefs (null)",
			params:   url.Values{"launch-options": []string{`{"firefoxUserPrefs": {"test": null}}`}},
			errMatch: MatchError(MatchRegexp(`.*bad launch options.*invalid firefoxUserPref.*`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			eb := new(mocks.EventBroker)
			now := new(mocks.NowFunc)
			cntr := NewPWController(nil, nil, eb, now.Execute, zaptest.NewLogger(t))
			ctx, _ := getPWContext("chrome", "def", "v1", tt.params)

			eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
				g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
				g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).
					To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Protocol":       Equal(models.BrowserProtocol("playwright")),
						"BrowserName":    Equal("chrome"),
						"BrowserVersion": Equal("v1"),
						"Error":          tt.errMatch,
					}))
			}).Once()

			err := cntr.CreateSession(ctx)
			g.Expect(err).To(tt.errMatch)
		})
	}
}

func TestPWController_CreateSession(t *testing.T) {
	g := NewWithT(t)
	rt := new(mocks.RoundTripper)
	s := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	cntr := NewPWController(s, rt, eb, now.Execute, zaptest.NewLogger(t))
	br := new(mocks.Browser)

	u, err := url.Parse("http://host:1234/qqq")
	g.Expect(err).ToNot(HaveOccurred())

	q := url.Values{
		"arg":             []string{"aaa", "bbb"},
		"headless":        []string{"false"},
		"resolution":      []string{"1x2x3"},
		"vnc":             []string{"false"}, // expected to be re-enabled by headless=false
		"env":             []string{"pw_var1=val1", "PW_VAR2=val2"},
		"label":           []string{"l1=v1", "l2=v2"},
		"link":            []string{"l1", "l2"},
		"host":            []string{"h1", "h2"},
		"network":         []string{"n1", "n2"},
		"channel":         []string{"test"},
		"firefoxUserPref": []string{"k1=true", "k2=123", "k3=false", "k4=abc"},
		"launch-options":  []string{`{"args": ["ccc"]}`},
	}
	ctx, rec := getPWContext("test", "custom", "v1", q)
	caps := &models.PWCapabilities{
		Flavor:           "custom",
		Browser:          "test",
		Version:          "v1",
		VNCEnabled:       true,
		ScreenResolution: "1x2x3",
		Env:              []string{"pw_var1=val1", "PW_VAR2=val2"},
		Links:            []string{"l1", "l2"},
		Hosts:            []string{"h1", "h2"},
		Networks:         []string{"n1", "n2"},
		Labels:           map[string]string{"l1": "v1", "l2": "v2"},
	}
	sess := createPWSession(br, caps, 122)
	s.EXPECT().CreateSession(context.Background(), caps).Return(sess, nil).Once()
	s.EXPECT().DeleteSession(sess).Once()
	br.EXPECT().GetURL().RunAndReturn(func() *url.URL {
		uCopy := *u
		return &uCopy
	})

	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`testdata`)),
	}

	rt.EXPECT().RoundTrip(mock.Anything).Run(func(req *http.Request) {
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL.String()).To(Equal(
			u.String() + "?arg=ccc&arg=aaa&arg=bbb&headless=false" +
				"&launch-options=%7B%22args%22%3A%5B%22ccc%22%2C%22aaa%22%2C%22bbb%22%5D%2C%22" +
				"headless%22%3Afalse%2C%22channel%22%3A%22test%22" +
				"%2C%22firefoxUserPrefs%22%3A%7B%22k1%22%3Atrue%2C%22k2%22%3A123%2C%22k3%22%3Afalse%2C%22k4%22%3A%22abc%22%7D%7D",
		))
		g.Expect(req.Host).To(Equal(u.Host))
	}).Return(mockResp, nil)

	now.EXPECT().Execute().Return(time.UnixMilli(111)).Once()
	now.EXPECT().Execute().Return(time.UnixMilli(144)).Once()

	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(Equal(evmodels.SessionRequested{
			Protocol:       "playwright",
			BrowserName:    "test",
			BrowserVersion: "v1",
			StartDuration:  11 * time.Millisecond,
			Error:          nil,
		}))
	}).Once()

	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.(*evmodels.Event[evmodels.SessionReleased]).Attributes).To(Equal(evmodels.SessionReleased{
			Protocol:        "playwright",
			BrowserName:     "test",
			BrowserVersion:  "v1",
			SessionDuration: 22 * time.Millisecond,
		}))
	}).Once()

	err = cntr.CreateSession(ctx)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(rec.Body.String()).To(Equal(`testdata`))
	br.AssertExpectations(t)
	s.AssertExpectations(t)
	eb.AssertExpectations(t)
	now.AssertExpectations(t)
}

func TestPWController_CreateSession_Error(t *testing.T) {
	g := NewWithT(t)
	s := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	cntr := NewPWController(s, nil, eb, now.Execute, zaptest.NewLogger(t))

	ctx, _ := getPWContext("test", "custom", "v2", nil)
	s.EXPECT().
		CreateSession(context.Background(), &models.PWCapabilities{Flavor: "custom", Browser: "test", Version: "v2"}).
		Return(nil, errors.New("test error")).
		Once()
	now.EXPECT().Execute().Return(time.UnixMilli(111)).Once()
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Protocol":       Equal(models.BrowserProtocol("playwright")),
			"BrowserName":    Equal("test"),
			"BrowserVersion": Equal("v2"),
			"Error":          MatchError("test error"),
		}))
	}).Once()

	err := cntr.CreateSession(ctx)

	g.Expect(err).To(MatchError(`test error`))

	s.AssertExpectations(t)
	eb.AssertExpectations(t)
	now.AssertExpectations(t)
}

func TestPWController_CreateSession_Cancelled(t *testing.T) {
	g := NewWithT(t)
	s := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	cntr := NewPWController(s, nil, eb, now.Execute, zaptest.NewLogger(t))

	ctx, _ := getPWContext("test", "custom", "v1", nil)
	expErr := errors.Wrap(context.Canceled, "error")
	s.EXPECT().
		CreateSession(context.Background(), &models.PWCapabilities{Flavor: "custom", Browser: "test", Version: "v1"}).
		Return(nil, expErr).
		Once()
	now.EXPECT().Execute().Return(time.UnixMilli(111)).Once()
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Protocol":       Equal(models.BrowserProtocol("playwright")),
			"BrowserName":    Equal("test"),
			"BrowserVersion": Equal("v1"),
			"Error":          MatchError(expErr),
		}))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes.Error.(models.ErrorWithCode).Code()).To(Equal(499))
	}).Once()

	err := cntr.CreateSession(ctx)

	g.Expect(err).To(MatchError(expErr))

	s.AssertExpectations(t)
	eb.AssertExpectations(t)
}

func TestPWController_CreateSession_Panic(t *testing.T) {
	g := NewWithT(t)
	s := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	cntr := NewPWController(s, nil, eb, now.Execute, zaptest.NewLogger(t))

	ctx, _ := getPWContext("test", "custom", "v1", nil)
	s.EXPECT().CreateSession(context.Background(), &models.PWCapabilities{Flavor: "custom", Browser: "test", Version: "v1"}).
		Run(func(_ context.Context, _ capabilities.Capabilities) {
			panic("test")
		}).Once()
	now.EXPECT().Execute().Return(time.UnixMilli(111)).Once()
	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Protocol":       Equal(models.BrowserProtocol("playwright")),
			"BrowserName":    Equal("test"),
			"BrowserVersion": Equal("v1"),
			"Error":          MatchError("panic: test"),
		}))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes.Error.(models.ErrorWithCode).Code()).
			To(Equal(http.StatusInternalServerError))
	}).Once()

	g.Expect(func() {
		_ = cntr.CreateSession(ctx)
	}).To(PanicWith("test"))
	s.AssertExpectations(t)
	eb.AssertExpectations(t)
	now.AssertExpectations(t)
}

func TestPWController_CreateSession_Proxy_Error(t *testing.T) {
	g := NewWithT(t)
	rt := new(mocks.RoundTripper)
	s := new(mocks.SessionService)
	eb := new(mocks.EventBroker)
	now := new(mocks.NowFunc)
	cntr := NewPWController(s, rt, eb, now.Execute, zaptest.NewLogger(t))
	br := new(mocks.Browser)

	u, err := url.Parse("http://host:1234/qqq")
	g.Expect(err).ToNot(HaveOccurred())

	ctx, rec := getPWContext("test", "custom", "v1", nil)
	caps := &models.PWCapabilities{Flavor: "custom", Browser: "test", Version: "v1"}
	sess := createPWSession(br, caps, 122)
	s.EXPECT().CreateSession(context.Background(), caps).Return(sess, nil).Once()
	s.EXPECT().DeleteSession(sess)
	br.EXPECT().GetURL().Return(u)

	var nilResp *http.Response
	rt.EXPECT().RoundTrip(mock.Anything).Run(func(req *http.Request) {
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL).To(Equal(u))
		g.Expect(req.Host).To(Equal(u.Host))
	}).Return(nilResp, errors.New("test error"))

	now.EXPECT().Execute().Return(time.UnixMilli(111)).Once()
	now.EXPECT().Execute().Return(time.UnixMilli(144)).Once()

	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.EventType()).To(Equal(evmodels.SessionRequestedEventType))
		g.Expect(e.(*evmodels.Event[evmodels.SessionRequested]).Attributes).To(Equal(evmodels.SessionRequested{
			Protocol:       "playwright",
			BrowserName:    "test",
			BrowserVersion: "v1",
			StartDuration:  11 * time.Millisecond,
			Error:          nil,
		}))
	}).Once()

	eb.EXPECT().Publish(mock.Anything).Run(func(e evmodels.IEvent) {
		g.Expect(e.(*evmodels.Event[evmodels.SessionReleased]).Attributes).To(Equal(evmodels.SessionReleased{
			Protocol:        "playwright",
			BrowserName:     "test",
			BrowserVersion:  "v1",
			SessionDuration: 22 * time.Millisecond,
		}))
	}).Once()

	g.Expect(cntr.CreateSession(ctx)).To(Succeed())

	g.Expect(rec.Code).To(Equal(http.StatusBadGateway))
	g.Expect(rec.Body.String()).To(MatchRegexp(`test error`))
	br.AssertExpectations(t)
	s.AssertExpectations(t)
	eb.AssertExpectations(t)
	now.AssertExpectations(t)
}

func TestPWController_ValidateSession(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	c := NewPWController(srv, nil, nil, nil, zaptest.NewLogger(t))

	s := &session.Session{}
	srv.EXPECT().FindSession("s1").Return(s, nil).Once()
	ctx, rec := getSessionContext(router.SessRoute("/sess/:%s"), "/sess/s1", "s1")
	err := c.ValidateSession(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	g.Expect(ctx.Get(SessionKey)).To(BeIdenticalTo(s))
	srv.AssertExpectations(t)
}

func TestPWController_ValidateSessionNotFound(t *testing.T) {
	g := NewWithT(t)
	srv := new(mocks.SessionService)
	c := NewPWController(srv, nil, nil, nil, zaptest.NewLogger(t))

	srv.EXPECT().FindSession("s2").Return(nil, errors.New("test session not found")).Once()
	ctx, _ := getSessionContext(router.SessRoute("/sess/:%s"), "/sess/s1", "s2")
	err := c.ValidateSession(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).To(MatchError("test session not found"))
	g.Expect(err.(*models.ErrorMessage).Code()).To(Equal(http.StatusNotFound))

	srv.AssertExpectations(t)
}

func getPWContext(name, flavor, version string, q url.Values) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ignored", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames(router.NameParam, router.VersionParam)
	c.SetParamValues(name, version)
	c.QueryParams().Set(router.FlavorQParam, flavor)
	for k, v := range q {
		c.QueryParams()[k] = v
	}
	return c, rec
}

func createPWSession(br *mocks.Browser, caps *models.PWCapabilities, ts int64) *session.Session {
	return session.NewSession(
		"12345",
		"cp/m",
		br,
		caps,
		nil,
		time.UnixMilli(ts),
		context.TODO(),
		nil,
	)
}
