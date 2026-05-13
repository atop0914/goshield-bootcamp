// Package backoff provides retry with configurable backoff strategies.
package backoff

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Strategy defines a backoff strategy.
type Strategy interface {
	// Next returns the next backoff duration. Returns -1 if retries are exhausted.
	Next(retry int) time.Duration
	// MaxRetries returns the maximum number of retries.
	MaxRetries() int
}

// ErrMaxRetriesExceeded is returned when the maximum number of retries is exceeded.
var ErrMaxRetriesExceeded = errors.New("max retries exceeded")

// FixedBackoff returns a fixed backoff duration.
type FixedBackoff struct {
	Interval   time.Duration
	MaxRetryCount int
}

func (b *FixedBackoff) Next(retry int) time.Duration {
	if retry >= b.MaxRetryCount {
		return -1
	}
	return b.Interval
}

func (b *FixedBackoff) MaxRetries() int {
	return b.MaxRetryCount
}

// ExponentialBackoff returns an exponentially increasing backoff duration.
type ExponentialBackoff struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxRetryCount   int
	Multiplier      float64
}

func (b *ExponentialBackoff) Next(retry int) time.Duration {
	if retry >= b.MaxRetryCount {
		return -1
	}

	interval := float64(b.InitialInterval) * math.Pow(b.getMultiplier(), float64(retry))
	if interval > float64(b.MaxInterval) {
		interval = float64(b.MaxInterval)
	}
	return time.Duration(interval)
}

func (b *ExponentialBackoff) MaxRetries() int {
	return b.MaxRetryCount
}

func (b *ExponentialBackoff) getMultiplier() float64 {
	if b.Multiplier <= 0 {
		return 2.0
	}
	return b.Multiplier
}

// ExponentialRandomBackoff returns an exponentially increasing backoff with jitter.
type ExponentialRandomBackoff struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxRetryCount   int
	Multiplier      float64
}

func (b *ExponentialRandomBackoff) Next(retry int) time.Duration {
	if retry >= b.MaxRetryCount {
		return -1
	}

	multiplier := b.Multiplier
	if multiplier <= 0 {
		multiplier = 2.0
	}

	interval := float64(b.InitialInterval) * math.Pow(multiplier, float64(retry))
	if interval > float64(b.MaxInterval) {
		interval = float64(b.MaxInterval)
	}

	// Add jitter: random value between 0 and interval
	jitter := rand.Float64() * interval
	return time.Duration(jitter)
}

func (b *ExponentialRandomBackoff) MaxRetries() int {
	return b.MaxRetryCount
}

// FibonacciBackoff returns a Fibonacci-based backoff duration.
type FibonacciBackoff struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxRetryCount   int
}

func (b *FibonacciBackoff) Next(retry int) time.Duration {
	if retry >= b.MaxRetryCount {
		return -1
	}

	fib := fibonacci(retry + 2) // +2 because fibonacci(1)=1, fibonacci(2)=1, fibonacci(3)=2...
	interval := time.Duration(int64(b.InitialInterval) * int64(fib))
	if interval > b.MaxInterval {
		interval = b.MaxInterval
	}
	return interval
}

func (b *FibonacciBackoff) MaxRetries() int {
	return b.MaxRetryCount
}

func fibonacci(n int) int {
	if n <= 0 {
		return 1
	}
	a, b := 0, 1
	for i := 1; i < n; i++ {
		a, b = b, a+b
	}
	return b
}

// RetryConfig holds the configuration for retry operations.
type RetryConfig struct {
	// Backoff is the backoff strategy to use.
	Backoff Strategy
	// RetryOn determines if the error should trigger a retry.
	// If nil, all errors trigger a retry.
	RetryOn func(err error) bool
	// OnRetry is called before each retry attempt.
	OnRetry func(retry int, err error)
}

// Execute executes the function with retry logic.
func Execute(ctx context.Context, config RetryConfig, fn func(ctx context.Context) (any, error)) (any, error) {
	var lastErr error

	for i := 0; ; i++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if config.RetryOn != nil && !config.RetryOn(err) {
			return nil, err
		}

		// Get next backoff duration
		backoff := config.Backoff.Next(i)
		if backoff < 0 {
			return nil, errors.Join(ErrMaxRetriesExceeded, lastErr)
		}

		// Call on retry callback
		if config.OnRetry != nil {
			config.OnRetry(i, err)
		}

		// Wait for backoff or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Continue to next retry
		}
	}
}
