package docker

import (
	"context"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
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
	res, err := c.dockerCli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           config,
		HostConfig:       hostConfig,
		NetworkingConfig: networkingConfig,
		Platform:         platform,
		Name:             containerName,
	})
	if err != nil {
		return container.CreateResponse{}, err
	}

	return container.CreateResponse{
		ID:       res.ID,
		Warnings: res.Warnings,
	}, nil
}

// ContainerStart Start container with default options
func (c *DockerClientImpl) ContainerStart(ctx context.Context, containerID string) error {
	_, err := c.dockerCli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	return err
}

// ContainerInspect Inspect container (API call as is)
func (c *DockerClientImpl) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	res, err := c.dockerCli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, err
	}
	return res.Container, nil
}

// ContainerRemove Remove container with optional force flag
func (c *DockerClientImpl) ContainerRemove(ctx context.Context, containerID string, force bool) error {
	_, err := c.dockerCli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force: force,
	})
	return err
}

// ContainerList List all running containers
func (c *DockerClientImpl) ContainerList(ctx context.Context) ([]container.Summary, error) {
	res, err := c.dockerCli.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}
