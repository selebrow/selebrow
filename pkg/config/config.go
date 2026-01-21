package config

import (
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ConfigPrefix       = "SB"
	DefaultVNCPassword = "selebrow"
)

type (
	BackendType     string
	PortMappingMode string

	ProxyHostFunc func() string
)

const (
	BackendAuto       BackendType = "auto"
	BackendKubernetes BackendType = "kubernetes"
	BackendDocker     BackendType = "docker"

	PortMappingAuto     PortMappingMode = "auto"
	PortMappingEnabled  PortMappingMode = "enabled"
	PortMappingDisabled PortMappingMode = "disabled"

	DefaultListen      = "0.0.0.0:4444"
	DefaultLocalListen = "127.0.0.1:4444"

	listen              = "listen"
	kubeConfig          = "kube-config"
	kubeClusterModeOut  = "kube-cluster-mode-out"
	namespace           = "namespace"
	createTimeout       = "create-timeout"
	createRetries       = "create-retries"
	connectTimeout      = "connect-timeout"
	poolMaxIdle         = "pool-max-idle"
	poolMaxAge          = "pool-max-age"
	poolIdleTimeout     = "pool-idle-timeout"
	kubeTemplatesPath   = "kube-templates-path"
	browsersURI         = "browsers-uri"
	fallbackBrowsersURI = "fallback-browsers-uri"
	backend             = "backend"
	dockerNetwork       = "docker-network"
	dockerPrivileged    = "docker-privileged"
	dockerPullImages    = "docker-pull-images"
	dockerPortMapping   = "docker-port-mapping"
	dockerPlatform      = "docker-platform"
	dockerEnv           = "docker-env"

	quotaLimit   = "quota-limit"
	queueSize    = "queue-size"
	queueTimeout = "queue-timeout"

	ui          = "ui"
	vncPassword = "vnc-password"

	proxyEnabled        = "proxy-enabled"
	proxyListen         = "proxy-listen"
	proxyAccessLogLevel = "proxy-access-log-level"
	proxyConnectTimeout = "proxy-connect-timeout"
	proxyResolveHost    = "proxy-resolve-host"
	proxyHost           = "proxy-host"
	noProxy             = "no-proxy"

	defaultConfigPath  = "config/"
	defaultBrowsersURI = defaultConfigPath + "browsers.yaml"
)

var (
	// set at build time
	DefaultFallbackBrowsersURI = ""

	envReplacer = strings.NewReplacer("-", "_")

	validBackends     = []BackendType{BackendAuto, BackendKubernetes, BackendDocker}
	validBackendsHelp = quoteStrings(validBackends)

	validPortMappingModes     = []PortMappingMode{PortMappingAuto, PortMappingEnabled, PortMappingDisabled}
	validPortMappingModesHelp = quoteStrings(validPortMappingModes)

	genLineage = uuid.NewString
)

type (
	BrowserConfig interface {
		CreateTimeout() time.Duration
		CreateRetries() int
		ConnectTimeout() time.Duration
	}

	WDSessionConfig interface {
		CreateTimeout() time.Duration
		ProxyDelete() bool
	}

	KubeConfig interface {
		Namespace() string
		KubeClusterModeOut() bool
		KubeConfig() string
		KubeTemplatesPath() string
	}

	CIConfig interface {
		JobID() string
		ProjectNamespace() string
		ProjectName() string
	}

	PoolConfig interface {
		MaxAge() time.Duration
		MaxIdle() int
		IdleTimeout() time.Duration
	}

	DockerConfig interface {
		DockerNetwork() string
		DockerPrivileged() bool
		DockerPullImages() bool
		DockerPortMapping() PortMappingMode
		DockerPlatform() string
		DockerEnv() map[string]string
	}

	QuotaConfig interface {
		QuotaLimit() int
		QueueSize() int
		QueueTimeout() time.Duration
	}

	ProxyOpts struct {
		ProxyHost string
		NoProxy   string
	}

	ProxyConfig interface {
		ProxyOpts(defaultProxyHostFn ProxyHostFunc) (*ProxyOpts, error)
		ProxyEnabled() bool
		ProxyListen() string
		ProxyAccessLogLevel() zapcore.Level
		ProxyConnectTimeout() time.Duration
		ProxyResolveHost() bool
	}

	Config interface {
		BrowserConfig
		WDSessionConfig
		KubeConfig
		CIConfig
		PoolConfig
		DockerConfig
		QuotaConfig
		ProxyConfig
		Listen() string
		Backend() BackendType
		BrowsersURI() []string
		Lineage() string
		UI() bool
		VNCPassword() string
	}

	ConfigViper struct {
		v                 *viper.Viper
		jobID             string
		projectNamespace  string
		projectName       string
		backend           BackendType
		dockerPortMapping PortMappingMode
		lineage           string
	}
)

