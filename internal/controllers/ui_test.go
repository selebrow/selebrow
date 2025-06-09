package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/models"
)

type quotaAuthorizerQueueMock struct {
	mocks.QuotaAuthorizer
	mocks.QuotaQueue
}

func TestUIController_Index(t *testing.T) {
	g := NewWithT(t)
	r := new(mocks.Renderer)
	c, rec := getUIContext("/ui", r)

	wdSvc := new(mocks.SessionService)
	pwSvc := new(mocks.SessionService)
	qa := new(quotaAuthorizerQueueMock)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.WebdriverProtocol:  wdSvc,
		models.PlaywrightProtocol: pwSvc,
	}, qa, "", "")

	qa.QuotaAuthorizer.EXPECT().Enabled().Return(true).Once()
	qa.QuotaAuthorizer.EXPECT().Allocated().Return(123).Once()
	qa.QuotaAuthorizer.EXPECT().Limit().Return(456).Once()

	qa.QuotaQueue.EXPECT().QueueLimit().Return(5).Twice()
	qa.QuotaQueue.EXPECT().QueueSize().Return(3).Once()

	wdSvc.EXPECT().ListSessions().Return([]*session.Session{{}, {}}).Once()
	pwSvc.EXPECT().ListSessions().Return([]*session.Session{{}, {}, {}}).Once()

	expData := &indexData{
		WDLink:  "/ui/wd",
		PWLink:  "/ui/pw",
		WDCount: 2,
		PWCount: 3,
		Quota: &quotaData{
			Allocated: 123,
			Limit:     456,
			Queue: &queueData{
				Limit: 5,
				Size:  3,
			},
		},
	}
	r.EXPECT().Render(mock.Anything, "index.tmpl", expData, c).Return(nil).Once()

	err := ui.Index(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	r.AssertExpectations(t)
	qa.QuotaAuthorizer.AssertExpectations(t)
	wdSvc.AssertExpectations(t)
	pwSvc.AssertExpectations(t)
}

func TestUIController_WDSessions(t *testing.T) {
	g := NewWithT(t)
	r := new(mocks.Renderer)
	c, rec := getUIContext("/ui/wd", r)

	wdSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.WebdriverProtocol: wdSvc,
	}, nil, "", "")

	testSessions := createTestSessions()
	wdSvc.EXPECT().ListSessions().Return(testSessions).Once()

	expData := &sessionData{
		Root:     "/ui",
		Protocol: "webdriver",
		Sessions: []sessionItem{
			{
				ID:             "1111",
				CreatedAt:      "1970-01-01 00:00:12",
				Browser:        "ie",
				BrowserVersion: "3.0",
				Name:           "test1",
				VNC:            false,
				VNCLink:        "/ui/wd/1111/vnc",
				ResetLink:      "/ui/wd/1111/reset",
			}, {
				ID:             "2222",
				CreatedAt:      "1970-01-01 00:00:13",
				Browser:        "netscape",
				BrowserVersion: "6.0",
				Name:           "test2",
				VNC:            true,
				VNCLink:        "/ui/wd/2222/vnc",
				ResetLink:      "/ui/wd/2222/reset",
			},
		},
	}
	r.EXPECT().Render(mock.Anything, "sessions.tmpl", expData, c).Return(nil).Once()

	err := ui.WDSessions(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	r.AssertExpectations(t)
	wdSvc.AssertExpectations(t)
}

func TestUIController_PWSessions(t *testing.T) {
	g := NewWithT(t)
	r := new(mocks.Renderer)
	c, rec := getUIContext("/ui/wd", r)

	pwSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.PlaywrightProtocol: pwSvc,
	}, nil, "", "")

	testSessions := createTestSessions()
	pwSvc.EXPECT().ListSessions().Return(testSessions).Once()

	expData := &sessionData{
		Root:     "/ui",
		Protocol: "playwright",
		Sessions: []sessionItem{
			{
				ID:             "1111",
				CreatedAt:      "1970-01-01 00:00:12",
				Browser:        "ie",
				BrowserVersion: "3.0",
				Name:           "test1",
				VNC:            false,
				VNCLink:        "/ui/pw/1111/vnc",
				ResetLink:      "/ui/pw/1111/reset",
			}, {
				ID:             "2222",
				CreatedAt:      "1970-01-01 00:00:13",
				Browser:        "netscape",
				BrowserVersion: "6.0",
				Name:           "test2",
				VNC:            true,
				VNCLink:        "/ui/pw/2222/vnc",
				ResetLink:      "/ui/pw/2222/reset",
			},
		},
	}
	r.EXPECT().Render(mock.Anything, "sessions.tmpl", expData, c).Return(nil).Once()

	err := ui.PWSessions(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	r.AssertExpectations(t)
	pwSvc.AssertExpectations(t)
}

