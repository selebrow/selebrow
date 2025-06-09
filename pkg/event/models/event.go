package models

import "time"

var now = time.Now

type IEvent interface {
	EventTime() time.Time
	EventType() string
}

type Event[T any] struct {
	eventTime  time.Time
	eventType  string
	Attributes T
}

func (e *Event[T]) EventTime() time.Time {
	return e.eventTime
}

func (e *Event[T]) EventType() string {
	return e.eventType
}

func NewEvent[T any](eventType string, evTime time.Time, attributes T) *Event[T] {
	return &Event[T]{
		eventTime:  evTime,
		eventType:  eventType,
		Attributes: attributes,
	}
}
