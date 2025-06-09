package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/distribution/reference"
	imagetypes "github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/registry"
)

// Below code is partially copied from docker/cli code with minor changes

// ImagePull Pull image with automatic registry auth (when required)
func (c *DockerClientImpl) ImagePull(ctx context.Context, image string) error {
	encodedAuth, err := c.retrieveAuthTokenFromImage(image)
	if err != nil {
		return err
	}

	resp, err := c.dockerCli.ImagePull(ctx, image, imagetypes.PullOptions{
		RegistryAuth: encodedAuth,
		Platform:     c.imagePlatform(),
	})
	if err != nil {
		return err
	}
	defer resp.Close()
	_, _ = io.Copy(io.Discard, resp)
	return ctx.Err()
}

// ImageInspect Inspect image with default options
func (c *DockerClientImpl) ImageInspect(ctx context.Context, image string) (imagetypes.InspectResponse, error) {
	return c.dockerCli.ImageInspect(ctx, image)
}

func (c *DockerClientImpl) imagePlatform() string {
	if c.platform == "" && c.arch == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", c.platform, c.arch)
}

func (c *DockerClientImpl) retrieveAuthTokenFromImage(image string) (string, error) {
	// Retrieve encoded auth token from the image reference
	authConfig, err := c.resolveAuthConfigFromImage(image)
	if err != nil {
		return "", err
	}
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return "", err
	}
	return encodedAuth, nil
}

// resolveAuthConfigFromImage retrieves that AuthConfig using the image string
func (c *DockerClientImpl) resolveAuthConfigFromImage(image string) (registrytypes.AuthConfig, error) {
	registryRef, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return registrytypes.AuthConfig{}, err
	}
	repoInfo, err := registry.ParseRepositoryInfo(registryRef)
	if err != nil {
		return registrytypes.AuthConfig{}, err
	}
	return c.resolveAuthConfig(repoInfo.Index), nil
}

func (c *DockerClientImpl) resolveAuthConfig(index *registrytypes.IndexInfo) registrytypes.AuthConfig {
	configKey := index.Name
	if index.Official {
		configKey = registry.IndexServer
	}

	a, _ := c.configFile.GetAuthConfig(configKey)
	return registrytypes.AuthConfig(a)
}
