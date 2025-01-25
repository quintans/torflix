package retry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/quintans/torflix/internal/lib/retry"
	"github.com/stretchr/testify/assert"
)

func TestDo_Success(t *testing.T) {
	coount := 0
	err := retry.Do(func() error {
		coount++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, coount)
}

func TestDo_Failure(t *testing.T) {
	count := 0
	expectedErr := errors.New("failed")
	err := retry.Do(func() error {
		count++
		return expectedErr
	}, retry.WithRetries(1))
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 2, count)
}

func TestDo_PermanentError(t *testing.T) {
	count := 0
	expectedErr := errors.New("permanent error")
	err := retry.Do(func() error {
		count++
		return retry.NewPermanentError(expectedErr)
	}, retry.WithRetries(3))
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, count)
}

func TestDo2_Success(t *testing.T) {
	result, err := retry.Do2(func() (int, error) {
		return 42, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestDo2_Failure(t *testing.T) {
	count := 0
	expectedErr := errors.New("failed")
	_, err := retry.Do2(func() (int, error) {
		count++
		return 0, expectedErr
	}, retry.WithRetries(1))
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 2, count)
}

func TestDo2_PermanentError(t *testing.T) {
	count := 0
	expectedErr := errors.New("permanent error")
	_, err := retry.Do2(func() (int, error) {
		count++
		return 0, retry.NewPermanentError(expectedErr)
	}, retry.WithRetries(3))
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, count)
}

func TestDo_WithDelay(t *testing.T) {
	start := time.Now()
	err := retry.Do(func() error {
		return errors.New("failed")
	}, retry.WithRetries(2), retry.WithDelay(250*time.Millisecond))
	assert.Error(t, err)
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 500*time.Millisecond)
}

func TestDo_WithDelayFunc(t *testing.T) {
	start := time.Now()
	err := retry.Do(
		func() error {
			return errors.New("failed")
		},
		retry.WithRetries(1),
		retry.WithDelayFunc(func(attempt int, err error) time.Duration {
			return 200 * time.Millisecond
		}),
	)
	assert.Error(t, err)
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 200*time.Millisecond)
}

func TestDo_WithDelays(t *testing.T) {
	start := time.Now()
	delays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond}
	err := retry.Do(
		func() error {
			return errors.New("failed")
		},
		retry.WithDelays(delays...),
	)
	assert.Error(t, err)
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond)
}

func TestDo_WithInfiniteDelays(t *testing.T) {
	start := time.Now()
	delays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond}
	err := retry.Do(
		func() error {
			return errors.New("failed")
		},
		retry.WithInfiniteDelays(delays...),
		retry.WithRetries(3),
	)
	assert.Error(t, err)
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 500*time.Millisecond)
}
