package app

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/http/httpproxy"
	stdProxy "golang.org/x/net/proxy"

	"github.com/selebrow/selebrow/internal/proxy"
	"github.com/selebrow/selebrow/internal/proxy/dialer"
	"github.com/selebrow/selebrow/internal/proxy/handlers"
	"github.com/selebrow/selebrow/pkg/config"
	"github.com/selebrow/selebrow/pkg/log"
)

func InitProxyFunc(cfg config.Config) *proxy.Proxy {
	logger := log.GetLogger().Named("proxy")
	handler := InitProxyHandler(cfg, logger)

	p := proxy.NewProxy(handler, cfg.ProxyListen(), logger)
	return p
}

func InitProxyHandlerFunc(cfg config.Config, logger *zap.Logger) http.Handler {
	d := &net.Dialer{
		Timeout:   cfg.ProxyConnectTimeout(),
		KeepAlive: 30 * time.Second,
	}

	proxyFunc := httpproxy.FromEnvironment().ProxyFunc()
	if cfg.ProxyResolveHost() {
		proxyFunc = proxy.ResolveProxyFunc(net.DefaultResolver, proxyFunc, logger.Named("resolver"))
	}
	tr := InitProxyTransport(cfg, d, proxyFunc)
	connectDialer := initProxyConnectDialer(cfg, d, proxyFunc)

	logger = logger.WithOptions(zap.IncreaseLevel(cfg.ProxyAccessLogLevel()))
	handler := handlers.NewProxyHandler(tr, connectDialer, logger.Named("access"))
	return handler
}

func InitProxyTransportFunc(_ config.Config, d stdProxy.ContextDialer, proxyFunc func(reqURL *url.URL) (*url.URL, error)) *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone() //nolint:errcheck // false positive
	tr.MaxIdleConnsPerHost = 25
	tr.DialContext = d.DialContext
	tr.Proxy = func(request *http.Request) (*url.URL, error) {
		return proxyFunc(request.URL)
	}
	return tr
}

func initProxyConnectDialer(_ config.Config, d *net.Dialer, proxyFunc func(reqURL *url.URL) (*url.URL, error)) stdProxy.ContextDialer {
	if proxyURL := getEnvAny("HTTPS_PROXY", "https_proxy"); proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			InitLog.With(zap.Error(err)).Fatalf("failed to parse proxy URL: %s", proxyURL)
		}
		proxyDialer := dialer.NewProxyDialer(u, d)
		perHostDialer := dialer.NewPerHostDialer(proxyDialer, d, proxyFunc)
		return perHostDialer
	}
	return d
}

func getEnvAny(names ...string) string {
	for _, n := range names {
		if val := os.Getenv(n); val != "" {
			return val
		}
	}
	return ""
}

func initProxyOpts(cfg config.Config, proxyHostFn config.ProxyHostFunc) *config.ProxyOpts {
	proxyOpts, err := cfg.ProxyOpts(proxyHostFn)
	if err != nil {
		InitLog.Fatalw("failed to initialize proxy options", zap.Error(err))
	}
	if proxyOpts != nil {
		InitLog.Infow(
			"browser proxy settings",
			zap.String("proxy_host", proxyOpts.ProxyHost),
			zap.String("no_proxy", proxyOpts.NoProxy),
		)
	}
	return proxyOpts
}
