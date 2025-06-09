package app

import (
	"context"

	"github.com/selebrow/selebrow/internal/browser/docker"
	"github.com/selebrow/selebrow/pkg/browsers"
	"github.com/selebrow/selebrow/pkg/config"
	dockerclient "github.com/selebrow/selebrow/pkg/docker"
	"github.com/selebrow/selebrow/pkg/log"
	"github.com/selebrow/selebrow/pkg/quota"

	"github.com/docker/docker/api/types/container"
	dc "github.com/docker/docker/client"
	"go.uber.org/zap"
)

func InitDockerClientFunc(cfg config.Config) dockerclient.DockerClient {
	dockerConfig, err := dockerclient.Load("")
	if err != nil {
		InitLog.Fatalw("failed to load Docker config", zap.Error(err))
	}

	opts, err := dockerclient.ClientOptsFromDockerContext(dockerConfig)
	if err != nil {
		InitLog.Fatalw("failed to load client options from Docker context", zap.Error(err))
	}

	opts = append(opts, dc.WithAPIVersionNegotiation())
	dockerCli, err := dc.NewClientWithOpts(opts...)
	if err != nil {
		InitLog.Fatalw("failed to initialize Docker client, check your environment", zap.Error(err))
	}

	_, err = dockerCli.Ping(context.Background())
	if err != nil {
		InitLog.Fatalw("failed to ping Docker daemon", zap.Error(err))
	}

	return dockerclient.NewDockerClientImpl(dockerCli, cfg.DockerPlatform(), dockerConfig)
}

func InitDockerQuotaAuthorizerFunc(cfg config.Config, client dockerclient.DockerClient) quota.QuotaAuthorizer {
	var (
		cpu int
		mem int64
		err error
	)

	if cfg.QuotaLimit() == 0 {
		cpu, mem, err = client.AvailableResources(context.Background())
		if err != nil {
			InitLog.Error("failed to get docker info")
		}
	}

	return initLimitQuotaAuthorizer(cfg, cpu, mem)
}

func initDockerWebDriverManager(
	cfg config.Config,
	cli dockerclient.DockerClient,
	cat browsers.BrowsersCatalog,
) *docker.DockerBrowserManager {
	l := log.GetLogger().Named("docker")

	opts := docker.DockerBrowserManagerOpts{
		Network:    container.NetworkMode(cfg.DockerNetwork()),
		MapPorts:   portMappingEnabled(cfg),
		Privileged: cfg.DockerPrivileged(),
		PullImages: cfg.DockerPullImages(),
	}

	dwm, err := docker.NewDockerBrowserManager(cli, cat, opts, l.Named("manager"))
	if err != nil {
		InitLog.Fatalw("failed to initialize docker browser manager", zap.Error(err))
	}

	return dwm
}

func portMappingEnabled(cfg config.DockerConfig) bool {
	// when run inside docker we should be able to directly communicate with containers, no need to use port mappings
	return cfg.DockerPortMapping() == config.PortMappingEnabled ||
		(cfg.DockerPortMapping() == config.PortMappingAuto && !InDocker())
}
