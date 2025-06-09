package controllers

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/selebrow/selebrow/internal/common/clock"
	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/internal/services/session"
	"github.com/selebrow/selebrow/pkg/event"
	evmodels "github.com/selebrow/selebrow/pkg/event/models"
	"github.com/selebrow/selebrow/pkg/models"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	PWVncParamQ             = "vnc"
	PWArgParamQ             = "arg"
	PWHeadlessParamQ        = "headless"
	PWChannelParamQ         = "channel"
	PWResolutionParamQ      = "resolution"
	PWEnvParamQ             = "env"
	PWLinkParamQ            = "link"
	PWHostParamQ            = "host"
	PWNetworkParamQ         = "network"
	PWLabelParamQ           = "label"
	PWFirefoxUserPrefParamQ = "firefoxUserPref"

	PWLaunchOptionsParamQ = "launch-options"
)

var (
	//nolint:gocritic // gocritic suggests wrong regex
	resolutionRegex = regexp.MustCompile(`^(|\d+x\d+x\d+)$`)
	pwEnvRegex      = regexp.MustCompile(`(?i)^[0-9A-Z_\-.]*$`)
)

type PWController struct {
	svc       session.SessionService
	transport http.RoundTripper
	eb        event.EventBroker
	now       clock.NowFunc
	l         *zap.SugaredLogger
}

type pwLaunchOptions struct {
	Args             []string       `json:"args,omitempty"`
	Headless         *bool          `json:"headless,omitempty"`
	Channel          string         `json:"channel,omitempty"`
	FirefoxUserPrefs map[string]any `json:"firefoxUserPrefs,omitempty"`
}

type pwOptions struct {
	Flavor     string
	Name       string
	Version    string
	LaunchOpts pwLaunchOptions
	VNCEnabled bool
	Resolution string
	Env        []string
	Links      []string
	Hosts      []string
	Networks   []string
	Labels     map[string]string
}

func NewPWController(
	svc session.SessionService,
	transport http.RoundTripper,
	eb event.EventBroker,
	now clock.NowFunc,
	l *zap.Logger,
) *PWController {
	return &PWController{
		svc:       svc,
		transport: transport,
		eb:        eb,
		now:       now,
		l:         l.Sugar(),
	}
}

func (p *PWController) CreateSession(c echo.Context) error {
	opts, err := parsePWOptions(c)
	ev := evmodels.SessionRequested{
		Protocol:       models.PlaywrightProtocol,
		BrowserName:    opts.Name,
		BrowserVersion: opts.Version,
	}
	if err != nil {
		ev.Error = models.NewBadRequestError(err)
		defer p.eb.Publish(evmodels.NewSessionRequestedEvent(ev))
		return ev.Error
	}

	sess, err := p.createSession(c, ev, opts)
	if err != nil {
		return err
	}

	defer func() {
		ev := evmodels.SessionReleased{
			Protocol:        models.PlaywrightProtocol,
			BrowserName:     opts.Name,
			BrowserVersion:  opts.Version,
			SessionDuration: p.now().Sub(sess.Created()),
		}
		p.eb.Publish(evmodels.NewSessionReleasedEvent(ev))
		p.svc.DeleteSession(sess)
	}()

	// XXX We need to use wrapped context here to allow resetting connections from UI
	// that's a bit awkward, need to think about resetting connections from Browser.Close()
	c.SetRequest(c.Request().Clone(sess.Context()))
	(&httputil.ReverseProxy{
		Transport: p.transport,
		Director: func(r *http.Request) {
			r.Host = sess.Browser().GetURL().Host
			r.URL = sess.Browser().GetURL()
			q := make(url.Values)
			if len(opts.LaunchOpts.Args) > 0 {
				q[PWArgParamQ] = opts.LaunchOpts.Args
			}
			if opts.LaunchOpts.Headless != nil {
				q.Set(PWHeadlessParamQ, strconv.FormatBool(*opts.LaunchOpts.Headless))
			}
			launchOptsVal, err := json.Marshal(opts.LaunchOpts)
			if err == nil { // actually always true
				q.Set(PWLaunchOptionsParamQ, string(launchOptsVal))
			}
			r.URL.RawQuery = q.Encode()
		},
		ErrorHandler: p.defaultErrorHandler(c.RealIP()),
	}).ServeHTTP(c.Response(), c.Request())

	return nil
}

