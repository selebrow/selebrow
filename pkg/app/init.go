package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/selebrow/selebrow/internal/browser/limited"
	"github.com/selebrow/selebrow/internal/browser/pool"
	hc "github.com/selebrow/selebrow/internal/common/client"
	"github.com/selebrow/selebrow/internal/common/conn"
	"github.com/selebrow/selebrow/internal/common/ws"
	"github.com/selebrow/selebrow/internal/services/pw"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/internal/services/wdsession"
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/browsers"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/config"
	dockerclient "github.com/selebrow/selebrow/pkg/docker"
	"github.com/selebrow/selebrow/pkg/event"
	"github.com/selebrow/selebrow/pkg/kubeapi"
	"github.com/selebrow/selebrow/pkg/log"
	"github.com/selebrow/selebrow/pkg/quota"
	"github.com/selebrow/selebrow/pkg/quota/limit"
	"github.com/selebrow/selebrow/pkg/signal"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	browsersFile    = "browsers.yaml"
	podTemplateFile = "pod-template.yaml"
	valuesFile      = "values.yaml"
)

var (
	InitLog       *zap.SugaredLogger
	templateFiles = []string{podTemplateFile, valuesFile}

	InKubernetes = kubeapi.InKubernetes
	InDocker     = dockerclient.InDocker
)

func InitLoggerFunc() *zap.Logger {
	logger := log.GetLogger()
	InitLog = logger.Sugar().Named("init")
	return logger
}

func InitConfigFunc() config.Config {
	flags, exit, err := config.ParseCmdLine(pflag.CommandLine, os.Args[1:])
	if err != nil {
		InitLog.Fatalw("failed to parse command line", zap.Error(err))
	}
	if exit {
		os.Exit(1)
	}

	cfg, err := config.NewConfig(viper.GetViper(), flags)
	if err != nil {
		InitLog.Fatalw("failed to initialize configuration", zap.Error(err))
	}

	return cfg
}

func InitDialerFunc(cfg config.Config) *net.Dialer {
	dialer := &net.Dialer{Timeout: cfg.ConnectTimeout()}
	return dialer
}

func InitTransportFunc(_ config.Config, dialer *net.Dialer) *http.Transport {
	//nolint:errcheck // not going to fail
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          http.DefaultTransport.(*http.Transport).MaxIdleConns,
		IdleConnTimeout:       http.DefaultTransport.(*http.Transport).IdleConnTimeout,
		TLSHandshakeTimeout:   http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout,
		ExpectContinueTimeout: http.DefaultTransport.(*http.Transport).ExpectContinueTimeout,
	}
	return transport
}

func InitHTTPClientFunc(_ config.Config, transport http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: transport,
	}
}

func InitSignalHandlerFunc(_ config.Config) *signal.Handler {
	l := log.GetLogger().Named("signal")
	return signal.NewHandler(5*time.Second, l)
}

func loadBrowsersConfig(cfg config.Config, httpClient hc.HTTPClient) []byte {
	httpPattern := regexp.MustCompile(`(?i)^https?://.+`)
	uris := cfg.BrowsersURI()

	for i, uri := range uris {
		var (
			data []byte
			err  error
		)

		if httpPattern.MatchString(uri) {
			data, err = downloadBrowsersConfig(httpClient, uri)
		} else {
			data, err = os.ReadFile(uri)
		}

		if err != nil {
			errMsg := "failed to load browsers config"
			l := InitLog.With(zap.Error(err), zap.String("uri", uri))
			if i >= len(uris)-1 {
				l.Errorw(errMsg)
			} else {
				l.Warn(errMsg + " (will try fallback URI)")
			}
		} else {
			return data
		}
	}

	InitLog.Fatalw("failed to load browsers config from configured URIs")
	// unreachable code
	return nil
}

