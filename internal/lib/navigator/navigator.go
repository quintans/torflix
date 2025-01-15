package navigator

import (
	"github.com/quintans/torflix/internal/lib/bus"
	"github.com/quintans/torflix/internal/lib/ds"
)

type To struct {
	Target string
	Back   bool
}

func (To) Kind() string {
	return "To"
}

type Nav struct {
	last  string
	stack *ds.Stack[string]
	bus   *bus.Bus
}

func New(b *bus.Bus) *Nav {
	return &Nav{
		stack: ds.NewStack[string](),
		bus:   b,
	}
}

func (n *Nav) Go(view string) {
	if n.last != "" {
		n.stack.Push(n.last)
	}

	n.last = view
	n.bus.Publish(To{Target: view})
}

func (n *Nav) Back() {
	view, ok := n.stack.Pop()
	if !ok {
		return
	}

	n.last = view
	n.bus.Publish(To{Target: view, Back: true})
}
