package eventbus

import (
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bus"
)

type EventBus struct {
	bus *bus.Bus
}

func New(bus *bus.Bus) *EventBus {
	return &EventBus{
		bus: bus,
	}
}

func (e *EventBus) Publish(msg app.Message) {
	e.bus.Publish(msg)
}

func (e *EventBus) Error(msg string, args ...interface{}) {
	e.bus.Publish(app.NewNotifyError(msg, args...))
}

func (e *EventBus) Warn(msg string, args ...interface{}) {
	e.bus.Publish(app.NewNotifyWarn(msg, args...))
}

func (e *EventBus) Success(msg string, args ...interface{}) {
	e.bus.Publish(app.NewNotifySuccess(msg, args...))
}

func (e *EventBus) Info(msg string, args ...interface{}) {
	e.bus.Publish(app.NewNotifyInfo(msg, args...))
}
