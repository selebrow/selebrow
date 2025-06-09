package event

import (
	"context"
	"testing"
	"time"

	"github.com/selebrow/selebrow/pkg/event/models"

	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

func TestEventBrokerImpl_Subscribe(t *testing.T) {
	g := NewWithT(t)
	b := NewEventBrokerImpl(1, zaptest.NewLogger(t))

	ch := b.Subscribe("test")

	ev1 := models.NewEvent("test", time.UnixMilli(111), "event1")
	b.Publish(ev1)
	ev2 := models.NewEvent("test", time.UnixMilli(122), "event2")
	b.Publish(ev2) // should be dropped

	var got models.IEvent
	g.Expect(ch).To(Receive(&got))
	g.Expect(got.(*models.Event[string])).To(Equal(ev1))
	g.Expect(ch).ToNot(Receive())

	err := b.ShutDown(context.TODO())

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ch).To(BeClosed())
}
