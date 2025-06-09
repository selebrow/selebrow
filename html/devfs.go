package html

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/selebrow/selebrow/pkg/config"
)

var (
	devFS   fs.FS
	devMode bool
)

func init() {
	_, devMode = os.LookupEnv(fmt.Sprintf("_%s_DEV_MODE", config.ConfigPrefix))

	if devMode {
		devFS = os.DirFS("html")
	}
}