func (s *PWController) ValidateSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param(router.SessionParam)
		i := strings.Index(c.Path(), fmt.Sprintf("/:%s", router.SessionParam))
		if i < 0 {
			s.l.Panicf("Middleware applied to the wrong route: %s", c.Path())
		}

		sess, err := s.svc.FindSession(id)
		if err != nil {
			return models.NewNotFoundError(err)
		}

		c.Set(SessionKey, sess)
		return next(c)
	}
}

func (p *PWController) createSession(c echo.Context, ev evmodels.SessionRequested, opts *pwOptions) (*session.Session, error) {
	defer func() {
		if r := recover(); r != nil {
			ev.Error = models.NewInternalServerError(errors.Errorf("panic: %v", r))
			p.eb.Publish(evmodels.NewSessionRequestedEvent(ev))
			panic(r)
		}
		p.eb.Publish(evmodels.NewSessionRequestedEvent(ev))
	}()

	caps := &models.PWCapabilities{
		Flavor:           opts.Flavor,
		Browser:          opts.Name,
		Version:          opts.Version,
		VNCEnabled:       opts.VNCEnabled,
		ScreenResolution: opts.Resolution,
		Env:              opts.Env,
		Links:            opts.Links,
		Hosts:            opts.Hosts,
		Networks:         opts.Networks,
		Labels:           opts.Labels,
	}

	start := p.now()
	sess, err := p.svc.CreateSession(c.Request().Context(), caps)
	if err != nil {
		p.l.Errorw("failed to create playwright session", zap.Error(err))
		ev.Error = models.WrapCancelledErr(err)
		return nil, ev.Error
	}
	ev.StartDuration = sess.Created().Sub(start)
	return sess, nil
}

func (p *PWController) defaultErrorHandler(remote string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		msg := fmt.Sprintf("proxy error %s->%v", remote, r.URL)
		p.l.Errorw(msg, zap.Error(err))
		w.WriteHeader(http.StatusBadGateway)
		if _, respErr := fmt.Fprintf(w, "%s: %v", msg, err); respErr != nil {
			p.l.Errorw("write error", zap.Error(respErr))
		}
	}
}

