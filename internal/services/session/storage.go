package session

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/selebrow/selebrow/pkg/models"
)

var ErrStorageShutdown = errors.New("session storage is shutdown")

type SessionStorage interface {
	Add(protocol models.BrowserProtocol, sess *Session) error
	Get(protocol models.BrowserProtocol, id string) (*Session, bool)
	List(protocol models.BrowserProtocol) []*Session
	Delete(protocol models.BrowserProtocol, id string) bool
	IsShutdown() bool
}

type LocalSessionStorage struct {
	sessions map[models.BrowserProtocol]map[string]*Session
	shutdown bool
	mtx      sync.RWMutex
	l        *zap.SugaredLogger
}

func NewLocalSessionStorage(l *zap.Logger) *LocalSessionStorage {
	return &LocalSessionStorage{
		sessions: make(map[models.BrowserProtocol]map[string]*Session),
		l:        l.Sugar(),
	}
}

func (s *LocalSessionStorage) Add(protocol models.BrowserProtocol, sess *Session) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.shutdown {
		return ErrStorageShutdown
	}
	ps := s.sessions[protocol]
	if ps == nil {
		ps = make(map[string]*Session)
		s.sessions[protocol] = ps
	}
	ps[sess.ID()] = sess
	return nil
}

func (s *LocalSessionStorage) Get(protocol models.BrowserProtocol, id string) (*Session, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	ps := s.sessions[protocol]
	if ps == nil {
		return nil, false
	}

	sess, ok := ps[id]
	return sess, ok
}

func (s *LocalSessionStorage) List(protocol models.BrowserProtocol) []*Session {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	ps := s.sessions[protocol]
	if ps == nil {
		return nil
	}

	res := make([]*Session, 0, len(ps))
	for _, sess := range ps {
		res = append(res, sess)
	}
	return res
}

func (s *LocalSessionStorage) Delete(protocol models.BrowserProtocol, id string) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	ps := s.sessions[protocol]
	if ps == nil {
		return false
	}

	if _, ok := ps[id]; !ok {
		return false
	}

	delete(ps, id)
	return true
}

func (s *LocalSessionStorage) IsShutdown() bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.shutdown
}

func (s *LocalSessionStorage) Shutdown(ctx context.Context) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.shutdown = true

	done := make(chan struct{})
	var wg sync.WaitGroup
	for p, ps := range s.sessions {
		s.l.Infof("session storage is shutting down, invalidating %d %s sessions", len(ps), p)
		for id, sess := range ps {
			wg.Add(1)
			go func(sess *Session) {
				defer wg.Done()
				sess.Browser().Close(ctx, true)
			}(sess)
			delete(ps, id)
		}
		delete(s.sessions, p)
	}

	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	return nil
}
