package docker

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/selebrow/selebrow/internal/netutils"
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/browsers"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/docker"
	"github.com/selebrow/selebrow/pkg/models"
)

const (
	portMappingWaitInitialInterval = 100 * time.Millisecond
	portMappingWaitBackoffFactor   = 2
	portMappingWaitBackoffSteps    = 5
)

var (
	LocalIPs = netutils.LocalIPs

	ErrNetworkNotConnected = errors.New("container is not connected to configured network")
	ErrIPUnknown           = errors.New("couldn't detect container ip address within configured network")
)

type DockerBrowserManagerOpts struct {
	Network    container.NetworkMode
	MapPorts   bool
	Privileged bool
	PullImages bool
}

type DockerBrowserManager struct {
	client docker.DockerClient
	cat    browsers.BrowsersCatalog
	opts   DockerBrowserManagerOpts
	host   string
	l      *zap.SugaredLogger
}

func NewDockerBrowserManager(
	client docker.DockerClient,
	cat browsers.BrowsersCatalog,
	opts DockerBrowserManagerOpts,
	l *zap.Logger,
) (*DockerBrowserManager, error) {
	logger := l.Sugar()
	if opts.PullImages {
		if err := pullImages(client, cat, logger); err != nil {
			return nil, errors.Wrap(err, "failed to pull images")
		}
	}

	var host string
	if opts.MapPorts {
		host = client.GetHost()
		logger.Infof("running in port mapping mode via %s", host)
	} else if opts.Network == "" {
		nw, err := detectNetwork(client, logger)
		if err != nil {
			return nil, errors.Wrap(err,
				"failed to detect own docker network, consider specifying --docker-network argument")
		}
		opts.Network = nw
	}

	return &DockerBrowserManager{
		client: client,
		cat:    cat,
		opts:   opts,
		host:   host,
		l:      logger,
	}, nil
}

func pullImages(client docker.DockerClient, cat browsers.BrowsersCatalog, l *zap.SugaredLogger) error {
	l.Info("pulling images for configured browsers ...")
	images := cat.GetImages()
	for _, image := range images {
		_, err := client.ImageInspect(context.Background(), image)
		if err != nil {
			if errdefs.IsNotFound(err) {
				if err := pullImage(context.Background(), client, image, l); err != nil {
					return errors.Wrapf(err, "failed to pull image %s", image)
				}
				continue
			}
			return errors.Wrapf(err, "inspect failed for image %s", image)
		}
	}
	return nil
}

func detectNetwork(client docker.DockerClient, l *zap.SugaredLogger) (container.NetworkMode, error) {
	ips, err := LocalIPs()
	if err != nil {
		return "", errors.Wrap(err, "failed to collect local ips")
	}

	conts, err := client.ContainerList(context.Background())
	if err != nil {
		return "", errors.Wrap(err, "failed to list running containers")
	}

	for _, cont := range conts {
		if cont.NetworkSettings == nil || len(cont.NetworkSettings.Networks) == 0 {
			continue
		}
		for name, nw := range cont.NetworkSettings.Networks {
			if slices.Contains(ips, nw.IPAddress) {
				l.Infow("detected own docker network", zap.String("network", name))
				return container.NetworkMode(name), nil
			}
		}
	}
	return "", errors.New("unable to find container with any local assigned ip addresses")
}

func (m *DockerBrowserManager) Allocate(
	ctx context.Context,
	protocol models.BrowserProtocol,
	caps capabilities.Capabilities,
) (browser.Browser, error) {
	browserName, flavor := caps.GetName(), caps.GetFlavor()

	verCfg, ok := m.cat.LookupBrowserImage(protocol, browserName, flavor)
	if !ok {
		return nil, models.NewBadRequestError(errors.Errorf("browser %s image flavor %s is not supported", browserName, flavor))
	}

	id, err := m.createContainer(ctx, verCfg, caps)
	if err != nil {
		return nil, err
	}

	info, err := m.startContainer(ctx, id)
	if err != nil {
		m.removeContainer(context.Background(), id)
		return nil, err
	}

	br, err := m.createBrowser(verCfg, caps.IsVNCEnabled(), info)
	if err != nil {
		m.removeContainer(context.Background(), id)
		return nil, err
	}

	return br, nil
}

func pullImage(ctx context.Context, client docker.DockerClient, image string, l *zap.SugaredLogger) error {
	l = l.With(zap.String("image", image))
	l.Info("pulling image")
	start := time.Now()
	if err := client.ImagePull(ctx, image); err != nil {
		return err
	}
	l.Infow("image pull completed", zap.Duration("duration", time.Since(start)))
	return nil
}

