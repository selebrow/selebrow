package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
)

func ParseCmdLine(f *pflag.FlagSet, args []string) (*pflag.FlagSet, bool, error) {
	help := f.BoolP("help", "h", false, "Show usage help")
	f.String(listen, "", "Listening address and/or port, default is "+
		fmt.Sprintf("%s when run in Kubernetes or Docker container and %s otherwise", DefaultListen, DefaultLocalListen))
	f.Bool(ui, uiDefault(), "Enable UI (disabled by default, when run in CI)")
	f.String(backend, string(BackendAuto), "Backend to use, valid options are: "+validBackendsHelp)

	f.String(namespace, "default", "Namespace for pods (kubernetes backend only)")
	f.Bool(kubeClusterModeOut, false, "Out of cluster mode (for debug purposes, kubernetes backend only)")
	f.String(kubeConfig, kubeConfigDefault(), "Kube config file location (kubernetes backend only"+
		" when --"+kubeClusterModeOut+" is set to true)")
	f.String(kubeTemplatesPath, defaultConfigPath, "Path to pod templates directory (kubernetes backend only)")

	f.Duration(createTimeout, 3*time.Minute, "Timeout for create session requests")
	f.Int(createRetries, 5, "Number of retries on transient errors, when creating browser pods (kubernetes backend only)")
	f.Duration(connectTimeout, 200*time.Millisecond, "Browser connection timeout")
	f.String(browsersURI, defaultBrowsersURI, "Path or URL to browsers YAML config file")
	f.String(fallbackBrowsersURI, DefaultFallbackBrowsersURI, "Fallback path or URL to browsers YAML config file"+
		" in case --"+browsersURI+" is not available")

	f.Int(poolMaxIdle, 5, "Maximum number of idle browsers in the pool (pool is disabled if set to zero)")
	f.Duration(poolIdleTimeout, 1*time.Minute, "Timeout idle browsers in the pool")
	f.Duration(poolMaxAge, 15*time.Minute, "Maximum browser age before it's evicted from the pool")

	f.String(dockerNetwork, "", "Docker network for browser containers (docker backend only)")
	f.Bool(dockerPrivileged, false, "Run browser docker containers in privileged mode (docker backend only)")
	f.Bool(dockerPullImages, false, "Pre-pull browser docker images on startup (docker backend only)")
	f.String(dockerPortMapping, string(PortMappingAuto), "Docker port mapping mode, valid options are: "+validPortMappingModesHelp)
	f.String(dockerPlatform, "", "Docker platform for browser containers/images, e.g. linux/amd64 (docker backend only)")

	f.Int(quotaLimit, 0, "Limit for simultaneously running browser containers/pods, "+
		"0 (default) - automatically calculate limit based on available resources, -1 to disable quota")
	f.Int(queueSize, 25, "Queue size for requests waiting for available quota, if set to 0, queue is disabled")
	f.Duration(queueTimeout, time.Minute, "Timeout to wait for available quota (when queue is enabled)")

	f.String(vncPassword, DefaultVNCPassword, "VNC password to be used when connecting to VNC via UI")

	if err := f.Parse(args); err != nil {
		return nil, true, err
	}
	if *help {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		f.PrintDefaults()
		return nil, true, nil
	}

	return f, false, nil
}

func uiDefault() bool {
	_, ok := os.LookupEnv("CI")
	// disable UI under CI by default
	return !ok
}

func kubeConfigDefault() string {
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}
