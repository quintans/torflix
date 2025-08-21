package bind

import (
	"slices"
	"sync"
)

type Notifier[T any] interface {
	Bind(func(T)) func()
	Notify(T)
	Clear()
}

type handler[T any] interface {
	handle(value T)
}

type handle[T any] struct {
	fn func(T)
}

func (h *handle[T]) handle(value T) {
	h.fn(value)
}

type Bind[T any] struct {
	mu        sync.RWMutex
	listeners sync.Map // map[Handler[T]]struct{}
	value     T
	set       bool // indicates if the value has been set
	equal     func(T, T) bool
}

func New[T comparable]() *Bind[T] {
	return &Bind[T]{
		equal: func(a, b T) bool {
			return a == b
		},
	}
}

func NewSlice[T comparable]() *Bind[[]T] {
	return &Bind[[]T]{
		equal: func(a, b []T) bool {
			return slices.Equal(a, b)
		},
	}
}

func NewSlicePtr[T comparable]() *Bind[[]*T] {
	return &Bind[[]*T]{
		equal: func(a, b []*T) bool {
			return slices.Equal(a, b)
		},
	}
}

func NewMap[K, V comparable]() *Bind[map[K]V] {
	return &Bind[map[K]V]{
		equal: func(a, b map[K]V) bool {
			if len(a) != len(b) {
				return false
			}
			for k, v := range a {
				if bv, ok := b[k]; !ok || v != bv {
					return false
				}
			}
			return true
		},
	}
}

func NewWithEqual[T any](equal func(T, T) bool) *Bind[T] {
	return &Bind[T]{
		equal: equal,
	}
}

func NewNotifier[T any]() *Bind[T] {
	return &Bind[T]{}
}

func (b *Bind[T]) Bind(h func(T)) func() {
	hh := &handle[T]{fn: h}

	b.mu.RLock()
	if b.set {
		h(b.value) // call on bind
	}
	b.mu.RUnlock()

	b.listeners.Store(hh, struct{}{})

	return func() {
		b.listeners.Delete(hh)
	}
}

func (b *Bind[T]) Set(value T) {
	b.mu.RLock()
	if b.equal == nil || b.equal(b.value, value) {
		b.mu.RUnlock()
		return
	}
	b.mu.RUnlock()

	b.Notify(value)
}

func (b *Bind[T]) Notify(value T) {
	b.mu.Lock()
	b.value = value
	b.set = true // mark as set
	b.mu.Unlock()

	b.listeners.Range(func(k, _ any) bool {
		k.(handler[T]).handle(value)
		return true
	})
}

func (b *Bind[T]) Get() T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.value
}

func (b *Bind[T]) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.value = *new(T)
	b.set = false
	b.listeners.Clear()
}
