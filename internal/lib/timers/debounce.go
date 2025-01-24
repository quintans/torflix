package timers

import (
	"sync"
	"time"
)

// Debounce is a timer that waits for a specified delay before executing a function.
type Debounce struct {
	mu    sync.Mutex
	next  time.Time
	close func()
	latch *Latch
}

func NewDebounce(delay time.Duration, fn func()) *Debounce {
	done := make(chan struct{})
	once := sync.Once{}
	close := func() {
		once.Do(func() {
			close(done)
		})
	}

	d := &Debounce{
		mu:    sync.Mutex{},
		next:  time.Now().Add(delay),
		close: close,
		latch: NewLatch(),
	}

	go func() {
		d.latch.Unlock()

		for {
			select {
			case <-d.latch.Wait():
			case <-done:
				return
			}

			for {
				d.mu.Lock()
				next := d.next
				d.next = time.Time{}
				d.mu.Unlock()

				if next.IsZero() {
					break
				}
				diff := time.Until(next)
				if diff > 0 {
					select {
					case <-time.After(diff):
					case <-done:
						return
					}
				} else {
					break
				}
			}

			fn()
			d.latch.Lock()
		}
	}()

	return d
}

// Delay resets the debounce timer to the specified delay.
// If the function has already been called, it will be called again after the new delay.
func (d *Debounce) Delay(delay time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.next = time.Now().Add(delay)
	d.latch.Unlock()
}

// Stop cancels the debounce timer.
func (d *Debounce) Stop() {
	d.close()
}

// Latch is a lock that can be waited on.
type Latch struct {
	mu     sync.Mutex
	locked bool
	done   chan struct{}
}

func NewLatch() *Latch {
	return &Latch{
		mu:     sync.Mutex{},
		locked: true,
		done:   make(chan struct{}),
	}
}

// Lock locks the latch.
// If the latch is already locked, it will have no effect.
func (l *Latch) Lock() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.locked {
		l.locked = true
		l.done = make(chan struct{})
	}
}

// Unlock unlocks the latch.
// If the latch is already unlocked, it will have no effect.
func (l *Latch) Unlock() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.locked {
		l.locked = false
		close(l.done)
	}
}

// Wait returns a channel that will be closed when the latch is unlocked.
func (l *Latch) Wait() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.done
}
