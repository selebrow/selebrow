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
)

var (
	ConfigPrefix       = "SB"
	DefaultVNCPassword = "selebrow"
)

type (
	BackendType     string
	PortMappingMode string
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

	quotaLimit   = "quota-limit"
	queueSize    = "queue-size"
	queueTimeout = "queue-timeout"

	ui          = "ui"
	vncPassword = "vnc-password"

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
	}

	QuotaConfig interface {
		QuotaLimit() int
		QueueSize() int
		QueueTimeout() time.Duration
	}

	Config interface {
		BrowserConfig
		WDSessionConfig
		KubeConfig
		CIConfig
		PoolConfig
		DockerConfig
		QuotaConfig
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
