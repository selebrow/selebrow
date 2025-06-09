package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/selebrow/selebrow/internal/common/ws"
	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/pkg/models"
)

const (
	ProxyURLKey  = "proxyURL"
	ProxyHostKey = "proxyHost"
)

var proxyRewriteRules = map[string]string{"/se/file": "/file"}

type ProxyController struct {
	rules     map[string]string
	transport http.RoundTripper
	wsproxy   ws.WSProxy
	l         *zap.SugaredLogger
}

func NewProxyController(transport http.RoundTripper, wsproxy ws.WSProxy, l *zap.Logger) *ProxyController {
	return &ProxyController{
		rules:     proxyRewriteRules,
		transport: transport,
		wsproxy:   wsproxy,
		l:         l.Sugar(),
	}
}

func (p *ProxyController) SetProxyURL(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		i := strings.Index(c.Path(), fmt.Sprintf("/:%s", router.SessionParam))
		if i < 0 {
			p.l.Panicf("middleware applied to the wrong route: %s", c.Path())
		}

		id := c.Param(router.SessionParam)
		sess, _ := c.Get(SessionKey).(*session.Session)

		u := sess.Browser().GetURL()
		u.Path = path.Join(
			u.Path,
			strings.TrimPrefix(c.Request().URL.Path[0:i+1], router.WDHUBPath),
			id,
			c.Request().URL.Path[i+len(id)+1:],
		)
		u.RawQuery = c.Request().URL.RawQuery
		u.Fragment = c.Request().URL.Fragment

		c.Set(ProxyURLKey, u)
		c.Set(ProxyHostKey, sess.Browser().GetHost())
		return next(c)
	}
}

func (p *ProxyController) SetPortProxyURL(port models.ContainerPort) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			i := strings.Index(c.Path(), fmt.Sprintf("/:%s", router.SessionParam))
			if i < 0 {
				p.l.Panicf("middleware applied to the wrong route: %s", c.Path())
			}

			id := c.Param(router.SessionParam)
			sess, _ := c.Get(SessionKey).(*session.Session)

			host := sess.Browser().GetHostPort(port)
			if host == "" {
				return models.NewServiceUnavailableError(errors.Errorf("port %v is not supported or not enabled", port))
			}

			u := &url.URL{
				Scheme:   "http",
				Host:     host,
				Path:     c.Request().URL.Path[i+len(id)+1:],
				RawQuery: c.Request().URL.RawQuery,
				Fragment: c.Request().URL.Fragment,
			}

			c.Set(ProxyURLKey, u)
			c.Set(ProxyHostKey, u.Host)
			return next(c)
		}
	}
}

func (p *ProxyController) RewriteProxyUrl(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if len(p.rules) > 0 {
			proxyURL, _ := c.Get(ProxyURLKey).(*url.URL)
			for k, v := range p.rules {
				if strings.HasSuffix(proxyURL.Path, k) {
					proxyURL.Path = strings.TrimSuffix(proxyURL.Path, k) + v
					break
				}
			}
		}

		return next(c)
	}
}

func (p *ProxyController) Proxy(c echo.Context) error {
	proxyURL, _ := c.Get(ProxyURLKey).(*url.URL)
	proxyHost, _ := c.Get(ProxyHostKey).(string)

	(&httputil.ReverseProxy{
		Transport: p.transport,
		Director: func(r *http.Request) {
			r.Host = proxyHost
			r.URL = proxyURL
		},
		ErrorHandler: p.defaultErrorHandler(c.RealIP()),
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func (p *ProxyController) VNCProxy(c echo.Context) error {
	proxyURL, _ := c.Get(ProxyURLKey).(*url.URL)
	p.wsproxy.Handler(proxyURL.Host).ServeHTTP(c.Response(), c.Request())
	return nil
}

func (p *ProxyController) defaultErrorHandler(remote string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		p.l.With(zap.Error(err)).Errorf("proxy error %s->%v", remote, r.URL)
		w.WriteHeader(http.StatusBadGateway)
		resp := models.NewW3CErr(http.StatusBadGateway, "proxy error", err)
		respErr := json.NewEncoder(w).Encode(resp)
		if respErr != nil {
			p.l.Errorw("write error", zap.Error(respErr))
		}
	}
}
