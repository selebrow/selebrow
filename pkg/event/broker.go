package event

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/selebrow/selebrow/pkg/event/models"
)

type EventBroker interface {
	Subscribe(eventTypes ...string) <-chan models.IEvent
	Publish(event models.IEvent)
}

type EventBrokerImpl struct {
	mtx   sync.RWMutex
	subs  map[string][]chan models.IEvent
	bSize int
	l     *zap.SugaredLogger
}

func NewEventBrokerImpl(bufferSize int, l *zap.Logger) *EventBrokerImpl {
	return &EventBrokerImpl{
		subs:  make(map[string][]chan models.IEvent),
		bSize: bufferSize,
		l:     l.Sugar(),
	}
}

func (b *EventBrokerImpl) Subscribe(eventTypes ...string) <-chan models.IEvent {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	ch := make(chan models.IEvent, b.bSize)
	for _, et := range eventTypes {
		b.subs[et] = append(b.subs[et], ch)
	}
	return ch
}

func (b *EventBrokerImpl) Publish(event models.IEvent) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	for _, ch := range b.subs[event.EventType()] {
		select {
		case ch <- event:
		default:
			b.l.With(zap.String("type", event.EventType())).
				Warnf("dropping published event, channel is full: length=%d", len(ch))
		}
	}
}

func (b *EventBrokerImpl) ShutDown(_ context.Context) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	closed := make(map[chan models.IEvent]bool)
	for et, chs := range b.subs {
		for _, ch := range chs {
			if !closed[ch] {
				close(ch)
				closed[ch] = true
			}
		}
		delete(b.subs, et)
	}
	b.l.Info("event broker shutdown completed")
	return nil
}
