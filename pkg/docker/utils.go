package docker

import (
	"os"
)

const dockerEnvFile = "/.dockerenv"

var (
	inDocker bool
)

func init() {
	_, err := os.Stat(dockerEnvFile)
	inDocker = err == nil
}

// InDocker check if we started inside docker container (works only for genuine Docker runtime)
func InDocker() bool {
	return inDocker
}
