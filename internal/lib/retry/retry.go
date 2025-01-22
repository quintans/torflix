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
	attempts int
	delay    time.Duration
}

func WithAttempts(attempts int) Option {
	return func(o *Options) {
		o.attempts = attempts
	}
}

func WithDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.delay = delay
	}
}

func Do(f func() error, options ...Option) error {
	opts := &Options{
		attempts: 3,
		delay:    time.Second,
	}
	for _, o := range options {
		o(opts)
	}

	var err error
	for i := 0; i < opts.attempts; i++ {
		err = f()
		if err == nil {
			return nil
		}

		var perr PermanentError
		if errors.As(err, &perr) {
			return perr.Unwrap()
		}

		if i < opts.attempts-1 {
			time.Sleep(opts.delay)
		}
	}
	return err
}
