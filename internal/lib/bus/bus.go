package bus

type Handler[T Message] func(T)

type Message interface {
	Kind() string
}

type Bus struct {
	handlers map[string][]func(Message)
}

func New() *Bus {
	return &Bus{
		handlers: make(map[string][]func(Message)),
	}
}

func Listen[T Message](bus *Bus, handler func(T)) {
	var zero T
	kind := zero.Kind()
	bus.handlers[kind] = append(bus.handlers[kind], func(m Message) {
		handler(m.(T))
	})
}

func (b *Bus) Publish(m Message) {
	kind := m.Kind()
	for _, handler := range b.handlers[kind] {
		handler(m)
	}
}
