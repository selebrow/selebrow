package kubernetes

import (
	"context"
	"net"
	"net/url"
	"strconv"

	"github.com/selebrow/selebrow/pkg/models"
)

type kubernetesBrowser struct {
	forwardedHost string
	u             *url.URL
	host          string
	ports         map[models.ContainerPort]int
	close         func(ctx context.Context)
}

func (b kubernetesBrowser) GetURL() *url.URL {
	u := *b.u
	return &u
}

func (b kubernetesBrowser) GetHost() string {
	return b.host
}

func (b kubernetesBrowser) GetHostPort(name models.ContainerPort) string {
	p := b.ports[name]
	if p == 0 {
		return ""
	}

	return net.JoinHostPort(b.forwardedHost, strconv.Itoa(p))
}

func (b kubernetesBrowser) Close(ctx context.Context, _ bool) {
	b.close(ctx)
}
