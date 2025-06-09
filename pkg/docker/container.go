package docker

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContainerCreate Create container with specified arch/platform
func (c *DockerClientImpl) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	containerName string,
) (container.CreateResponse, error) {
	platform := &ocispec.Platform{
		Architecture: c.arch,
		OS:           c.platform,
	}
	return c.dockerCli.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

// ContainerStart Start container with default options
func (c *DockerClientImpl) ContainerStart(ctx context.Context, containerID string) error {
	return c.dockerCli.ContainerStart(ctx, containerID, container.StartOptions{})
}

// ContainerInspect Inspect container (API call as is)
func (c *DockerClientImpl) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return c.dockerCli.ContainerInspect(ctx, containerID)
}

// ContainerRemove Remove container with optional force flag
func (c *DockerClientImpl) ContainerRemove(ctx context.Context, containerID string, force bool) error {
	return c.dockerCli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: force,
	})
}

// ContainerList List all running containers
func (c *DockerClientImpl) ContainerList(ctx context.Context) ([]container.Summary, error) {
	return c.dockerCli.ContainerList(ctx, container.ListOptions{})
}
