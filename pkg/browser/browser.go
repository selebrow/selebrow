package browser

import (
	"context"
	"net/url"

	"github.com/selebrow/selebrow/pkg/models"
)

const DefaultPlatform = "LINUX"

type Browser interface {
	GetURL() *url.URL
	GetHost() string
	GetHostPort(name models.ContainerPort) string
	Close(ctx context.Context, trash bool)
}
