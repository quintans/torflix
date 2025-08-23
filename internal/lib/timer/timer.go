package timer

import (
	"sync"
	"time"
)

// Timer executes a function after a delay, with cancellation and replacement.
type Timer struct {
	mu       sync.Mutex
	timer    *time.Timer
	deadline time.Time
	fn       func()
}

// New creates a new DelayedExecutor.
func New(delay time.Duration, fn func()) *Timer {
	exec := &Timer{
		deadline: time.Now().Add(delay),
		fn:       fn,
	}

	exec.timer = time.AfterFunc(delay, func() {
		// Always fetch the latest function under lock at execution time
		exec.mu.Lock()
		defer exec.mu.Unlock()

		exec.fn()
		exec.fn = nil // Clear the function to avoid re-execution
	})

	return exec
}

// Stop stops the scheduled execution.
func (d *Timer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
		d.fn = nil
	}
}

// ReplaceFn replaces the function. If the delay has already passed, run it immediately.
func (d *Timer) ReplaceFn(newFn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.fn == nil {
		// If the function is nil, it means the timer has already executed
		newFn()
		return
	}
	d.fn = newFn
}
