package dialer

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/net/proxy"
)

type (
	ProxyDialer struct {
		proxyAddress string
		dialer       proxy.ContextDialer
	}

	proxyRequestKeyType struct{}
)

var ProxyRequestKey = proxyRequestKeyType{}

func NewProxyDialer(proxyUrl *url.URL, dialer proxy.ContextDialer) *ProxyDialer {
	return &ProxyDialer{
		proxyAddress: proxyUrl.Host,
		dialer:       dialer,
	}
}

func (d *ProxyDialer) DialContext(ctx context.Context, network, _ string) (net.Conn, error) {
	proxyRequest, ok := ctx.Value(ProxyRequestKey).(*http.Request)
	if !ok {
		return nil, errors.New("proxy request is not set in the context")
	}

	conn, err := d.dialer.DialContext(ctx, network, d.proxyAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to upstream proxy")
	}

	// forward proxy request
	if err := proxyRequest.Write(conn); err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "failed to send proxy request to upstream proxy")
	}

	// read connect response
	// Okay to use and discard buffered reader here, because
	// TLS server will not speak until spoken to.
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, proxyRequest)
	if err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "failed to read proxy response from upstream proxy")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, errors.Errorf("failed to connect to upstream proxy: %s", resp.Status)
	}
	return conn, nil
}

func (d *ProxyDialer) Dial(network, addr string) (c net.Conn, err error) {
	return d.DialContext(context.Background(), network, addr)
}
