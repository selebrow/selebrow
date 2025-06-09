package models

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestNewSessionRequestedEvent(t *testing.T) {
	g := NewWithT(t)
	tm := time.UnixMilli(123)
	now = func() time.Time {
		return tm
	}

	se := SessionRequested{
		Protocol:       "testproto",
		BrowserName:    "test",
		BrowserVersion: "1.1",
		StartDuration:  time.Millisecond,
		Error:          nil,
	}

	e := NewSessionRequestedEvent(se)

	g.Expect(e.EventTime()).To(Equal(tm))
	g.Expect(e.EventType()).To(Equal(SessionRequestedEventType))
	g.Expect(e.Attributes).To(Equal(se))
}

func TestNewSessionReleasedEvent(t *testing.T) {
	g := NewWithT(t)
	tm := time.UnixMilli(222)
	now = func() time.Time {
		return tm
	}

	sr := SessionReleased{
		Protocol:        "testproto",
		BrowserName:     "test",
		BrowserVersion:  "1.1",
		SessionDuration: time.Minute,
	}

	e := NewSessionReleasedEvent(sr)

	g.Expect(e.EventTime()).To(Equal(tm))
	g.Expect(e.EventType()).To(Equal(SessionReleasedEventType))
	g.Expect(e.Attributes).To(Equal(sr))
}
