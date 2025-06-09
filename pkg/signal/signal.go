package signal

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type ShutdownHook func(ctx context.Context) error

type Handler struct {
	hooks   map[any][]ShutdownHook
	timeout time.Duration
	l       *zap.SugaredLogger
}

func NewHandler(timeout time.Duration, l *zap.Logger) *Handler {
	return &Handler{
		hooks:   make(map[any][]ShutdownHook),
		timeout: timeout,
		l:       l.Sugar(),
	}
}

func (h *Handler) Start() int {
	c := make(chan os.Signal, 2)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	sig := <-c

	h.l.Infow("signal caught, shutting down...", zap.String("signal", sig.String()))

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	done := h.performShutdown(ctx)

	select {
	case <-ctx.Done():
		h.l.Warnf("shutdown hooks did not complete within %v, exiting immediately", h.timeout)
		return 1
	case <-done:
		h.l.Infof("graceful shutdown completed in %v", time.Since(start))
		return 0
	case sig = <-c:
		h.l.Infow("second signal caught, exiting immediately", zap.String("signal", sig.String()))
	}

	return 1
}

func (h *Handler) RegisterShutdownHook(group any, hook ShutdownHook) {
	h.hooks[group] = append(h.hooks[group], hook)
}

func (h *Handler) performShutdown(ctx context.Context) <-chan struct{} {
	var wg sync.WaitGroup
	done := make(chan struct{})
	for _, s := range h.hooks {
		wg.Add(1)
		go func(hooks []ShutdownHook) {
			defer wg.Done()
			for i := len(hooks) - 1; i >= 0; i-- {
				if err := hooks[i](ctx); err != nil {
					h.l.Warnw("shutdown hook failed", zap.Error(err))
				}
			}
		}(s)
	}

	go func() {
		defer close(done)
		wg.Wait()
	}()

	return done
}
