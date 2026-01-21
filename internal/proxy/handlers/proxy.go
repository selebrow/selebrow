package handlers

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/proxy"
)

type ResponseWriterHijacker interface {
	http.ResponseWriter
	http.Hijacker
}

type ProxyHandler struct {
	transport    http.RoundTripper
	connDialer   proxy.ContextDialer
	proxyConnect bool
	logger       *zap.SugaredLogger
}

func NewProxyHandler(transport http.RoundTripper, connDialer proxy.ContextDialer, logger *zap.Logger) *ProxyHandler {
	return &ProxyHandler{
		transport:    transport,
		connDialer:   connDialer,
		proxyConnect: true,
		logger:       logger.Sugar(),
	}
}

func (p *ProxyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	start := time.Now()
	logger := p.logger.With(
		"method", request.Method,
		"host", request.Host,
		"path", request.URL.Path,
		"remote_ip", request.RemoteAddr,
	)

	if request.Method == http.MethodConnect {
		p.handleConnect(writer, request, start, logger)
	} else {
		p.handleHTTP(writer, request, start, logger)
	}
}

func httpError(w http.ResponseWriter, status int, err error, start time.Time, logger *zap.SugaredLogger) {
	http.Error(w, http.StatusText(status), status)
	logger.With(
		"status", status,
		"latency", time.Since(start),
		zap.Error(err),
	).Warn("request failed")
}
