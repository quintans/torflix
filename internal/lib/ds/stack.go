package ds

type Stack[T any] struct {
	tail *stackNode[T]
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

type stackNode[T any] struct {
	Value T
	Prev  *stackNode[T]
}

func (n *Stack[T]) Push(c T) {
	node := &stackNode[T]{
		Value: c,
		Prev:  n.tail,
	}
	n.tail = node
}

func (n *Stack[T]) Pop() (T, bool) {
	if n.tail == nil {
		var zero T
		return zero, false
	}
	node := n.tail
	n.tail = node.Prev
	return node.Value, true
}

func (n *Stack[T]) Reset() {
	n.tail = nil
}