func downloadBrowsersConfig(httpClient hc.HTTPClient, uri string) ([]byte, error) {
	InitLog.Infow("downloading browsers config from remote URL", zap.String("url", uri))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, errors.Errorf("request %s failed with code %d", uri, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func InitBrowsersCatalogFunc(_ config.Config, browsersConfig []byte) browsers.BrowsersCatalog {
	cat, err := browsers.NewYamlBrowsersCatalog(browsersConfig)
	if err != nil {
		InitLog.Fatalw("failed to initialize browsers catalog", zap.Error(err))
	}

	return cat
}

func detectBackend(cfg config.Config) config.BackendType {
	if cfg.Backend() == config.BackendKubernetes || (cfg.Backend() == config.BackendAuto && InKubernetes()) {
		return config.BackendKubernetes
	}
	return config.BackendDocker
}

const (
	memoryPerBrowser = 1500 * 1000 * 1000
	cpuPerBrowser    = 1
)

func initLimitQuotaAuthorizer(cfg config.Config, cpus int, memory int64) *limit.LimitQuotaAuthorizer {
	lim := cfg.QuotaLimit()
	if lim < 0 {
		return nil
	} else if lim == 0 {
		if cpus == 0 || memory == 0 {
			InitLog.Warn("available resources information is missing, quota will not be enabled")
			return nil
		}
		// guess quota limit based on assumption that every browser required ~1 core and ~1.5Gb of memory
		cpuLim := cpus / cpuPerBrowser
		memLim := int(memory / memoryPerBrowser)
		lim = max(min(cpuLim, memLim), 1)
		InitLog.Infow("calculated quota limit based on available resources",
			zap.Int("limit", lim), zap.Int("cpus", cpus), zap.Int64("memory", memory))
	}

	l := log.GetLogger().Named("quota")
	return limit.NewLimitQuotaAuthorizer(lim, cfg.QueueSize(), l)
}

func InitPoolManagerFunc(cfg config.Config, mgr browser.BrowserManager, sig *signal.Handler) browser.BrowserManager {
	if cfg.MaxIdle() > 0 {
		l := log.GetLogger().Named("pool")
		f := pool.NewIdleBrowserPoolFactory(cfg, mgr, l)
		pm := pool.NewBrowserPoolManager(f, capabilities.GetHash)
		sig.RegisterShutdownHook(mgr, pm.Shutdown)
		return pm
	}

	return mgr
}

func InitLimitedBrowserManagerFunc(cfg config.Config, mgr browser.BrowserManager, qa quota.QuotaAuthorizer) browser.BrowserManager {
	if !qa.Enabled() {
		return mgr
	}
	l := log.GetLogger().Named("limit")
	return limited.NewLimitedBrowserManager(mgr, qa, cfg.QueueTimeout(), l)
}

func initSessionStorage(sig *signal.Handler) session.SessionStorage {
	l := log.GetLogger().Named("session")

	s := session.NewLocalSessionStorage(l)
	sig.RegisterShutdownHook(s, s.Shutdown)
	return s
}

func InitEventBrokerFunc(_ config.Config, sig *signal.Handler) event.EventBroker {
	const defaultEventBufferSize = 100
	l := log.GetLogger().Named("event")
	eb := event.NewEventBrokerImpl(defaultEventBufferSize, l)
	sig.RegisterShutdownHook(eb, eb.ShutDown)
	return eb
}

func initWDSessionService(
	cfg config.Config,
	mgr browser.BrowserManager,
	storage session.SessionStorage,
	httpClient hc.HTTPClient,
) *wdsession.WDSessionService {
	l := log.GetLogger().Named("wdsession")
	srv := wdsession.NewWDSessionServiceImpl(mgr, storage, httpClient, cfg, time.Now, l)
	return srv
}

func initPWSessionService(
	cfg config.Config,
	dialer *net.Dialer,
	backend config.BackendType,
	mgr browser.BrowserManager,
	storage session.SessionStorage,
) *pw.PWSessionService {
	l := log.GetLogger().Named("playwright")
	// check connection only in docker port mapping mode
	checkConn := backend == config.BackendDocker && portMappingEnabled(cfg)
	s := pw.NewPWSessionService(mgr, storage, dialer, cfg.CreateTimeout(), checkConn, time.Now, l)
	return s
}

func initWSProxy() ws.WSProxy {
	l := log.GetLogger().Named("wsproxy")
	return ws.NewWSProxyImpl(&conn.TcpConnFactory{}, l)
}

func listen(cfg config.Config) string {
	if val := cfg.Listen(); val != "" {
		return val
	}
	if InKubernetes() || InDocker() {
		return config.DefaultListen
	}
	return config.DefaultLocalListen
}
