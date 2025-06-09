package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/posener/wstest"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	ws "golang.org/x/net/websocket"

	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/models"
)

func TestWDProxyController_SetProxyUrl(t *testing.T) {
	g := NewGomegaWithT(t)
	cntr := NewProxyController(nil, nil, zaptest.NewLogger(t))

	u := "http://host:5566/wdhub"
	s := session.NewSession("12345", "DARWIN", getWebdriverMock(g, u, "hst:321"), nil, nil, time.Now(), nil, nil)
	path := "/session/12345/tail"
	ctx, rec := getSessionContext(router.SessRoute("/session/:%s"), path, "12345")
	ctx.Set(SessionKey, s)

	err := cntr.SetProxyURL(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec.Code).Should(Equal(http.StatusOK))

	got := ctx.Get(ProxyURLKey).(*url.URL)
	g.Expect(got.String()).Should(Equal(u + path))

	gotHost := ctx.Get(ProxyHostKey).(string)
	g.Expect(gotHost).Should(Equal("hst:321"))
}

func TestWDProxyController_SetPortProxyUrl(t *testing.T) {
	g := NewGomegaWithT(t)
	cntr := NewProxyController(nil, nil, zaptest.NewLogger(t))

	hp := "fs1:8088"
	s := session.NewSession("1122", "OPENBSD", getWebdriverPortMock(models.FileserverPort, hp), nil, nil, time.Now(), nil, nil)
	path := "/session/1122/tail"
	ctx, rec := getSessionContext(router.SessRoute("/session/:%s"), path, "1122")
	ctx.Set(SessionKey, s)

	err := cntr.SetPortProxyURL(models.FileserverPort)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec.Code).Should(Equal(http.StatusOK))

	got := ctx.Get(ProxyURLKey).(*url.URL)
	exp := &url.URL{
		Scheme: "http",
		Host:   hp,
		Path:   "/tail",
	}
	g.Expect(got).Should(Equal(exp))

	gotHost := ctx.Get(ProxyHostKey).(string)
	g.Expect(gotHost).Should(Equal(hp))
}

func TestWDProxyController_SetPortProxyUrlUnavailable(t *testing.T) {
	g := NewGomegaWithT(t)
	cntr := NewProxyController(nil, nil, zaptest.NewLogger(t))

	s := session.NewSession("1122", "OPENBSD", getWebdriverPortMock(models.ClipboardPort, ""), nil, nil, time.Now(), nil, nil)
	path := "/session/1122/tail"
	ctx, _ := getSessionContext(router.SessRoute("/session/:%s"), path, "1122")
	ctx.Set(SessionKey, s)

	err := cntr.SetPortProxyURL(models.ClipboardPort)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(models.ErrorWithCode).Code()).Should(Equal(http.StatusServiceUnavailable))
}

func TestWDProxyController_ProxyURLRewrite(t *testing.T) {
	g := NewGomegaWithT(t)
	cntr := ProxyController{rules: map[string]string{"/rewrite": "/done"}}
	route := "http://host.tld:1234"

	path1 := "/session/12345/rewrite"
	url1, err := url.Parse(route + path1)
	g.Expect(err).ToNot(HaveOccurred())
	ctx, rec := getSessionContext(router.SessRoute("/session/:%s"), path1, "")
	ctx.Set(ProxyURLKey, url1)

	err = cntr.RewriteProxyUrl(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec.Code).Should(Equal(http.StatusOK))

	got := ctx.Get(ProxyURLKey).(*url.URL)
	exp, err := url.Parse(route + "/session/12345/done")
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(got).Should(Equal(exp))
}

