package proxy

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Proxy struct {
	server *http.Server
	listen string
	logger *zap.SugaredLogger
}

func NewProxy(handler http.Handler, listen string, logger *zap.Logger) *Proxy {
	server := &http.Server{
		Addr:              listen,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}
	return &Proxy{
		server: server,
		listen: listen,
		logger: logger.Sugar(),
	}
}

func (p *Proxy) Start() {
	p.logger.Infof("starting proxy on %s", p.listen)
	if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		p.logger.Fatalw("failed to start proxy", zap.Error(err))
	}
	p.logger.Infof("proxy stopped")
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	return p.server.Shutdown(ctx)
}