func (m *DockerBrowserManager) createContainer(
	ctx context.Context,
	cfg models.BrowserImageConfig,
	caps capabilities.Capabilities,
) (string, error) {
	version := caps.GetVersion()
	tag, ok := cfg.GetTag(version)
	if !ok {
		return "", models.NewBadRequestError(errors.Errorf("image tag is missing for version %s", version))
	}

	image := fmt.Sprintf("%s:%s", cfg.Image, tag)
	ports := cfg.GetPorts(caps.IsVNCEnabled())

	config := &container.Config{
		ExposedPorts: getExposedPorts(ports),
		Env:          getEnv(cfg.Env, caps),
		Cmd:          slices.Clone(cfg.Cmd),
		Image:        image,
		Labels:       getLabels(cfg.Labels, caps.GetLabels()),
	}

	var portMap nat.PortMap
	if m.opts.MapPorts {
		portMap = getPortBindings(ports)
	}

	hostConfig := &container.HostConfig{
		Binds:        cfg.Volumes,
		NetworkMode:  m.opts.Network,
		PortBindings: portMap,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyDisabled,
		},
		ExtraHosts: caps.GetHosts(),
		Links:      caps.GetLinks(),
		Privileged: m.opts.Privileged,
		Tmpfs:      getTmpfs(cfg.Tmpfs),
		ShmSize:    cfg.ShmSize,
		Resources:  getResources(cfg.Limits),
	}

	networkingConfig := &network.NetworkingConfig{}

	created, err := m.doCreateContainer(ctx, config, hostConfig, networkingConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to create container")
	}

	return created.ID, nil
}

func (m *DockerBrowserManager) doCreateContainer(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
) (container.CreateResponse, error) {
	created, err := m.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, "")
	if err != nil {
		if errdefs.IsNotFound(err) {
			// pull image (if pre pull was disabled at startup)
			errCh := make(chan error, 1)
			go func() {
				// we are using context.Background here to avoid pull cancel if client is not patient enough
				// in this case it will continue in background
				// it's safe to pull the same image from different requests (docker does proper locking internally)
				errCh <- pullImage(context.Background(), m.client, config.Image, m.l)
			}()
			select {
			case <-ctx.Done():
				return container.CreateResponse{}, errors.Wrapf(ctx.Err(), "cancelled while image pull is still in progress")
			case err := <-errCh:
				if err != nil {
					return container.CreateResponse{}, errors.Wrapf(err, "failed to pull image %s", config.Image)
				}
			}
			return m.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, "")
		}

		return container.CreateResponse{}, err
	}
	return created, nil
}

func (m *DockerBrowserManager) startContainer(ctx context.Context, id string) (*container.InspectResponse, error) {
	l := m.l.With(zap.String("container", id))
	if err := m.client.ContainerStart(ctx, id); err != nil {
		return nil, errors.Wrap(err, "failed to start container")
	}

	// wait till all port mappings are ready (it happens asynchronously at least on macos/docker desktop)
	var backoff *wait.Backoff
	for {
		inspect, err := m.client.ContainerInspect(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err, "failed to inspect started container")
		}
		if m.opts.MapPorts && !allPortsMapped(inspect.NetworkSettings.Ports) {
			if backoff == nil {
				l.Info("waiting for port mappings to get ready...")
				backoff = &wait.Backoff{
					Duration: portMappingWaitInitialInterval,
					Factor:   portMappingWaitBackoffFactor,
					Steps:    portMappingWaitBackoffSteps,
				}
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff.Step()):
				continue
			}
		}
		if !inspect.State.Running {
			return nil, errors.Errorf("container state is %s", inspect.State.Status)
		}

		l.Infow("container started",
			zap.Any("ports", inspect.NetworkSettings.Ports))
		return &inspect, nil
	}
}

func (m *DockerBrowserManager) removeContainer(ctx context.Context, id string) {
	l := m.l.With(zap.String("container", id))
	err := m.client.ContainerRemove(ctx, id, true)
	if err != nil {
		l.Errorw("failed to remove container", zap.Error(err))
	} else {
		l.Info("container has been removed")
	}
}