func TestWDProxyController_ProxyURLRewriteSeDownload(t *testing.T) {
	g := NewGomegaWithT(t)
	cntr := NewProxyController(nil, nil, zaptest.NewLogger(t))
	route := "http://host.tld:1234"

	path1 := "/session/12345/se/file"
	url1, err := url.Parse(route + path1)
	g.Expect(err).ToNot(HaveOccurred())
	ctx, rec := getSessionContext(router.SessRoute("/session/:%s"), path1, "")
	ctx.Set(ProxyURLKey, url1)

	err = cntr.RewriteProxyUrl(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec.Code).Should(Equal(http.StatusOK))

	got := ctx.Get(ProxyURLKey).(*url.URL)
	exp, err := url.Parse(route + "/session/12345/file")
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(got).Should(Equal(exp))
}

func TestWDProxyController_Proxy(t *testing.T) {
	g := NewGomegaWithT(t)
	rt := new(mocks.RoundTripper)
	cntr := NewProxyController(rt, nil, zaptest.NewLogger(t))

	u, err := url.Parse("http://host:3215/wd/hub/session/s1/some-request")
	g.Expect(err).ToNot(HaveOccurred())
	path := "/session/s1/some-request"
	ctx, rec := getSessionContext(router.SessRoute("/session/:%s"), path, "s1")
	ctx.Set(ProxyURLKey, u)
	ctx.Set(ProxyHostKey, "hst:123")

	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(``)),
	}

	rt.EXPECT().RoundTrip(mock.Anything).Run(func(req *http.Request) {
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL).To(Equal(u))
		g.Expect(req.Host).To(Equal("hst:123"))
	}).Return(mockResp, nil)

	g.Expect(cntr.Proxy(ctx)).To(Succeed())
	g.Expect(rec.Code).Should(Equal(http.StatusOK))
}

func TestWDProxyController_ProxyError(t *testing.T) {
	g := NewGomegaWithT(t)
	rt := new(mocks.RoundTripper)
	cntr := NewProxyController(rt, nil, zaptest.NewLogger(t))

	u, err := url.Parse("http://host:3215/wd/hub/session/s1/some-request")
	g.Expect(err).ToNot(HaveOccurred())
	path := "/session/s1/some-request"
	ctx, rec := getSessionContext(router.SessRoute("/session/:%s"), path, "s1")
	ctx.Set(ProxyURLKey, u)
	ctx.Set(ProxyHostKey, "hst:123")

	var nilResp *http.Response
	rt.EXPECT().RoundTrip(mock.Anything).Return(nilResp, errors.New("test proxy error"))

	g.Expect(cntr.Proxy(ctx)).To(Succeed())
	g.Expect(rec.Code).Should(Equal(http.StatusBadGateway))

	var w3cErr models.W3CError
	err = json.NewDecoder(rec.Body).Decode(&w3cErr)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(w3cErr.Value.StackTrace).To(MatchRegexp(".*test proxy error.*"))
}

func TestWDProxyController_VNCProxy(t *testing.T) {
	g := NewGomegaWithT(t)
	p := new(mocks.WSProxy)
	cntr := NewProxyController(nil, p, zaptest.NewLogger(t))

	u, err := url.Parse("http://vnchost:4321/ignored")
	g.Expect(err).ToNot(HaveOccurred())

	e := echo.New()
	e.GET(router.SessRoute("/session/vnc/:%s"), cntr.VNCProxy, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ProxyURLKey, u)
			return next(c)
		}
	})
	d := wstest.NewDialer(e)

	hdr := make(http.Header)
	hdr.Set("Origin", "http://testclient")

	ch := make(chan struct{})
	p.EXPECT().Handler("vnchost:4321").Return(func(conn *ws.Conn) {
		close(ch)
	})

	wc, r, err := d.Dial("ws://ignored/session/vnc/s1", hdr)
	g.Expect(err).ToNot(HaveOccurred())
	defer wc.Close()
	defer r.Body.Close()

	g.Expect(r).To(HaveHTTPStatus("101 Switching Protocols"))
	p.AssertExpectations(t)
	g.Eventually(ch).Should(BeClosed())
}

func getWebdriverPortMock(port models.ContainerPort, hostport string) *mocks.Browser {
	wd := new(mocks.Browser)
	wd.EXPECT().GetHostPort(port).Return(hostport)
	return wd
}