func TestUIController_WDVNC(t *testing.T) {
	g := NewWithT(t)
	r := new(mocks.Renderer)
	c, rec := getUIContext("/ui/wd/123/vnc", r)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	wdSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.WebdriverProtocol: wdSvc,
	}, nil, "", "qwerty")

	sess := createTestSession("123", true)
	wdSvc.EXPECT().FindSession("123").Return(sess, nil).Once()

	expData := &vncData{ID: "123", URL: "ws://example.com/vnc/123", Password: "qwerty"}
	r.EXPECT().Render(mock.Anything, "vnc.tmpl", expData, c).Return(nil).Once()

	err := ui.WDVNC(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	r.AssertExpectations(t)
	wdSvc.AssertExpectations(t)
}

func TestUIController_WDVNC_NotEnabled(t *testing.T) {
	g := NewWithT(t)
	c, _ := getUIContext("/ui/wd/321/vnc", nil)
	c.SetParamNames("sess")
	c.SetParamValues("321")

	wdSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.WebdriverProtocol: wdSvc,
	}, nil, "", "qwerty")

	sess := createTestSession("321", false)
	wdSvc.EXPECT().FindSession("321").Return(sess, nil).Once()

	err := ui.WDVNC(c)
	g.Expect(err).To(MatchError(MatchRegexp("not enabled")))
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusBadRequest))

	wdSvc.AssertExpectations(t)
}

func TestUIController_WDVNC_NotFound(t *testing.T) {
	g := NewWithT(t)
	c, _ := getUIContext("/ui/wd/123/vnc", nil)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	wdSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.WebdriverProtocol: wdSvc,
	}, nil, "", "qwerty")

	wdSvc.EXPECT().FindSession("123").Return(nil, errors.New("not found")).Once()

	err := ui.WDVNC(c)
	g.Expect(err).To(MatchError("not found"))
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusNotFound))

	wdSvc.AssertExpectations(t)
}

func TestUIController_PWVNC(t *testing.T) {
	g := NewWithT(t)
	r := new(mocks.Renderer)
	c, rec := getUIContext("/ui/wd/123/vnc", r)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	pwSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.PlaywrightProtocol: pwSvc,
	}, nil, "", "qwerty")

	sess := createTestSession("123", true)
	pwSvc.EXPECT().FindSession("123").Return(sess, nil).Once()

	expData := &vncData{ID: "123", URL: "ws://example.com/pw/vnc/123", Password: "qwerty"}
	r.EXPECT().Render(mock.Anything, "vnc.tmpl", expData, c).Return(nil).Once()

	err := ui.PWVNC(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	r.AssertExpectations(t)
	pwSvc.AssertExpectations(t)
}

func TestUIController_PWVNC_NotEnabled(t *testing.T) {
	g := NewWithT(t)
	c, _ := getUIContext("/ui/wd/123/vnc", nil)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	pwSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.PlaywrightProtocol: pwSvc,
	}, nil, "", "qwerty")

	sess := createTestSession("123", false)
	pwSvc.EXPECT().FindSession("123").Return(sess, nil).Once()

	err := ui.PWVNC(c)
	g.Expect(err).To(MatchError(MatchRegexp("not enabled")))
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusBadRequest))

	pwSvc.AssertExpectations(t)
}

func TestUIController_PWVNC_NotFound(t *testing.T) {
	g := NewWithT(t)
	c, _ := getUIContext("/ui/pw/123/vnc", nil)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	pwSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.PlaywrightProtocol: pwSvc,
	}, nil, "", "qwerty")

	pwSvc.EXPECT().FindSession("123").Return(nil, errors.New("not found")).Once()

	err := ui.PWVNC(c)
	g.Expect(err).To(MatchError("not found"))
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusNotFound))

	pwSvc.AssertExpectations(t)
}

func TestUIController_WDReset(t *testing.T) {
	g := NewWithT(t)
	c, rec := getUIContext("/ui/wd/123/reset", nil)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	wdSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.WebdriverProtocol: wdSvc,
	}, nil, "", "")

	sess := createTestSession("123", true)
	wdSvc.EXPECT().FindSession("123").Return(sess, nil).Once()
	wdSvc.EXPECT().DeleteSession(sess).Once()

	err := ui.WDReset(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusSeeOther))
	g.Expect(rec).To(HaveHTTPHeaderWithValue("Location", "/ui/wd"))

	wdSvc.AssertExpectations(t)
}

