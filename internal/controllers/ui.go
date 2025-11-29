package controllers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/pkg/models"
	"github.com/selebrow/selebrow/pkg/quota"
)

const (
	StaticRoot = "/static/"
)

type UIController struct {
	services    map[models.BrowserProtocol]session.SessionService
	qa          quota.QuotaAuthorizer
	url         string
	vncPassword string
}

type queueData struct {
	Limit int
	Size  int
}

type quotaData struct {
	Allocated int
	Limit     int
	Queue     *queueData
}

type indexData struct {
	WDLink  string
	PWLink  string
	WDCount int
	PWCount int
	Quota   *quotaData
}

type sessionData struct {
	Root     string
	Protocol string
	Sessions []sessionItem
}

type sessionItem struct {
	ID             string
	CreatedAt      string
	Browser        string
	BrowserVersion string
	Name           string
	VNC            bool
	VNCLink        string
	ResetLink      string
}

type vncData struct {
	ID       string
	URLPath  string
	Password string
}

func NewUIController(
	services map[models.BrowserProtocol]session.SessionService,
	qa quota.QuotaAuthorizer,
	listen, vncPassword string,
) *UIController {
	u := getURL(listen)

	return &UIController{
		services:    services,
		qa:          qa,
		url:         u,
		vncPassword: vncPassword,
	}
}

func (u *UIController) Index(c echo.Context) error {
	var qData *quotaData
	if u.qa != nil && u.qa.Enabled() {
		var quData *queueData
		if qq, ok := u.qa.(quota.QuotaQueue); ok && qq.QueueLimit() > 0 {
			quData = &queueData{
				Limit: qq.QueueLimit(),
				Size:  qq.QueueSize(),
			}
		}
		qData = &quotaData{
			Allocated: u.qa.Allocated(),
			Limit:     u.qa.Limit(),
			Queue:     quData,
		}
	}

	data := &indexData{
		WDLink:  path.Join(router.UIRoot, router.UIWDRoot),
		PWLink:  path.Join(router.UIRoot, router.UIPWRoot),
		WDCount: len(u.services[models.WebdriverProtocol].ListSessions()),
		PWCount: len(u.services[models.PlaywrightProtocol].ListSessions()),
		Quota:   qData,
	}
	return c.Render(http.StatusOK, "index.tmpl", data)
}

func (u *UIController) WDSessions(c echo.Context) error {
	return u.sessions(c, models.WebdriverProtocol, router.UIWDRoot)
}

func (u *UIController) PWSessions(c echo.Context) error {
	return u.sessions(c, models.PlaywrightProtocol, router.UIPWRoot)
}

func (u *UIController) sessions(c echo.Context, protocol models.BrowserProtocol, basePath string) error {
	data := &sessionData{
		Root:     router.UIRoot,
		Protocol: string(protocol),
		Sessions: u.getSessions(protocol, basePath),
	}
	return c.Render(http.StatusOK, "sessions.tmpl", data)
}

func (u *UIController) WDVNC(c echo.Context) error {
	return u.vnc(c, models.WebdriverProtocol, router.VNCPath)
}

func (u *UIController) PWVNC(c echo.Context) error {
	return u.vnc(c, models.PlaywrightProtocol, path.Join(router.PWPath, router.VNCPath))
}

func (u *UIController) WDReset(c echo.Context) error {
	return u.reset(c, models.WebdriverProtocol, router.UIWDRoot)
}

func (u *UIController) PWReset(c echo.Context) error {
	return u.reset(c, models.PlaywrightProtocol, router.UIPWRoot)
}

func (u *UIController) URL() string {
	return u.url
}

func (u *UIController) vnc(c echo.Context, protocol models.BrowserProtocol, root string) error {
	id := c.Param(router.SessionParam)
	ps := u.services[protocol]
	s, err := ps.FindSession(id)
	if err != nil {
		return models.NewNotFoundError(err)
	}

	if !s.ReqCaps().IsVNCEnabled() {
		return models.NewBadRequestError(errors.New("VNC was not enabled for this session"))
	}

	data := &vncData{
		ID:       id,
		URLPath:  path.Join(root, id),
		Password: u.vncPassword,
	}
	return c.Render(http.StatusOK, "vnc.tmpl", data)
}

func (u *UIController) reset(c echo.Context, protocol models.BrowserProtocol, root string) error {
	id := c.Param(router.SessionParam)

	ps := u.services[protocol]
	s, err := ps.FindSession(id)

	if err != nil {
		return models.NewNotFoundError(err)
	}

	ps.DeleteSession(s)
	return c.Redirect(http.StatusSeeOther, path.Join(router.UIRoot, root))
}

func (u *UIController) getSessions(protocol models.BrowserProtocol, basePath string) []sessionItem {
	sessions := u.services[protocol].ListSessions()
	slices.SortFunc(sessions, func(a, b *session.Session) int {
		return int(a.Created().UnixMilli() - b.Created().UnixMilli())
	})
	res := make([]sessionItem, len(sessions))
	for i, s := range sessions {
		res[i] = sessionItem{
			ID:             s.ID(),
			CreatedAt:      s.Created().Format(time.DateTime),
			Browser:        s.ReqCaps().GetName(),
			BrowserVersion: s.ReqCaps().GetVersion(),
			Name:           s.ReqCaps().GetTestName(),
			VNC:            s.ReqCaps().IsVNCEnabled(),
			VNCLink:        path.Join(router.UIRoot, basePath, s.ID(), router.UIVNCPath),
			ResetLink:      path.Join(router.UIRoot, basePath, s.ID(), router.UIResetPath),
		}
	}
	return res
}

func getURL(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return listen
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	hostport := host
	if port != "" {
		iPort, err := net.DefaultResolver.LookupPort(context.Background(), "tcp", port)
		if err != nil {
			return listen
		}
		hostport = fmt.Sprintf("%s:%d", host, iPort)
	}

	u := url.URL{
		Scheme: "http",
		Host:   hostport,
		Path:   router.UIRoot,
	}
	return u.String()
}
