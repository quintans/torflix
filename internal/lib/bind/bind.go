package bind

import (
	"slices"
	"sync"

	"fyne.io/fyne/v2"
)

type Setter[T any] interface {
	Set(T)
	SetAsync(T)
	Common[T]
}

type Notifier[T any] interface {
	Notify(T)
	NotifyAsync(T)
	Common[T]
}

type Common[T any] interface {
	Listen(func(T)) func()
	Bind(func(T)) func()
	UnbindAll()
	Get() T
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
	equal     func(T, T) bool
}

func New[T comparable](v T) *Bind[T] {
	return constructor(
		v,
		func(a, b T) bool {
			return a == b
		},
	)
}

func NewSlice[T comparable](v []T) *Bind[[]T] {
	return constructor(
		v,
		func(a, b []T) bool {
			return slices.Equal(a, b)
		},
	)
}

func NewSlicePtr[T comparable](v []*T) *Bind[[]*T] {
	return constructor(
		v,
		func(a, b []*T) bool {
			return slices.Equal(a, b)
		},
	)
}

func NewMap[K, V comparable](v map[K]V) *Bind[map[K]V] {
	return constructor(
		v,
		func(a, b map[K]V) bool {
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
	)
}

func NewWithEqual[T any](v T, equal func(T, T) bool) *Bind[T] {
	return constructor(v, equal)
}

func NewNotifier[T any]() *Bind[T] {
	return constructor(*new(T), nil)
}

func constructor[T any](v T, equal func(T, T) bool) *Bind[T] {
	return &Bind[T]{
		value: v,
		equal: equal,
	}
}

// Bind binds a handler and calls it immediately with the current value.
func (b *Bind[T]) Bind(h func(T)) func() {
	h(b.Get()) // call immediately with current value

	return b.Listen(h)
}

// Listen adds a handler to the list of listeners.
func (b *Bind[T]) Listen(h func(T)) func() {
	hh := &handle[T]{fn: h}
	b.listeners.Store(hh, struct{}{})

	return func() {
		b.listeners.Delete(hh)
	}
}

// Set sets the value and notifies all listeners.
// If the value as been set and is the same as the current value, it does nothing.
// If the equal function is nil, it does nothing.
func (b *Bind[T]) Set(value T) {
	b.mu.RLock()
	if b.equal == nil || b.equal(b.value, value) {
		b.mu.RUnlock()
		return
	}
	b.mu.RUnlock()

	b.Notify(value)
}

func (b *Bind[T]) SetAsync(value T) {
	fyne.DoAndWait(func() {
		b.Set(value)
	})
}

// Notify sets the value without and notifies all listeners.
func (b *Bind[T]) Notify(value T) {
	b.mu.Lock()
	b.value = value
	b.mu.Unlock()

	b.listeners.Range(func(k, _ any) bool {
		k.(handler[T]).handle(value)
		return true
	})
}

func (b *Bind[T]) NotifyAsync(value T) {
	fyne.DoAndWait(func() {
		b.Notify(value)
	})
}

// Get returns the current value.
func (b *Bind[T]) Get() T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.value
}

func (b *Bind[T]) UnbindAll() {
	b.listeners.Clear()
}
