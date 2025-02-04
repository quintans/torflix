package bus

import (
	"sync"
)

type Handler[T Message] func(T)

type Message interface {
	Kind() string
}

type Bus struct {
	handlers      map[string]map[uint64]func(Message)
	nextHandlerID uint64
	mu            sync.Mutex
}

func New() *Bus {
	return &Bus{
		handlers:      make(map[string]map[uint64]func(Message)),
		nextHandlerID: 0,
	}
}

func Register[T Message](bus *Bus, handler func(T)) uint64 {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	var zero T
	kind := zero.Kind()
	if bus.handlers[kind] == nil {
		bus.handlers[kind] = make(map[uint64]func(Message))
	}

	id := bus.nextHandlerID
	bus.nextHandlerID++

	bus.handlers[kind][id] = func(m Message) {
		handler(m.(T))
	}

	return id
}

func (b *Bus) Unregister(kind string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if handlers, ok := b.handlers[kind]; ok {
		delete(handlers, id)
		if len(handlers) == 0 {
			delete(b.handlers, kind)
		}
	}
}

func (b *Bus) Publish(m Message) {
	b.mu.Lock()
	handlers := b.handlers[m.Kind()]
	b.mu.Unlock()

	for _, handler := range handlers {
		handler(m)
	}
}