func NewConfig(v *viper.Viper, f *pflag.FlagSet) (*ConfigViper, error) {
	if err := v.BindPFlags(f); err != nil {
		return nil, err
	}
	if err := bindEnvVars(v); err != nil {
		return nil, err
	}

	back := BackendType(strings.ToLower(v.GetString(backend)))
	if !slices.Contains(validBackends, back) {
		return nil, errors.Errorf("invalid backend parameter specified (%s), valid options are: %s", back, validBackendsHelp)
	}

	portMapping := PortMappingMode(strings.ToLower(v.GetString(dockerPortMapping)))
	if !slices.Contains(validPortMappingModes, portMapping) {
		return nil, errors.Errorf("invalid docker port mapping mode specified (%s), valid options are: %s",
			portMapping,
			validPortMappingModesHelp)
	}

	return &ConfigViper{
		v:                 v,
		jobID:             os.Getenv("CI_JOB_ID"),
		projectNamespace:  os.Getenv("CI_PROJECT_NAMESPACE"),
		projectName:       os.Getenv("CI_PROJECT_NAME"),
		backend:           back,
		dockerPortMapping: portMapping,
		lineage:           genLineage(),
	}, nil
}

func (c *ConfigViper) QuotaLimit() int {
	return c.v.GetInt(quotaLimit)
}

func (c *ConfigViper) QueueSize() int {
	return c.v.GetInt(queueSize)
}

func (c *ConfigViper) QueueTimeout() time.Duration {
	return c.v.GetDuration(queueTimeout)
}

func (c *ConfigViper) DockerNetwork() string {
	return c.v.GetString(dockerNetwork)
}

func (c *ConfigViper) DockerPrivileged() bool {
	return c.v.GetBool(dockerPrivileged)
}

func (c *ConfigViper) DockerPullImages() bool {
	return c.v.GetBool(dockerPullImages)
}

func (c *ConfigViper) DockerPortMapping() PortMappingMode {
	return c.dockerPortMapping
}

func (c *ConfigViper) DockerPlatform() string {
	return c.v.GetString(dockerPlatform)
}

func (c *ConfigViper) DockerEnv() map[string]string {
	envParams := c.v.GetStringSlice(dockerEnv)
	env := make(map[string]string, len(envParams))
	for _, param := range envParams {
		v := strings.SplitN(param, "=", 2)
		if len(v) == 2 {
			env[v[0]] = v[1]
		} else if len(v) == 1 {
			if val, ok := os.LookupEnv(v[0]); ok {
				env[v[0]] = val
			}
		}
	}
	return env
}

func (c *ConfigViper) Backend() BackendType {
	return c.backend
}

func (c *ConfigViper) Lineage() string {
	return c.lineage
}

func (c *ConfigViper) MaxAge() time.Duration {
	return c.v.GetDuration(poolMaxAge)
}

func (c *ConfigViper) MaxIdle() int {
	return c.v.GetInt(poolMaxIdle)
}

func (c *ConfigViper) IdleTimeout() time.Duration {
	return c.v.GetDuration(poolIdleTimeout)
}

func (c *ConfigViper) JobID() string {
	return c.jobID
}

func (c *ConfigViper) ProjectNamespace() string {
	return c.projectNamespace
}

func (c *ConfigViper) ProjectName() string {
	return c.projectName
}

