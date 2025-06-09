package docker

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const (
	defaultPlatform = "linux"
)

type DockerClient interface {
	GetHost() string
	ImagePull(ctx context.Context, image string) error
	ImageInspect(ctx context.Context, image string) (imagetypes.InspectResponse, error)
	ContainerCreate(
		ctx context.Context,
		config *container.Config,
		hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig,
		containerName string,
	) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string) error
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
	ContainerRemove(ctx context.Context, containerID string, force bool) error
	ContainerList(ctx context.Context) ([]container.Summary, error)
	AvailableResources(ctx context.Context) (cpus int, memory int64, err error)
}

const defaultDockerHost = "127.0.0.1"

type DockerClientImpl struct {
	dockerCli  *client.Client
	platform   string
	arch       string
	configFile *configfile.ConfigFile
	host       string
}

func NewDockerClientImpl(dockerClient *client.Client, dockerPlatform string, configFile *configfile.ConfigFile) *DockerClientImpl {
	u, err := url.Parse(dockerClient.DaemonHost())
	if err != nil {
		panic(err) // technically could not happen, since docker cli will validate it first
	}

	var host string
	if u.Host == "" || u.Scheme == "unix" {
		host = defaultDockerHost
	} else {
		host = getHostOnly(u.Host)
	}

	platform, arch := parseDockerPlatform(dockerPlatform)

	return &DockerClientImpl{
		dockerCli:  dockerClient,
		platform:   platform,
		arch:       arch,
		configFile: configFile,
		host:       host,
	}
}

// GetHost get docker host from initialized cli (Host part of DOCKER_HOST)
func (c *DockerClientImpl) GetHost() string {
	return c.host
}

func (c *DockerClientImpl) AvailableResources(ctx context.Context) (cpus int, memory int64, err error) {
	info, err := c.dockerCli.Info(ctx)
	if err != nil {
		return
	}
	return info.NCPU, info.MemTotal, nil
}

func getHostOnly(hostPort string) string {
	// no port
	if strings.LastIndexByte(hostPort, ':') < 0 {
		return hostPort
	}
	hostOnly, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		// technically could not happen, since docker cli will validate it first
		panic(err)
	}
	return hostOnly
}

func parseDockerPlatform(dockerPlatform string) (platform string, arch string) {
	if dockerPlatform == "" {
		return
	}
	v := strings.Split(dockerPlatform, "/")
	if len(v) < 2 {
		platform = defaultPlatform
		arch = v[0]
		return
	}
	platform = v[0]
	arch = v[1]
	return
}
