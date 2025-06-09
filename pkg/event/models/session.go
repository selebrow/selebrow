package models

import (
	"time"

	"github.com/selebrow/selebrow/pkg/models"
)

const (
	SessionRequestedEventType = "SessionRequested"
	SessionReleasedEventType  = "SessionReleased"
)

type SessionRequested struct {
	Protocol       models.BrowserProtocol
	BrowserName    string
	BrowserVersion string
	StartDuration  time.Duration
	Error          error
}

type SessionReleased struct {
	Protocol        models.BrowserProtocol
	BrowserName     string
	BrowserVersion  string
	SessionDuration time.Duration
}

func NewSessionRequestedEvent(s SessionRequested) *Event[SessionRequested] {
	return NewEvent(SessionRequestedEventType, now(), s)
}

func NewSessionReleasedEvent(s SessionReleased) *Event[SessionReleased] {
	return NewEvent(SessionReleasedEventType, now(), s)
}