//nolint:gocyclo // does not make sense to split
func parsePWOptions(c echo.Context) (*pwOptions, error) {
	opts := &pwOptions{
		Flavor:  c.QueryParam(router.FlavorQParam),
		Name:    c.Param(router.NameParam),
		Version: c.Param(router.VersionParam),
	}

	if launchOptsVal := c.QueryParam(PWLaunchOptionsParamQ); launchOptsVal != "" {
		if err := json.Unmarshal([]byte(launchOptsVal), &opts.LaunchOpts); err != nil {
			return opts, errors.Wrap(err, "malformed launch-options parameter")
		}
		if err := validateLaunchOpts(opts.LaunchOpts); err != nil {
			return opts, errors.Wrap(err, "bad launch options")
		}
	}

	if args := c.QueryParams()[PWArgParamQ]; len(args) > 0 {
		opts.LaunchOpts.Args = append(opts.LaunchOpts.Args, args...)
	}

	if channel := c.QueryParam(PWChannelParamQ); channel != "" {
		opts.LaunchOpts.Channel = channel
	}

	if vnc := c.QueryParam(PWVncParamQ); vnc != "" {
		v, err := strconv.ParseBool(vnc)
		if err != nil {
			return opts, errors.Wrap(err, "bad vnc parameter")
		}
		opts.VNCEnabled = v
		opts.LaunchOpts.Headless = ref(!v)
	}

	if headless := c.QueryParam(PWHeadlessParamQ); headless != "" {
		h, err := strconv.ParseBool(headless)
		if err != nil {
			return opts, errors.Wrap(err, "bad headless parameter")
		}
		opts.LaunchOpts.Headless = ref(h)
		opts.VNCEnabled = !h
	}

	if res := c.QueryParam(PWResolutionParamQ); res != "" {
		if !resolutionRegex.MatchString(res) {
			return opts, errors.New("incorrect resolution parameter format (expected WIDTHxHEIGHTxBPP)")
		}
		opts.Resolution = res
	}

	if env := c.QueryParams()[PWEnvParamQ]; len(env) > 0 {
		if err := validatePlaywrightEnv(env); err != nil {
			return opts, errors.Wrap(err, "bad environment")
		}
		opts.Env = env
	}

	if links := c.QueryParams()[PWLinkParamQ]; len(links) > 0 {
		opts.Links = links
	}

	if hosts := c.QueryParams()[PWHostParamQ]; len(hosts) > 0 {
		opts.Hosts = hosts
	}

	if networks := c.QueryParams()[PWNetworkParamQ]; len(networks) > 0 {
		opts.Networks = networks
	}

	if labels := c.QueryParams()[PWLabelParamQ]; len(labels) > 0 {
		labelsMap, err := parsePlaywrightLabels(labels)
		if err != nil {
			return opts, errors.Wrap(err, "bad label")
		}
		opts.Labels = labelsMap
	}

	if prefs := c.QueryParams()[PWFirefoxUserPrefParamQ]; len(prefs) > 0 {
		ffUserPrefs, err := parseFirefoxUserPrefs(prefs)
		if err != nil {
			return opts, errors.Wrap(err, "bad firefoxUserPrefs")
		}
		if opts.LaunchOpts.FirefoxUserPrefs == nil {
			opts.LaunchOpts.FirefoxUserPrefs = make(map[string]any)
		}
		maps.Copy(opts.LaunchOpts.FirefoxUserPrefs, ffUserPrefs)
	}

	return opts, nil
}

func validateLaunchOpts(opts pwLaunchOptions) error {
	for k, v := range opts.FirefoxUserPrefs {
		switch v.(type) {
		case map[string]interface{}, []interface{}, nil:
			return errors.Errorf("invalid firefoxUserPref %s value, only primitive non null types allowed", k)
		default:
		}
	}
	return nil
}

func parseFirefoxUserPrefs(prefs []string) (map[string]any, error) {
	res := make(map[string]any)
	for _, p := range prefs {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return nil, errors.Errorf("malformed firefoxUserPref param (key=value expected): %s", p)
		}
		// since we don't have type information here let's guess. In case we want to set "123" to a string pref
		// we still have an option to provide `raw` launch-options query parameter
		i, err := strconv.ParseInt(kv[1], 10, 64)
		switch {
		case err == nil:
			res[kv[0]] = i
		case kv[1] == "true":
			res[kv[0]] = true
		case kv[1] == "false":
			res[kv[0]] = false
		default:
			res[kv[0]] = kv[1]
		}
	}
	return res, nil
}

func validatePlaywrightEnv(env []string) error {
	for _, e := range env {
		v := strings.SplitN(e, "=", 2)
		if len(v) != 2 {
			return errors.Errorf("malformed env param (key=value expected): %s", e)
		}

		if !pwEnvRegex.MatchString(v[0]) {
			return errors.Errorf("invalid env name (only PW_XXXX allowed): %s", v[0])
		}
	}
	return nil
}

func parsePlaywrightLabels(labels []string) (map[string]string, error) {
	res := make(map[string]string)
	for _, l := range labels {
		v := strings.SplitN(l, "=", 2)
		if len(v) < 2 {
			return nil, errors.Errorf("malformed label format '%s' (expected key=value)", l)
		}
		res[v[0]] = v[1]
	}
	return res, nil
}

func ref[T any](v T) *T {
	return &v
}
