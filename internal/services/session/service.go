package session

import (
	"context"

	"github.com/selebrow/selebrow/pkg/capabilities"
)

type SessionService interface {
	CreateSession(ctx context.Context, caps capabilities.Capabilities) (*Session, error)
	FindSession(id string) (*Session, error)
	ListSessions() []*Session
	DeleteSession(sess *Session)
}
