package docker

import (
	"context"
	"net"
	"net/url"
	"strconv"

	"github.com/selebrow/selebrow/pkg/models"
)

type dockerBrowser struct {
	forwardedHost string
	u             *url.URL
	host          string
	ports         map[models.ContainerPort]int
	close         func(ctx context.Context)
}

func (b dockerBrowser) GetURL() *url.URL {
	u := *b.u
	return &u
}

func (b dockerBrowser) GetHost() string {
	return b.host
}

func (b dockerBrowser) GetHostPort(name models.ContainerPort) string {
	p := b.ports[name]
	if p == 0 {
		return ""
	}

	return net.JoinHostPort(b.forwardedHost, strconv.Itoa(p))
}

func (b dockerBrowser) Close(ctx context.Context, _ bool) {
	b.close(ctx)
}