func TestUIController_WDReset_NotFound(t *testing.T) {
	g := NewWithT(t)
	c, _ := getUIContext("/ui/wd/123/reset", nil)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	wdSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.WebdriverProtocol: wdSvc,
	}, nil, "", "qwerty")

	wdSvc.EXPECT().FindSession("123").Return(nil, errors.New("not found")).Once()

	err := ui.WDReset(c)
	g.Expect(err).To(MatchError("not found"))
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusNotFound))

	wdSvc.AssertExpectations(t)
}

func TestUIController_PWReset(t *testing.T) {
	g := NewWithT(t)
	c, rec := getUIContext("/ui/pw/123/reset", nil)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	pwSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.PlaywrightProtocol: pwSvc,
	}, nil, "", "")

	sess := createTestSession("123", true)
	pwSvc.EXPECT().FindSession("123").Return(sess, nil).Once()
	pwSvc.EXPECT().DeleteSession(sess).Once()

	err := ui.PWReset(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusSeeOther))
	g.Expect(rec).To(HaveHTTPHeaderWithValue("Location", "/ui/pw"))

	pwSvc.AssertExpectations(t)
}

func TestUIController_PWReset_NotFound(t *testing.T) {
	g := NewWithT(t)
	c, _ := getUIContext("/ui/pw/123/reset", nil)
	c.SetParamNames("sess")
	c.SetParamValues("123")

	pwSvc := new(mocks.SessionService)

	ui := NewUIController(map[models.BrowserProtocol]session.SessionService{
		models.PlaywrightProtocol: pwSvc,
	}, nil, "", "qwerty")

	pwSvc.EXPECT().FindSession("123").Return(nil, errors.New("not found")).Once()

	err := ui.PWReset(c)
	g.Expect(err).To(MatchError("not found"))
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusNotFound))

	pwSvc.AssertExpectations(t)
}

func TestUIController_URL(t *testing.T) {
	tests := []struct {
		name   string
		listen string
		want   string
	}{
		{
			name:   "numeric port only",
			listen: ":1234",
			want:   "http://localhost:1234/ui",
		},
		{
			name:   "named port only",
			listen: ":http",
			want:   "http://localhost:80/ui",
		},
		{
			name:   "specific ip",
			listen: "1.2.3.4:321",
			want:   "http://1.2.3.4:321/ui",
		},
		{
			name:   "ipv4 bind all",
			listen: "0.0.0.0:321",
			want:   "http://localhost:321/ui",
		},
		{
			name:   "ipv6 bind all",
			listen: "[::]:321",
			want:   "http://localhost:321/ui",
		},
		{
			name:   "malformed listen string",
			listen: "qqq",
			want:   "qqq",
		},
		{
			name:   "unresolvable port",
			listen: "0.0.0.0:qqqq",
			want:   "0.0.0.0:qqqq",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			u := NewUIController(nil, nil, tt.listen, "")
			got := u.URL()
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func createTestSession(id string, vncEnabled bool) *session.Session {
	caps := new(mocks.Capabilities)
	caps.EXPECT().IsVNCEnabled().Return(vncEnabled).Once()
	sess := session.NewSession(id, "", nil, caps, nil, time.Time{}, nil, nil)
	return sess
}

func createTestSessions() []*session.Session {
	caps1 := new(mocks.Capabilities)
	caps1.EXPECT().GetName().Return("ie").Once()
	caps1.EXPECT().GetVersion().Return("3.0").Once()
	caps1.EXPECT().GetTestName().Return("test1").Once()
	caps1.EXPECT().IsVNCEnabled().Return(false).Once()

	s1 := session.NewSession("1111", "", nil, caps1, nil, time.UnixMilli(12345).UTC(), nil, nil)

	caps2 := new(mocks.Capabilities)
	caps2.EXPECT().GetName().Return("netscape").Once()
	caps2.EXPECT().GetVersion().Return("6.0").Once()
	caps2.EXPECT().GetTestName().Return("test2").Once()
	caps2.EXPECT().IsVNCEnabled().Return(true).Once()
	s2 := session.NewSession("2222", "", nil, caps2, nil, time.UnixMilli(13345).UTC(), nil, nil)
	testSessions := []*session.Session{s1, s2}
	return testSessions
}

func getUIContext(target string, r *mocks.Renderer) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Renderer = r

	req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}
