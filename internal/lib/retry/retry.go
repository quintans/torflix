package retry

import (
	"errors"
	"time"
)

type PermanentError struct {
	err error
}

func (e PermanentError) Error() string {
	return e.err.Error()
}

func (e PermanentError) Unwrap() error {
	return e.err
}

func NewPermanentError(err error) error {
	return PermanentError{err: err}
}

type Option func(*Options)

type Options struct {
	retries   int
	delayFunc func(attempt int, err error) time.Duration
}

// WithRetries sets the number of retries. If retries is 0, it will retry forever.
// If not set, it will use a default of 3 retries.
func WithRetries(retries int) Option {
	return func(o *Options) {
		o.retries = retries
	}
}

// WithDelay sets the delay between retries.
// If not set, it will use a default delay of 1 second.
func WithDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.delayFunc = func(attempt int, err error) time.Duration {
			return delay
		}
	}
}

// WithDelayFunc sets the delay function between retries.
// If not set, it will use a default delay of 1 second.
// If the function returns 0, it will stop retrying.
func WithDelayFunc(f func(retry int, err error) time.Duration) Option {
	return func(o *Options) {
		o.delayFunc = f
	}
}

func Do(f func() error, options ...Option) error {
	_, err := Do2[any](func() (any, error) {
		return nil, f()
	})
	return err
}

func Do2[T any](f func() (T, error), options ...Option) (T, error) {
	opts := &Options{
		retries: 3,
		delayFunc: func(int, error) time.Duration {
			return time.Second
		},
	}
	for _, o := range options {
		o(opts)
	}

	var err error
	var t T
	for i := 1; opts.retries == 0 || i <= opts.retries; i++ {
		t, err = f()
		if err == nil {
			return t, nil
		}

		var perr PermanentError
		if errors.As(err, &perr) {
			return t, perr.Unwrap()
		}

		if opts.retries == 0 || i <= opts.retries {
			delay := opts.delayFunc(i, err)
			if delay == 0 {
				return t, err
			}
			time.Sleep(delay)
		}
	}
	return t, err
}
