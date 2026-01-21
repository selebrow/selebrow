package app

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	stdProxy "golang.org/x/net/proxy"

	"github.com/selebrow/selebrow/internal/proxy"
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/browsers"
	"github.com/selebrow/selebrow/pkg/config"
	dockerclient "github.com/selebrow/selebrow/pkg/docker"
	"github.com/selebrow/selebrow/pkg/event"
	"github.com/selebrow/selebrow/pkg/kubeapi"
	"github.com/selebrow/selebrow/pkg/quota"
	"github.com/selebrow/selebrow/pkg/signal"
)

var (
	InitLogger                func() *zap.Logger                                                                  = InitLoggerFunc
	InitConfig                func() config.Config                                                                = InitConfigFunc
	InitDialer                func(config.Config) *net.Dialer                                                     = InitDialerFunc
	InitTransport             func(config.Config, *net.Dialer) *http.Transport                                    = InitTransportFunc
	InitHTTPClient            func(config.Config, http.RoundTripper) *http.Client                                 = InitHTTPClientFunc
	InitBrowsersCatalog       func(config.Config, []byte) browsers.BrowsersCatalog                                = InitBrowsersCatalogFunc
	InitSignalHandler         func(config.Config) *signal.Handler                                                 = InitSignalHandlerFunc
	InitKubeClient            func(config.Config) kubeapi.KubernetesClient                                        = InitKubeClientFunc
	InitDockerClient          func(config.Config) dockerclient.DockerClient                                       = InitDockerClientFunc
	InitPoolManager           func(config.Config, browser.BrowserManager, *signal.Handler) browser.BrowserManager = InitPoolManagerFunc
	InitEventBroker           func(config.Config, *signal.Handler) event.EventBroker                              = InitEventBrokerFunc
	InitMiddleware            func(config.Config, *echo.Echo, *zap.Logger)                                        = InitMiddlewareFunc
	InitDockerQuotaAuthorizer func(
		config.Config,
		dockerclient.DockerClient,
	) quota.QuotaAuthorizer = InitDockerQuotaAuthorizerFunc
	InitLimitedBrowserManager func(
		config.Config,
		browser.BrowserManager,
		quota.QuotaAuthorizer,
	) browser.BrowserManager = InitLimitedBrowserManagerFunc
	InitKubernetesQuotaAuthorizer func(
		config.Config,
		kubeapi.KubernetesClient,
		*signal.Handler,
	) quota.QuotaAuthorizer = InitKubernetesQuotaAuthorizerFunc
	InitAPI func(
		config.Config,
		*echo.Echo,
		ConfigController,
		WDSessionController,
		ProxyController,
		BrowsersCatalogController,
		QuotaController,
		InfoController,
		WDStatusController,
		PWController,
	) = InitAPIFunc

	InitEventAdapter func(
		config.Config,
		event.EventBroker,
		config.BackendType,
		*signal.Handler,
	) = func(_ config.Config, _ event.EventBroker, _ config.BackendType, _ *signal.Handler) {}

	InitProxy          func(cfg config.Config) *proxy.Proxy                     = InitProxyFunc
	InitProxyHandler   func(cfg config.Config, logger *zap.Logger) http.Handler = InitProxyHandlerFunc
	InitProxyTransport func(
		_ config.Config,
		d stdProxy.ContextDialer,
		proxyFunc func(reqURL *url.URL) (*url.URL, error),
	) *http.Transport = InitProxyTransportFunc
)

func Run(gitRef, gitSha, appName string) {
	l := InitLogger()
	mainLog := l.Sugar().Named("app")
	appVersion := fmt.Sprintf("%s-%s", gitRef, gitSha)
	mainLog.Infof("starting %s build %s (%s/%s)", appName, appVersion, runtime.GOOS, runtime.GOARCH)

	cfg := InitConfig()
	sig := InitSignalHandler(cfg)

	dialer := InitDialer(cfg)
	transport := InitTransport(cfg, dialer)
	client := InitHTTPClient(cfg, transport)

	browsersConfig := loadBrowsersConfig(cfg, http.DefaultClient) // using Default client with sane timeout defaults
	catalog := InitBrowsersCatalog(cfg, browsersConfig)

	backend := detectBackend(cfg)
	qa, mgr, proxyOpts := initBackend(cfg, backend, catalog, sig)

	mgr = InitPoolManager(cfg, mgr, sig)
	mgr = InitLimitedBrowserManager(cfg, mgr, qa)

	sStorage := initSessionStorage(sig)

	eb := InitEventBroker(cfg, sig)
	InitEventAdapter(cfg, eb, backend, sig)

	wdSvc := initWDSessionService(cfg, mgr, sStorage, client)
	pwSvc := initPWSessionService(cfg, dialer, backend, mgr, sStorage)

	cLog := l.Named("controller")
	wsproxy := initWSProxy()

	configController := initConfigController(browsersConfig)
	sessionController := initWDSessionController(wdSvc, eb, proxyOpts, cLog)
	proxyController := initProxyController(transport, wsproxy, cLog)
	catalogController := initBrowsersCatalogController(catalog)
	wdStatusController := initWDStatusController()
	quotaController := initQuotaController(qa)
	infoController := initInfoController(appName, gitRef, gitSha)
	playwrightController := initPlayWrightController(pwSvc, transport, eb, proxyOpts, cLog)

	srvLog := l.Named("server")
	e := initEcho(cfg, srvLog)
	// Routes
	initUI(cfg, e, qa, wdSvc, pwSvc)
	InitAPI(
		cfg,
		e,
		configController,
		sessionController,
		proxyController,
		catalogController,
		quotaController,
		infoController,
		wdStatusController,
		playwrightController,
	)

	// Start proxy if enabled
	if cfg.ProxyEnabled() {
		p := InitProxy(cfg)
		sig.RegisterShutdownHook(nil, p.Shutdown)
		go p.Start()
	}

	// Start server
	go func() {
		lstn := listen(cfg)
		sl := srvLog.Sugar()
		sl.Infof("listening on %s", lstn)
		if err := e.Start(lstn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sl.Fatalw("failed to start the server", zap.Error(err))
		}
	}()

	sig.RegisterShutdownHook(nil, e.Shutdown)
	os.Exit(sig.Start())
}

func initBackend(
	cfg config.Config,
	backend config.BackendType,
	catalog browsers.BrowsersCatalog,
	sig *signal.Handler,
) (quota.QuotaAuthorizer, browser.BrowserManager, *config.ProxyOpts) {
	var (
		qa          quota.QuotaAuthorizer
		mgr         browser.BrowserManager
		proxyHostFn config.ProxyHostFunc
	)

	InitLog.With(zap.String("lineage", cfg.Lineage())).Infof("initializing %s backend", backend)

	if backend == config.BackendKubernetes {
		client := InitKubeClient(cfg)
		qa = InitKubernetesQuotaAuthorizer(cfg, client, sig)
		templatesData := readKubeTemplates(cfg)
		mgr = initKubernetesWebDriverManager(cfg, client, templatesData, catalog, sig)
		// proxy host expected to be set externally via Helm
		proxyHostFn = func() string {
			return ""
		}
	} else {
		client := InitDockerClient(cfg)
		qa = InitDockerQuotaAuthorizer(cfg, client)
		mgr, proxyHostFn = initDockerWebDriverManager(cfg, client, catalog)
	}
	proxyOpts := initProxyOpts(cfg, proxyHostFn)
	return qa, mgr, proxyOpts
}
