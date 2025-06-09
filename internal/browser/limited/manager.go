package limited

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/models"
	"github.com/selebrow/selebrow/pkg/quota"
)

type LimitedBrowserManager struct {
	mgr          browser.BrowserManager
	qa           quota.QuotaAuthorizer
	queueTimeout time.Duration
	l            *zap.SugaredLogger
}

func NewLimitedBrowserManager(
	mgr browser.BrowserManager,
	qa quota.QuotaAuthorizer,
	queueTimeout time.Duration,
	l *zap.Logger,
) *LimitedBrowserManager {
	return &LimitedBrowserManager{
		mgr:          mgr,
		qa:           qa,
		queueTimeout: queueTimeout,
		l:            l.Sugar(),
	}
}

func (m *LimitedBrowserManager) Allocate(
	ctx context.Context,
	protocol models.BrowserProtocol,
	caps capabilities.Capabilities,
) (browser.Browser, error) {
	qCtx, cancel := context.WithTimeout(ctx, m.queueTimeout)
	defer cancel()
	if err := m.qa.Reserve(qCtx); err != nil {
		return nil, err
	}

	br, err := m.mgr.Allocate(ctx, protocol, caps)
	if err != nil {
		m.qa.Release()
		return nil, err
	}

	return NewLimitedBrowser(br, m.qa.Release), nil
}
