package https_test

import (
	"errors"
	"testing"
	"time"

	"github.com/quintans/torflix/internal/lib/fails"
	"github.com/quintans/torflix/internal/lib/https"
	"github.com/quintans/torflix/internal/lib/retry"
	"github.com/stretchr/testify/assert"
)

func TestDelayFunc(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want time.Duration
	}{
		{
			name: "no error",
			err:  nil,
			want: time.Duration(0),
		},
		{
			name: "no retry-after",
			err:  errors.New("no retry-after"),
			want: time.Second,
		},
		{
			name: "retry-after not a number",
			err:  fails.New("too many requests", "retry-after", "not a number"),
			want: time.Second,
		},
		{
			name: "retry-after",
			err:  fails.New("too many requests", "retry-after", "2"),
			want: 6 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			err := retry.Do(func() error {
				return tt.err
			}, retry.WithDelayFunc(https.DelayFunc))
			assert.ErrorIs(t, err, tt.err)
			elapsed := time.Since(start)
			assert.GreaterOrEqual(t, elapsed, tt.want)
		})
	}
}
