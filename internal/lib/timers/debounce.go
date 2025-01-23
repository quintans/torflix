package timers

import (
	"sync"
	"time"
)

type Debounce struct {
	mu    sync.Mutex
	next  time.Time
	close func()
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
	}

	go func() {
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
	}()

	return d
}

func (d *Debounce) Delay(delay time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.next = time.Now().Add(delay)
}

func (d *Debounce) Stop() {
	d.close()
}