func (m *DockerBrowserManager) createBrowser(
	cfg models.BrowserImageConfig,
	vncEnabled bool,
	info *container.InspectResponse,
) (*dockerBrowser, error) {
	var (
		forwardedHost string
		ports         map[models.ContainerPort]int
	)

	containerIP := info.NetworkSettings.IPAddress
	if containerIP == "" {
		nw := m.getNetwork(info.NetworkSettings.Networks)
		if nw == nil {
			return nil, ErrNetworkNotConnected
		}
		containerIP = nw.IPAddress
	}
	if containerIP == "" {
		return nil, ErrIPUnknown
	}
	host := fmt.Sprintf("%s:%d", containerIP, cfg.Ports[models.BrowserPort])

	if !m.opts.MapPorts {
		forwardedHost = containerIP
		ports = maps.Clone(cfg.GetPorts(vncEnabled))
	} else {
		forwardedHost = m.host
		ports = make(map[models.ContainerPort]int)
		for name, p := range cfg.GetPorts(vncEnabled) {
			mappedPort, ok := getMappedPort(info.NetworkSettings.Ports, p)
			if !ok || mappedPort == 0 {
				return nil, errors.Errorf("failed to get container mapped port for %s port (%d)", name, p)
			}
			ports[name] = mappedPort
		}
	}
	u, err := url.Parse(fmt.Sprintf("http://%s:%d%s", forwardedHost, ports[models.BrowserPort], cfg.Path))
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct webdriver URL")
	}
	delete(ports, models.BrowserPort)

	return &dockerBrowser{
		forwardedHost: forwardedHost,
		u:             u,
		host:          host,
		ports:         ports,
		close: func(ctx context.Context) {
			m.removeContainer(ctx, info.ID)
		},
	}, nil
}

func (m *DockerBrowserManager) getNetwork(networks map[string]*network.EndpointSettings) *network.EndpointSettings {
	if len(networks) == 0 {
		return nil
	}

	name := string(m.opts.Network)
	// if network is not specified - return "first" connected
	if name == "" {
		nets := maps.Keys(networks)
		for _, n := range slices.Sorted(nets) {
			name = n
			break
		}
	}

	return networks[name]
}

func getLabels(cfgLabels map[string]string, capsLabels map[string]string) map[string]string {
	labels := make(map[string]string)
	maps.Copy(labels, cfgLabels)
	maps.Copy(labels, capsLabels)
	return labels
}

func getMappedPort(ports nat.PortMap, p int) (int, bool) {
	port := tcpPort(p)
	mappings, ok := ports[port]
	if !ok {
		return 0, false
	}
	if len(mappings) < 1 {
		return 0, false
	}
	v := strings.Split(mappings[0].HostPort, "/")
	mappedPort, err := strconv.Atoi(v[0])
	return mappedPort, err == nil
}

func getTmpfs(tmpfs []string) map[string]string {
	res := make(map[string]string)
	for _, t := range tmpfs {
		v := strings.SplitN(t, ":", 2)
		if len(v) < 2 {
			res[v[0]] = ""
		} else {
			res[v[0]] = v[1]
		}
	}
	return res
}

func getResources(limits map[string]string) container.Resources {
	return container.Resources{
		Memory:   getLimit(limits, "memory").Value(),
		NanoCPUs: getLimit(limits, "cpu").ScaledValue(resource.Nano),
	}
}

func getLimit(limits map[string]string, name string) *resource.Quantity {
	lim, ok := limits[name]
	if !ok {
		return resource.NewQuantity(0, resource.DecimalSI)
	}

	q := resource.MustParse(lim) // should not panic as we assume config validation on startup
	return &q
}

func getPortBindings(ports map[models.ContainerPort]int) nat.PortMap {
	res := make(nat.PortMap)
	for _, p := range ports {
		port := tcpPort(p)
		res[port] = []nat.PortBinding{ // dynamic binding
			{
				HostIP:   "",
				HostPort: "",
			},
		}
	}
	return res
}

func getExposedPorts(ports map[models.ContainerPort]int) nat.PortSet {
	res := make(nat.PortSet)
	for _, p := range ports {
		port := tcpPort(p)
		res[port] = struct{}{}
	}
	return res
}

func tcpPort(p int) nat.Port {
	return nat.Port(fmt.Sprintf("%d/tcp", p))
}

func getEnv(configEnv map[string]string, caps capabilities.Capabilities) []string {
	overrides := make(map[string]string)
	for _, e := range caps.GetEnvs() {
		var value string
		v := strings.SplitN(e, "=", 2)
		if len(v) > 1 {
			value = v[1]
		}
		overrides[v[0]] = value
	}

	overrides["ENABLE_VNC"] = strconv.FormatBool(caps.IsVNCEnabled())
	overrides["SCREEN_RESOLUTION"] = caps.GetResolution()

	combined := make(map[string]string)
	maps.Copy(combined, configEnv)
	maps.Copy(combined, overrides)

	keys := maps.Keys(combined)
	res := make([]string, len(combined))
	for i, k := range slices.Sorted(keys) {
		res[i] = fmt.Sprintf("%s=%s", k, combined[k])
	}
	return res
}

func allPortsMapped(ports nat.PortMap) bool {
	var mapped bool
	for _, p := range ports {
		if len(p) == 0 || p[0].HostPort == "" {
			return false
		}
		mapped = true
	}
	return mapped
}
