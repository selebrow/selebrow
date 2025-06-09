package models

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestNewEvent(t *testing.T) {
	g := NewWithT(t)
	tm := time.UnixMilli(123)
	e := NewEvent[string]("test", tm, "event1")
	g.Expect(e.EventType()).To(Equal("test"))
	g.Expect(e.EventTime()).To(Equal(tm))
	g.Expect(e.Attributes).To(Equal("event1"))
}
