package docker

import (
	"context"
	"io"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/moby/api/pkg/authconfig"
	imagetypes "github.com/moby/moby/api/types/image"
	registrytypes "github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	defaultRegistryHost = "docker.io"
	dockerIndexServer   = "https://index.docker.io/v1/"
)

// Below code is partially copied from docker/cli code with minor changes

// ImagePull Pull image with automatic registry auth (when required)
func (c *DockerClientImpl) ImagePull(ctx context.Context, image string) error {
	encodedAuth, err := c.retrieveAuthTokenFromImage(image)
	if err != nil {
		return err
	}

	resp, err := c.dockerCli.ImagePull(ctx, image, client.ImagePullOptions{
		RegistryAuth: encodedAuth,
		Platforms:    c.imagePlatforms(),
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
	res, err := c.dockerCli.ImageInspect(ctx, image)
	if err != nil {
		return imagetypes.InspectResponse{}, err
	}
	return res.InspectResponse, nil
}

func (c *DockerClientImpl) imagePlatforms() []ocispec.Platform {
	if c.platform == "" && c.arch == "" {
		return nil
	}
	return []ocispec.Platform{{
		Architecture: c.arch,
		OS:           c.platform,
	}}
}

func (c *DockerClientImpl) retrieveAuthTokenFromImage(image string) (string, error) {
	// Retrieve encoded auth token from the image reference
	authConfig, err := c.resolveAuthConfigFromImage(image)
	if err != nil {
		return "", err
	}
	return authconfig.Encode(authConfig)
}

// resolveAuthConfigFromImage retrieves that AuthConfig using the image string
func (c *DockerClientImpl) resolveAuthConfigFromImage(image string) (registrytypes.AuthConfig, error) {
	registryRef, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return registrytypes.AuthConfig{}, err
	}

	registryHost := reference.Domain(registryRef)
	if registryHost == "" || registryHost == defaultRegistryHost || registryHost == "index.docker.io" {
		return c.resolveAuthConfig(dockerIndexServer), nil
	}

	return c.resolveAuthConfig(registryHost), nil
}

func (c *DockerClientImpl) resolveAuthConfig(configKey string) registrytypes.AuthConfig {
	authConfig, _ := c.configFile.GetAuthConfig(configKey)

	if authConfig == (types.AuthConfig{}) && configKey == dockerIndexServer {
		authConfig, _ = c.configFile.GetAuthConfig(defaultRegistryHost)
	}

	return registrytypes.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}
}
