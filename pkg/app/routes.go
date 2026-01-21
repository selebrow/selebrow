package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/selebrow/selebrow/html"
	"github.com/selebrow/selebrow/internal/common/ws"
	"github.com/selebrow/selebrow/internal/controllers"
	"github.com/selebrow/selebrow/internal/router"
	quotasrv "github.com/selebrow/selebrow/internal/services/quota"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/pkg/browsers"
	"github.com/selebrow/selebrow/pkg/config"
	"github.com/selebrow/selebrow/pkg/event"
	"github.com/selebrow/selebrow/pkg/models"
	"github.com/selebrow/selebrow/pkg/quota"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

type (
	ConfigController interface {
		List(c echo.Context) error
		GetConfig(c echo.Context) error
	}

	WDSessionController interface {
		CreateSession(ctx echo.Context) error
		DeleteSession(ctx echo.Context) error
		Status(ctx echo.Context) error
		ValidateSession(next echo.HandlerFunc) echo.HandlerFunc
	}

	ProxyController interface {
		Proxy(c echo.Context) error
		RewriteProxyUrl(next echo.HandlerFunc) echo.HandlerFunc
		SetPortProxyURL(port models.ContainerPort) echo.MiddlewareFunc
		SetProxyURL(next echo.HandlerFunc) echo.HandlerFunc
		VNCProxy(c echo.Context) error
	}

	BrowsersCatalogController interface {
		Browsers(c echo.Context) error
	}

	QuotaController interface {
		QuotaUsage(c echo.Context) error
	}

	InfoController interface {
		Info(c echo.Context) error
	}

	WDStatusController interface {
		Status(c echo.Context) error
	}

	PWController interface {
		CreateSession(c echo.Context) error
		ValidateSession(next echo.HandlerFunc) echo.HandlerFunc
	}
)

func initEcho(cfg config.Config, l *zap.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = controllers.ErrorHandler

	// Middleware
	InitMiddleware(cfg, e, l)
	return e
}

func InitMiddlewareFunc(_ config.Config, e *echo.Echo, srvLogger *zap.Logger) {
	isStatic := func(c echo.Context) bool {
		return strings.HasPrefix(c.Request().URL.Path, "/static/")
	}

	isUI := func(c echo.Context) bool {
		return strings.HasPrefix(c.Request().URL.Path, "/ui")
	}

	if srvLogger.Core().Enabled(zap.DebugLevel) {
		accLogger := srvLogger.Named("access").Sugar()
		e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			Skipper: func(c echo.Context) bool {
				return isStatic(c) || isUI(c)
			},
			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
				l := accLogger.With(zap.Time("start_time", v.StartTime),
					zap.String("method", v.Method),
					zap.String("uri", v.URI),
					zap.String("remote_ip", v.RemoteIP),
					zap.Duration("latency", v.Latency),
					zap.Int("status", v.Status))
				if v.Error != nil {
					l = l.With(zap.Error(v.Error))
				}
				l.Debug()
				return nil
			},
			LogLatency:   true,
			LogRemoteIP:  true,
			LogMethod:    true,
			LogURI:       true,
			LogRequestID: true,
			LogUserAgent: true,
			LogStatus:    true,
			LogError:     true,
			HandleError:  true,
		}))
	}

	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true, // this will be handled by zap logger
		LogErrorFunc: func(c echo.Context, err error, _ []byte) error {
			srvLogger.With(zap.Error(err), zap.String("uri", c.Request().RequestURI)).Error("panic recovered")
			return err
		},
	}))
}

func InitAPIFunc(
	_ config.Config,
	e *echo.Echo,
	configController ConfigController,
	sessionController WDSessionController,
	proxyController ProxyController,
	catalogController BrowsersCatalogController,
	quotaController QuotaController,
	infoController InfoController,
	wdStatusController WDStatusController,
	playwrightController PWController,
) {
	e.GET("/browsers", catalogController.Browsers)
	e.GET("/status", sessionController.Status)
	e.GET("/quota", quotaController.QuotaUsage)
	e.GET("/info", infoController.Info)
	e.GET("/config", configController.List)
	e.GET("/config/:name", configController.GetConfig)
	e.GET(
		router.SessRoute("/vnc/:%s"),
		proxyController.VNCProxy,
		sessionController.ValidateSession,
		proxyController.SetPortProxyURL(models.VNCPort),
	)
	e.Any(
		router.SessRoute("/download/:%s/*"),
		proxyController.Proxy,
		sessionController.ValidateSession,
		proxyController.SetPortProxyURL(models.FileserverPort),
	)
	e.Any(
		router.SessRoute("/download/:%s"),
		proxyController.Proxy,
		sessionController.ValidateSession,
		proxyController.SetPortProxyURL(models.FileserverPort),
	)
	e.Any(
		router.SessRoute("/clipboard/:%s"),
		proxyController.Proxy,
		sessionController.ValidateSession,
		proxyController.SetPortProxyURL(models.ClipboardPort),
	)
	e.Any(
		router.SessRoute("/devtools/:%s"),
		proxyController.Proxy,
		sessionController.ValidateSession,
		proxyController.SetPortProxyURL(models.DevtoolsPort),
	)
	e.Any(
		router.SessRoute("/devtools/:%s/*"),
		proxyController.Proxy,
		sessionController.ValidateSession,
		proxyController.SetPortProxyURL(models.DevtoolsPort),
	)
	wdhub := e.Group(router.WDHUBPath)
	wdhub.GET("/status", wdStatusController.Status)
	wdhub.POST(router.SessionPath, sessionController.CreateSession)
	wdhub.DELETE(router.SessRoute(router.SessionPath+"/:%s"), sessionController.DeleteSession, sessionController.ValidateSession)
	wdhub.Any(
		router.SessRoute(router.SessionPath+"/:%s/*"),
		proxyController.Proxy,
		sessionController.ValidateSession,
		proxyController.SetProxyURL,
		proxyController.RewriteProxyUrl,
	)

	pw := e.Group(router.PWPath)
	pw.Any(
		router.SessRoute("/vnc/:%s"),
		proxyController.VNCProxy,
		playwrightController.ValidateSession,
		proxyController.SetPortProxyURL(models.VNCPort),
	)
	pwBrowser := pw.Group(router.NameRoute("/:%s"))
	pwBrowser.GET("", playwrightController.CreateSession)
	pwBrowser.GET(router.VersionRoute("/:%s"), playwrightController.CreateSession)
}

