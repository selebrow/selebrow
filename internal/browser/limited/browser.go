package limited

import (
	"context"
	"net/url"

	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/models"
)

type ReleaseFunc func() int

type LimitedBrowser struct {
	br      browser.Browser
	release ReleaseFunc
}

func NewLimitedBrowser(br browser.Browser, rel ReleaseFunc) *LimitedBrowser {
	return &LimitedBrowser{
		br:      br,
		release: rel,
	}
}

func (b *LimitedBrowser) GetURL() *url.URL {
	return b.br.GetURL()
}

func (b *LimitedBrowser) GetHost() string {
	return b.br.GetHost()
}

func (b *LimitedBrowser) GetHostPort(name models.ContainerPort) string {
	return b.br.GetHostPort(name)
}

func (b *LimitedBrowser) Close(ctx context.Context, trash bool) {
	b.release()
	b.br.Close(ctx, trash)
}
