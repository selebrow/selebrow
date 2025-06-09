package html

import (
	"embed"
	"io/fs"
)

const StaticFSRoot = "static/"

//go:embed static
var staticEmbedFS embed.FS

func StaticFS() fs.FS {
	if devMode {
		return devFS
	}
	return staticEmbedFS
}
