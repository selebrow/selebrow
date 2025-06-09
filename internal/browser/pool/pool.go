package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/config"
	"github.com/selebrow/selebrow/pkg/models"

	"go.uber.org/zap"
)

type BrowserPool interface {
	Checkout(ctx context.Context, protocol models.BrowserProtocol, caps capabilities.Capabilities) (browser.Browser, error)
	Shutdown(ctx context.Context) error
}

type IdleBrowserPool struct {
	name        string
	idleWd      map[string]*PooledBrowser
	mgr         browser.BrowserManager
	m           sync.RWMutex
	maxIdle     int
	maxAge      time.Duration
	idleTimeout time.Duration
	shutdown    bool
	l           *zap.SugaredLogger
}

func NewIdleBrowserPool(name string, mgr browser.BrowserManager, cfg config.PoolConfig, l *zap.Logger) *IdleBrowserPool {
	pl := l.Sugar().With(zap.String("pool", name))
	pl.Infof("starting pool: maxIdle=%d, maxAge=%v, idleTimeout=%v", cfg.MaxIdle(), cfg.MaxAge(), cfg.IdleTimeout())
	return &IdleBrowserPool{
		name:        name,
		idleWd:      make(map[string]*PooledBrowser),
		mgr:         mgr,
		maxIdle:     cfg.MaxIdle(),
		maxAge:      cfg.MaxAge(),
		idleTimeout: cfg.IdleTimeout(),
		l:           pl,
	}
}

func (p *IdleBrowserPool) Shutdown(ctx context.Context) error {
	p.m.Lock()
	defer p.m.Unlock()
	p.shutdown = true

	p.l.Infof("shutting down the pool, idleCount=%d", len(p.idleWd))
	var wg sync.WaitGroup
	done := make(chan struct{})
	for id, wd := range p.idleWd {
		wd.idle.Stop()
		delete(p.idleWd, id)
		wg.Add(1)
		go func(wd *PooledBrowser) {
			defer wg.Done()
			wd.br.Close(ctx, true)
		}(wd)
	}

	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	return nil
}

func (p *IdleBrowserPool) Checkout(
	ctx context.Context,
	protocol models.BrowserProtocol,
	caps capabilities.Capabilities,
) (browser.Browser, error) {
	wd, err := p.popIdle()
	if err != nil {
		return nil, err
	}

	if wd != nil {
		return wd, nil
	}

	wd, err = p.mgr.Allocate(ctx, protocol, caps)
	if err != nil {
		return nil, err
	}

	return NewPooledBrowser(wd, p.checkin), nil
}

func (p *IdleBrowserPool) PoolState() (int, bool) {
	p.m.RLock()
	defer p.m.RUnlock()
	return len(p.idleWd), p.shutdown
}

func (p *IdleBrowserPool) popIdle() (browser.Browser, error) {
	p.m.Lock()
	defer p.m.Unlock()
	if p.shutdown {
		return nil, fmt.Errorf("pool [%s] is shutdown", p.name)
	}
	for id, wd := range p.idleWd {
		wd.idle.Stop()
		p.l.Debugw("checking out browser", zap.String("browser_id", wd.id), zap.String("url", wd.br.GetURL().String()))
		delete(p.idleWd, id)
		return wd, nil
	}
	return nil, nil
}

func (p *IdleBrowserPool) checkin(wd *PooledBrowser) {
	if cnt, shutdown := p.PoolState(); shutdown || cnt >= p.maxIdle {
		p.l.With(zap.String("browser_id", wd.id)).
			Debugf("dropping browser idleCount=%d, shutdown=%v", cnt, shutdown)
		wd.br.Close(context.Background(), true)
		return
	}

	if age := time.Since(*wd.tm); age > p.maxAge {
		p.l.With(zap.String("browser_id", wd.id), zap.String("url", wd.br.GetURL().String())).
			Debugf("recycling aged browser, age=%v", age)
		wd.br.Close(context.Background(), true)
		return
	}

	p.pushIdle(wd)
}

func (p *IdleBrowserPool) pushIdle(wd *PooledBrowser) {
	p.m.Lock()
	defer p.m.Unlock()

	expireTime := wd.tm.Add(p.maxAge)
	timeout := time.Until(expireTime)
	if timeout > p.idleTimeout {
		timeout = p.idleTimeout
	}

	wd.idle = time.AfterFunc(timeout, func() {
		p.evictIdle(wd)
	})
	p.l.Debugw("checking in browser", zap.String("browser_id", wd.id), zap.String("url", wd.br.GetURL().String()))
	p.idleWd[wd.id] = wd
}

func (p *IdleBrowserPool) evictIdle(wd *PooledBrowser) {
	p.m.Lock()
	if _, ok := p.idleWd[wd.id]; ok {
		delete(p.idleWd, wd.id)
		p.m.Unlock()
		p.l.Debugw("evicting browser", zap.String("browser_id", wd.id), zap.String("url", wd.br.GetURL().String()))
		wd.br.Close(context.Background(), true)
	} else {
		p.m.Unlock()
	}
}
