package pool

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/models"
)

type (
	GetHashFunc func(caps capabilities.Capabilities) []byte

	BrowserPoolManager struct {
		pools    map[string]BrowserPool
		m        sync.RWMutex
		f        BrowserPoolFactory
		getHash  GetHashFunc
		shutdown bool
	}
)

func NewBrowserPoolManager(f BrowserPoolFactory, getHash GetHashFunc) *BrowserPoolManager {
	return &BrowserPoolManager{
		pools:   make(map[string]BrowserPool),
		f:       f,
		getHash: getHash,
	}
}

func (m *BrowserPoolManager) Allocate(
	ctx context.Context,
	protocol models.BrowserProtocol,
	caps capabilities.Capabilities,
) (browser.Browser, error) {
	if m.isShutdown() {
		return nil, errors.New("Pool manager was shutdown")
	}

	name := m.getUniquePoolName(protocol, caps)

	pool, ok := m.getPool(name)
	if !ok {
		pool = m.createPool(name)
	}
	return pool.Checkout(ctx, protocol, caps)
}

func (m *BrowserPoolManager) Shutdown(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.shutdown = true

	var wg sync.WaitGroup
	done := make(chan struct{})
	for _, p := range m.pools {
		wg.Add(1)
		go func(pool BrowserPool) {
			defer wg.Done()
			_ = pool.Shutdown(ctx)
		}(p)
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

func (m *BrowserPoolManager) getPool(name string) (BrowserPool, bool) {
	m.m.RLock()
	defer m.m.RUnlock()
	p, ok := m.pools[name]
	return p, ok
}

func (m *BrowserPoolManager) createPool(name string) BrowserPool {
	m.m.Lock()
	defer m.m.Unlock()
	// double check to avoid race condition with getPool/createPool
	p, ok := m.pools[name]
	if !ok {
		p = m.f.GetPool(name)
		m.pools[name] = p
	}
	return p
}

func (m *BrowserPoolManager) isShutdown() bool {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.shutdown
}

func (m *BrowserPoolManager) getUniquePoolName(protocol models.BrowserProtocol, caps capabilities.Capabilities) string {
	return fmt.Sprintf("%s-%s-%s", string(protocol), caps.GetName(), hex.EncodeToString(m.getHash(caps)))
}