func (c *ConfigViper) KubeTemplatesPath() string {
	return c.v.GetString(kubeTemplatesPath)
}

func (c *ConfigViper) BrowsersURI() []string {
	urls := []string{c.v.GetString(browsersURI)}
	if fallback := c.v.GetString(fallbackBrowsersURI); fallback != "" {
		urls = append(urls, fallback)
	}
	return urls
}

func (c *ConfigViper) CreateTimeout() time.Duration {
	return c.v.GetDuration(createTimeout)
}

func (c *ConfigViper) CreateRetries() int {
	return c.v.GetInt(createRetries)
}

func (c *ConfigViper) ConnectTimeout() time.Duration {
	return c.v.GetDuration(connectTimeout)
}

func (c *ConfigViper) ProxyDelete() bool {
	return c.MaxIdle() > 0
}

func (c *ConfigViper) Namespace() string {
	return c.v.GetString(namespace)
}

func (c *ConfigViper) KubeClusterModeOut() bool {
	return c.v.GetBool(kubeClusterModeOut)
}

func (c *ConfigViper) KubeConfig() string {
	return c.v.GetString(kubeConfig)
}

func (c *ConfigViper) Listen() string {
	return c.v.GetString(listen)
}

func (c *ConfigViper) UI() bool {
	return c.v.GetBool(ui)
}

func (c *ConfigViper) VNCPassword() string {
	return c.v.GetString(vncPassword)
}

func (c *ConfigViper) ProxyOpts(defaultProxyHostFn ProxyHostFunc) (*ProxyOpts, error) {
	var err error
	host := c.v.GetString(proxyHost)
	if host == "" {
		if !c.ProxyEnabled() {
			return nil, nil
		}
		host, err = c.defaultProxyHost(defaultProxyHostFn)
		if err != nil {
			return nil, err
		}
	}
	return &ProxyOpts{
		ProxyHost: host,
		NoProxy:   c.v.GetString(noProxy),
	}, nil
}

func (c *ConfigViper) defaultProxyHost(defaultProxyHostFn ProxyHostFunc) (string, error) {
	host := defaultProxyHostFn()
	if host == "" {
		return "", errors.New("can't determine proxy host")
	}
	if !strings.Contains(host, ":") {
		pListen := c.ProxyListen()
		i := strings.Index(pListen, ":")
		if i < 0 {
			return "", errors.Errorf("failed to get proxy port from listen spec: %s", pListen)
		}
		host += pListen[i:]
	}
	return host, nil
}

func (c *ConfigViper) ProxyEnabled() bool {
	return c.v.GetBool(proxyEnabled)
}

func (c *ConfigViper) ProxyListen() string {
	return c.v.GetString(proxyListen)
}

func (c *ConfigViper) ProxyAccessLogLevel() zapcore.Level {
	return ZapLogLevel(c.v.GetString(proxyAccessLogLevel), zapcore.WarnLevel)
}

func (c *ConfigViper) ProxyConnectTimeout() time.Duration {
	return c.v.GetDuration(proxyConnectTimeout)
}

func (c *ConfigViper) ProxyResolveHost() bool {
	return c.v.GetBool(proxyResolveHost)
}

func bindEnvVars(v *viper.Viper) error {
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(envReplacer)
	v.SetEnvPrefix(ConfigPrefix)

	// we want NAMESPACE to also work along with SB_NAMESPACE for backward compatibility
	return v.BindEnv(namespace, "NAMESPACE")
}

func quoteStrings[T ~string](vals []T) string {
	var sb strings.Builder
	for i, v := range vals {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteRune('"')
		sb.WriteString(string(v))
		sb.WriteRune('"')
	}
	return sb.String()
}

var logLevelMap = map[string]zapcore.Level{
	"debug": zap.DebugLevel,
	"info":  zap.InfoLevel,
	"warn":  zap.WarnLevel,
	"error": zap.ErrorLevel,
}

func ZapLogLevel(strLevel string, defaultLevel zapcore.Level) zapcore.Level {
	if lvl, ok := logLevelMap[strings.ToLower(strLevel)]; ok {
		return lvl
	}
	return defaultLevel
}
