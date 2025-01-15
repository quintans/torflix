package safe

import "sync"

type Safe[T any] struct {
	value T
	mu    sync.RWMutex
}

func New[T any](value T) *Safe[T] {
	return &Safe[T]{
		value: value,
	}
}

func (s *Safe[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (s *Safe[T]) Set(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
}

func (s *Safe[T]) Update(f func(T) T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = f(s.value)
}
