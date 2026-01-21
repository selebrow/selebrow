package handlers

import (
	"net/http"
	"net/http/httputil"
	"time"

	"go.uber.org/zap"
)

func (p *ProxyHandler) handleHTTP(w http.ResponseWriter, r *http.Request, start time.Time, logger *zap.SugaredLogger) {
	proxy := httputil.ReverseProxy{
		Transport: p.transport,
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			proxyRequest.Out.URL = r.URL
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			httpError(w, http.StatusBadGateway, err, start, logger)
		},
		ModifyResponse: func(resp *http.Response) error {
			logger.With(
				"status", resp.StatusCode,
				"latency", time.Since(start),
			).Debug("request completed")
			return nil
		},
	}
	proxy.ServeHTTP(w, r)
}
