package pool

import (
	"context"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/models"
)

type CheckinFunc func(wd *PooledBrowser)

type PooledBrowser struct {
	id      string
	br      browser.Browser
	idle    *time.Timer
	tm      *time.Time
	checkin CheckinFunc
}

func NewPooledBrowser(wd browser.Browser, ch CheckinFunc) *PooledBrowser {
	now := time.Now()
	return &PooledBrowser{
		id:      uuid.New().String(),
		br:      wd,
		tm:      &now,
		checkin: ch,
	}
}

func (w *PooledBrowser) GetURL() *url.URL {
	return w.br.GetURL()
}

func (w *PooledBrowser) GetHost() string {
	return w.br.GetHost()
}

func (w *PooledBrowser) GetHostPort(name models.ContainerPort) string {
	return w.br.GetHostPort(name)
}

func (w *PooledBrowser) Close(ctx context.Context, trash bool) {
	if trash {
		w.br.Close(ctx, true)
	} else {
		w.checkin(w)
	}
}
