package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/selebrow/selebrow/internal/common/clock"
	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/config"
	"github.com/selebrow/selebrow/pkg/dto"
	"github.com/selebrow/selebrow/pkg/event"
	evmodels "github.com/selebrow/selebrow/pkg/event/models"
	"github.com/selebrow/selebrow/pkg/models"
)

const SessionKey = "session"

type WDSessionController struct {
	srv   session.SessionService
	eb    event.EventBroker
	now   clock.NowFunc
	proxy *models.ProxyOptions
	l     *zap.SugaredLogger
}

func NewWDSessionController(
	srv session.SessionService,
	eb event.EventBroker,
	now clock.NowFunc,
	proxyOpts *config.ProxyOpts,
	l *zap.Logger,
) *WDSessionController {
	var proxy *models.ProxyOptions
	if proxyOpts != nil {
		proxy = models.NewHTTPProxy(proxyOpts.ProxyHost, proxyOpts.NoProxy)
	}
	return &WDSessionController{
		srv:   srv,
		eb:    eb,
		now:   now,
		proxy: proxy,
		l:     l.Sugar(),
	}
}

func (s *WDSessionController) CreateSession(ctx echo.Context) error {
	ev := evmodels.SessionRequested{
		Protocol: models.WebdriverProtocol,
	}

	defer func() {
		if r := recover(); r != nil {
			ev.Error = models.NewInternalServerError(errors.Errorf("panic: %v", r))
			s.eb.Publish(evmodels.NewSessionRequestedEvent(ev))
			panic(r)
		}
		s.eb.Publish(evmodels.NewSessionRequestedEvent(ev))
	}()

	caps, err := capabilities.NewCapabilities(ctx.Request().Body, s.proxy)
	if err != nil {
		ev.Error = models.BadWDSessionParameters(err)
		return ev.Error
	}

	ev.BrowserName = caps.GetName()
	ev.BrowserVersion = caps.GetVersion()
	if s.l.Desugar().Core().Enabled(zap.DebugLevel) {
		var c map[string]interface{}
		// error can't happen (already checked in capabilities.NewCapabilities above)
		_ = json.Unmarshal(caps.GetRawCapabilities(), &c)
		s.l.Debugw("Webdriver session requested", zap.Any("caps", c))
	}

	start := s.now()
	sess, err := s.srv.CreateSession(ctx.Request().Context(), caps)
	if err != nil {
		s.l.Errorw("failed to create session", zap.Error(err))
		ev.Error = models.WDSessionNotCreatedError(models.WrapCancelledErr(err))
		return ev.Error
	}
	ev.StartDuration = sess.Created().Sub(start)
	return ctx.JSON(http.StatusOK, sess.Resp())
}

func (s *WDSessionController) ValidateSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param(router.SessionParam)
		i := strings.Index(c.Path(), fmt.Sprintf("/:%s", router.SessionParam))
		if i < 0 {
			s.l.Panicf("Middleware applied to the wrong route: %s", c.Path())
		}

		sess, err := s.srv.FindSession(id)
		if err != nil {
			return models.NewW3CErr(http.StatusNotFound, "unknown session", err)
		}

		c.Set(SessionKey, sess)
		return next(c)
	}
}

func (s *WDSessionController) DeleteSession(ctx echo.Context) error {
	sess, _ := ctx.Get(SessionKey).(*session.Session)
	ev := evmodels.SessionReleased{
		Protocol:        models.WebdriverProtocol,
		BrowserName:     sess.ReqCaps().GetName(),
		BrowserVersion:  sess.ReqCaps().GetVersion(),
		SessionDuration: s.now().Sub(sess.Created()),
	}
	s.srv.DeleteSession(sess)
	s.eb.Publish(evmodels.NewSessionReleasedEvent(ev))

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"value": nil,
	})
}

func (s *WDSessionController) Status(ctx echo.Context) error {
	sl := s.srv.ListSessions()
	status := &dto.Status{
		Total:    len(sl),
		Sessions: make(map[string][]dto.SessionStatus),
	}
	for _, sess := range sl {
		p := sess.Platform()
		status.Sessions[p] = append(status.Sessions[p], dto.SessionStatus{
			ID:  sess.ID(),
			URL: sess.Browser().GetURL().String(),
		})
	}
	return ctx.JSON(http.StatusOK, status)
}
