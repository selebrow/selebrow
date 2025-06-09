package pool

import (
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/config"

	"go.uber.org/zap"
)

type BrowserPoolFactory interface {
	GetPool(name string) BrowserPool
}

type IdleBrowserPoolFactory struct {
	cfg config.PoolConfig
	mgr browser.BrowserManager
	l   *zap.Logger
}

func NewIdleBrowserPoolFactory(cfg config.PoolConfig, mgr browser.BrowserManager, l *zap.Logger) *IdleBrowserPoolFactory {
	return &IdleBrowserPoolFactory{
		cfg: cfg,
		mgr: mgr,
		l:   l,
	}
}

func (f *IdleBrowserPoolFactory) GetPool(name string) BrowserPool {
	return NewIdleBrowserPool(name, f.mgr, f.cfg, f.l)
}
