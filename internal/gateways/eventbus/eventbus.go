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
