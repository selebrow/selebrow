package docker

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/docker/client"
)

// Below code is based on docker/cli context implementation

const (
	defaultContextName = "default"
	envOverrideContext = "DOCKER_CONTEXT"
)

type DockerContext map[string]any

func ClientOptsFromDockerContext(cfg *configfile.ConfigFile) ([]client.Opt, error) {
	ctxName := getDockerContext(cfg)
	if ctxName == defaultContextName {
		return []client.Opt{client.FromEnv}, nil
	}

	s := store.New(contextStoreDir(), defaultContextStoreConfig())
	ctxMeta, err := s.GetMetadata(ctxName)
	if err != nil {
		return nil, err
	}

	epMeta, err := docker.EndpointFromContext(ctxMeta)
	if err != nil {
		return nil, err
	}

	ep, err := docker.WithTLSData(s, ctxName, epMeta)
	if err != nil {
		return nil, err
	}

	return ep.ClientOpts()
}

func getDockerContext(cfg *configfile.ConfigFile) string {
	if os.Getenv(client.EnvOverrideHost) != "" {
		return defaultContextName
	}

	if ctxName := os.Getenv(envOverrideContext); ctxName != "" {
		return ctxName
	}

	if cfg.CurrentContext != "" {
		return cfg.CurrentContext
	}

	return defaultContextName
}

func contextStoreDir() string {
	return filepath.Join(dir(), contextsDir)
}

func defaultContextStoreConfig() store.Config {
	var defaultStoreEndpoints = []store.NamedTypeGetter{
		store.EndpointTypeGetter(docker.DockerEndpoint, func() any { return &docker.EndpointMeta{} }),
	}

	return store.NewConfig(
		func() any { return &DockerContext{} },
		defaultStoreEndpoints...,
	)
}
