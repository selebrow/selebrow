package browser

import (
	"context"

	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/models"
)

type BrowserManager interface {
	Allocate(ctx context.Context, protocol models.BrowserProtocol, caps capabilities.Capabilities) (Browser, error)
}