func initUI(cfg config.Config, e *echo.Echo, qa quota.QuotaAuthorizer, wdSvc session.SessionService, pwSvc session.SessionService) {
	if !cfg.UI() {
		return
	}
	initRenderer(e)

	uictrl := initUIController(
		cfg,
		map[models.BrowserProtocol]session.SessionService{
			models.WebdriverProtocol:  wdSvc,
			models.PlaywrightProtocol: pwSvc,
		},
		qa,
	)

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, router.UIRoot)
	})
	e.StaticFS(controllers.StaticRoot, echo.MustSubFS(html.StaticFS(), html.StaticFSRoot))

	ui := e.Group(router.UIRoot, middleware.RemoveTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		RedirectCode: http.StatusTemporaryRedirect,
	}))
	ui.GET("", uictrl.Index)

	wd := ui.Group(router.UIWDRoot)
	wd.GET("", uictrl.WDSessions)

	wsSess := wd.Group(router.SessRoute("/:%s"))
	wsSess.GET(router.UIVNCPath, uictrl.WDVNC)
	wsSess.GET(router.UIResetPath, uictrl.WDReset)

	pw := ui.Group(router.UIPWRoot)
	pw.GET("", uictrl.PWSessions)

	pwSess := pw.Group(router.SessRoute("/:%s"))
	pwSess.GET(router.UIVNCPath, uictrl.PWVNC)
	pwSess.GET(router.UIResetPath, uictrl.PWReset)

	InitLog.Infof("UI initialized at %s", uictrl.URL())
}

func initRenderer(e *echo.Echo) {
	r, err := html.NewTemplateRenderer(html.TemplatesFS())
	if err != nil {
		InitLog.Fatalw("failed to initialize template renderer", zap.Error(err))
	}
	e.Renderer = r
}

func initUIController(
	cfg config.Config,
	services map[models.BrowserProtocol]session.SessionService,
	qa quota.QuotaAuthorizer,
) *controllers.UIController {
	return controllers.NewUIController(services, qa, listen(cfg), cfg.VNCPassword())
}

func initConfigController(browsersConfig []byte) *controllers.ConfigController {
	return controllers.NewConfigController(map[string]string{browsersFile: string(browsersConfig)})
}

func initWDSessionController(
	svc session.SessionService,
	eb event.EventBroker,
	proxyOpts *config.ProxyOpts,
	cLog *zap.Logger,
) *controllers.WDSessionController {
	return controllers.NewWDSessionController(svc, eb, time.Now, proxyOpts, cLog.Named("wdsession"))
}

func initProxyController(transport http.RoundTripper, p ws.WSProxy, cLog *zap.Logger) *controllers.ProxyController {
	return controllers.NewProxyController(transport, p, cLog.Named("proxy"))
}

func initBrowsersCatalogController(cat browsers.BrowsersCatalog) *controllers.BrowsersCatalogController {
	return controllers.NewBrowsersCatalogController(cat)
}

func initWDStatusController() *controllers.WDStatusController {
	return controllers.NewWDStatusController()
}

func initQuotaController(qa quota.QuotaAuthorizer) *controllers.QuotaController {
	srv := quotasrv.NewQuotaService(qa)
	return controllers.NewQuotaController(srv)
}

func initInfoController(appName, gitRef, gitSha string) *controllers.InfoController {
	return controllers.NewInfoController(appName, gitRef, gitSha)
}

func initPlayWrightController(
	svc session.SessionService,
	transport http.RoundTripper,
	eb event.EventBroker,
	proxyOpts *config.ProxyOpts,
	cLog *zap.Logger,
) *controllers.PWController {
	return controllers.NewPWController(svc, transport, eb, time.Now, proxyOpts, cLog.Named("playwright"))
}
