package proxy

import (
	"context"
	"net"
	"net/url"

	"go.uber.org/zap"
)

type HostResolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
}

func ResolveProxyFunc(
	resolver HostResolver,
	proxyFunc func(*url.URL) (*url.URL, error),
	logger *zap.Logger,
) func(*url.URL) (*url.URL, error) {
	sLogger := logger.Sugar()
	return func(u *url.URL) (*url.URL, error) {
		host := u.Hostname()
		if ip := net.ParseIP(host); ip != nil {
			// already IP address
			return proxyFunc(u)
		}

		addrs, err := resolver.LookupHost(context.Background(), host)
		if err != nil || len(addrs) == 0 {
			// no IP address found - let upstream proxy and non ip rules decide
			sLogger.Warnw("failed to resolve host", "host", host, zap.Error(err))
			return proxyFunc(u)
		}
		ipURL := *u
		ipURL.Host = addrs[0]
		if port := u.Port(); port != "" {
			ipURL.Host = net.JoinHostPort(addrs[0], port)
		}
		return proxyFunc(&ipURL)
	}
}
