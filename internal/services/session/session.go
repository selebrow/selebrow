package session

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/capabilities"
)

type Session struct {
	mu       sync.RWMutex
	id       string
	platform string
	br       browser.Browser
	reqCaps  capabilities.Capabilities
	resp     map[string]interface{}
	created  time.Time
	lastUsed time.Time
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewSession(
	id string,
	platform string,
	br browser.Browser,
	reqCaps capabilities.Capabilities,
	resp map[string]interface{},
	created time.Time,
	ctx context.Context,
	cancel context.CancelFunc,
) *Session {
	return &Session{
		id:       id,
		platform: platform,
		br:       br,
		reqCaps:  reqCaps,
		resp:     resp,
		created:  created,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Platform() string {
	return s.platform
}

func (s *Session) Browser() browser.Browser {
	return s.br
}

func (s *Session) ReqCaps() capabilities.Capabilities {
	return s.reqCaps
}

func (s *Session) Resp() map[string]interface{} {
	return maps.Clone(s.resp)
}

func (s *Session) Created() time.Time {
	return s.created
}

func (s *Session) Context() context.Context {
	return s.ctx
}

func (s *Session) Cancel() context.CancelFunc {
	return s.cancel
}

func (s *Session) LastUsed() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUsed
}

func (s *Session) SetLastUsed(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsed = t
}
