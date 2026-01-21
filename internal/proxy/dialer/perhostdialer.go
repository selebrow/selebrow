package dialer

import (
	"context"
	"net"
	"net/url"

	"golang.org/x/net/proxy"
)

type (
	PerHostDialer struct {
		dialer       proxy.ContextDialer
		bypassDialer proxy.ContextDialer
		proxyFunc    func(*url.URL) (*url.URL, error)
	}
)

func NewPerHostDialer(
	dialer proxy.ContextDialer,
	bypass proxy.ContextDialer,
	proxyFunc func(*url.URL) (*url.URL, error),
) *PerHostDialer {
	return &PerHostDialer{
		dialer:       dialer,
		bypassDialer: bypass,
		proxyFunc:    proxyFunc,
	}
}

func (d *PerHostDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	var err error
	dialer, err := d.getDialer(address)
	if err != nil {
		return nil, err
	}

	return dialer.DialContext(ctx, network, address)
}

func (d *PerHostDialer) Dial(network, addr string) (c net.Conn, err error) {
	return d.DialContext(context.Background(), network, addr)
}

func (d *PerHostDialer) getDialer(address string) (proxy.ContextDialer, error) {
	u, err := url.Parse("https://" + address)
	if err != nil {
		return nil, err
	}

	proxyURL, err := d.proxyFunc(u)
	if err != nil {
		return nil, err
	}
	if proxyURL == nil {
		return d.bypassDialer, nil
	}
	return d.dialer, nil
}
