package docker

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/pkg/errors"
)

// Below code is copied from docker/cli with minor changes

const (
	envOverrideConfigDir = "DOCKER_CONFIG"

	configFileName = "config.json"
	configFileDir  = ".docker"

	contextsDir = "contexts"
)

// Load reads the configuration file ([configFileName]) from the given directory.
// If no directory is given, it uses the default [dir]. A [*configfile.ConfigFile]
// is returned containing the contents of the configuration file, or a default
// struct if no configfile exists in the given location.
//
// Load returns an error if a configuration file exists in the given location,
// but cannot be read, or is malformed. Consumers must handle errors to prevent
// overwriting an existing configuration file.
func Load(configDir string) (*configfile.ConfigFile, error) {
	if configDir == "" {
		configDir = dir()
	}
	return load(configDir)
}

func load(configDir string) (*configfile.ConfigFile, error) {
	filename := filepath.Join(configDir, configFileName)
	configFile := configfile.New(filename)

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// It is OK for no configuration file to be present, in which
			// case we return a default struct.
			return configFile, nil
		}
		// Any other error happening when failing to read the file must be returned.
		return configFile, errors.Wrap(err, "loading config file")
	}
	defer file.Close()
	err = configFile.LoadFromReader(file)
	if err != nil {
		err = errors.Wrapf(err, "parsing config file (%s)", filename)
	}
	return configFile, err
}

// dir returns the directory the configuration file is stored in
func dir() string {
	configDir := os.Getenv(envOverrideConfigDir)
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, configFileDir)
	}
	return configDir
}
