package quota

import "context"

type QuotaAuthorizer interface {
	Enabled() bool
	Reserve(ctx context.Context) error
	Release() int
	Limit() int
	Allocated() int
}
