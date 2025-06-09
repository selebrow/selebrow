package main

import (
	"github.com/selebrow/selebrow/pkg/app"
)

const appName = "selebrow"

var (
	GitSha = "unknown"
	GitRef = "unknown"
)

func main() {
	app.Run(GitRef, GitSha, appName)
}
