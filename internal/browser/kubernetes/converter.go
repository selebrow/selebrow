package kubernetes

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	core "k8s.io/api/core/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/config"
	"github.com/selebrow/selebrow/pkg/kubeapi"
	"github.com/selebrow/selebrow/pkg/models"
)

type (
	BrowserConverter interface {
		ToPod(cfg models.BrowserImageConfig, caps capabilities.Capabilities) (core.Pod, error)
	}

	BrowserConverterConfig interface {
		config.CIConfig
		Lineage() string
	}

	TemplatedBrowserConverter struct {
		tmpl          *template.Template
		vals          map[string]interface{}
		ciEnvironment ciContext
		lineage       string
		logger        *zap.SugaredLogger
	}

	templateContext struct {
		Browser       browserContext
		Options       optionsContext
		Values        map[string]interface{}
		CIEnvironment ciContext
	}

	browserContext struct {
		Image  string
		Cmd    []string
		Ports  map[string]interface{}
		Path   string
		Env    map[string]interface{}
		Limits map[string]interface{}
	}

	optionsContext struct {
		Env        map[string]interface{}
		VNCEnabled bool
		Resolution string
		Labels     map[string]interface{}
		Hosts      map[string]interface{}
	}

	ciContext struct {
		JobID            string
		ProjectNamespace string
		ProjectName      string
	}
)

func NewTemplatedBrowserConverter(
	cfg BrowserConverterConfig,
	tpl string,
	values []byte,
	logger *zap.Logger,
) (*TemplatedBrowserConverter, error) {
	vals := make(map[string]interface{})
	if len(values) > 0 {
		err := yaml.Unmarshal(values, &vals)
		if err != nil {
			return nil, err
		}
	}

	funcs := sprig.TxtFuncMap()
	funcs["toYaml"] = toYAML

	tmpl, err := template.New("pod").Funcs(funcs).Option("missingkey=error").Parse(tpl)
	if err != nil {
		return nil, err
	}

	return &TemplatedBrowserConverter{
		tmpl: tmpl,
		vals: vals,
		ciEnvironment: ciContext{
			JobID:            cfg.JobID(),
			ProjectNamespace: cfg.ProjectNamespace(),
			ProjectName:      cfg.ProjectName(),
		},
		lineage: cfg.Lineage(),
		logger:  logger.Sugar(),
	}, nil
}

func (t *TemplatedBrowserConverter) ToPod(cfg models.BrowserImageConfig, caps capabilities.Capabilities) (core.Pod, error) {
	tplCtx, err := t.buildTemplateContext(cfg, caps)
	if err != nil {
		return core.Pod{}, err
	}

	var sb strings.Builder
	if err := t.tmpl.Execute(&sb, tplCtx); err != nil {
		return core.Pod{}, errors.Wrap(err, "failed to render template")
	}

	var pod core.Pod
	if err := yamlutil.NewYAMLOrJSONDecoder(strings.NewReader(sb.String()), 200).Decode(&pod); err != nil {
		const msg = "failed to deserialize rendered manifest into pod"
		t.logger.Debugw(msg, "manifest", sb.String())
		return core.Pod{}, errors.Wrap(err, msg)
	}
	// Set mandatory labels, everything else is up to template
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[kubeapi.LineageLabel] = t.lineage

	return pod, nil
}

func (t *TemplatedBrowserConverter) buildTemplateContext(
	cfg models.BrowserImageConfig,
	caps capabilities.Capabilities,
) (templateContext, error) {
	brCtx, err := buildBrowserContext(cfg, caps.GetVersion(), caps.IsVNCEnabled())
	if err != nil {
		return templateContext{}, err
	}
	optsCtx := buildOptionsContext(caps)

	values := maps.Clone(t.vals)

	return templateContext{
		Browser:       brCtx,
		Options:       optsCtx,
		Values:        values,
		CIEnvironment: t.ciEnvironment,
	}, nil
}

func buildOptionsContext(caps capabilities.Capabilities) optionsContext {
	env := parseEnv(caps.GetEnvs())

	env["ENABLE_VNC"] = strconv.FormatBool(caps.IsVNCEnabled())
	env["SCREEN_RESOLUTION"] = caps.GetResolution()

	return optionsContext{
		Env:        env,
		VNCEnabled: caps.IsVNCEnabled(),
		Resolution: caps.GetResolution(),
		Labels:     getLabels(caps.GetLabels()),
		Hosts:      getHosts(caps.GetHosts()),
	}
}

func buildBrowserContext(cfg models.BrowserImageConfig, version string, vncEnabled bool) (browserContext, error) {
	tag, ok := cfg.GetTag(version)
	if !ok {
		return browserContext{}, models.NewBadRequestError(errors.Errorf("image tag is missing for version %s", version))
	}

	var brPorts = make(map[string]interface{})
	for k, v := range cfg.GetPorts(vncEnabled) {
		brPorts[string(k)] = v
	}

	brLimits := make(map[string]interface{}, len(cfg.Limits))
	for k, v := range cfg.Limits {
		brLimits[k] = v
	}

	env := make(map[string]interface{}, len(cfg.Env))
	for k, v := range cfg.Env {
		env[k] = v
	}

	return browserContext{
		Image:  fmt.Sprintf("%s:%s", cfg.Image, tag),
		Cmd:    slices.Clone(cfg.Cmd),
		Ports:  brPorts,
		Path:   cfg.Path,
		Env:    env,
		Limits: brLimits,
	}, nil
}

func toYAML(v interface{}) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

func parseEnv(env []string) map[string]interface{} {
	res := make(map[string]interface{})
	for _, e := range env {
		var value string
		v := strings.SplitN(e, "=", 2)
		if len(v) > 1 {
			value = v[1]
		}
		res[v[0]] = value
	}
	return res
}

func getLabels(labels map[string]string) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range labels {
		res[k] = v
	}
	return res
}

func getHosts(hosts []string) map[string]interface{} {
	ipMap := make(map[string][]string)
	for _, h := range hosts {
		v := strings.SplitN(h, ":", 2)
		if len(v) < 2 {
			continue
		}
		ipMap[v[1]] = append(ipMap[v[1]], v[0])
	}

	res := make(map[string]interface{})
	for k, v := range ipMap {
		res[k] = v
	}
	return res
}
