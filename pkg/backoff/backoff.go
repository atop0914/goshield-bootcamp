// Package backoff provides retry with configurable backoff strategies.
//
// Example:
//
//    result, err := backoff.Execute(ctx, backoff.RetryConfig{
//        Backoff: &backoff.ExponentialBackoff{
//            InitialInterval: 100 * time.Millisecond,
//            MaxInterval:     10 * time.Second,
//            MaxRetryCount:   5,
//        },
//        RetryOn: func(err error) bool {
//            return !errors.Is(err, context.Canceled)
//        },
//    }, func(ctx context.Context) (any, error) {
//        return myService.Call(ctx)
//    })
package backoff

import (
	"context"
	"time"

	"github.com/atop0914/goshield/internal/backoff"
)

// Strategy defines a backoff strategy.
type Strategy = backoff.Strategy

// FixedBackoff returns a fixed backoff duration.
type FixedBackoff = backoff.FixedBackoff

// ExponentialBackoff returns an exponentially increasing backoff duration.
type ExponentialBackoff = backoff.ExponentialBackoff

// ExponentialRandomBackoff returns an exponentially increasing backoff with jitter.
type ExponentialRandomBackoff = backoff.ExponentialRandomBackoff

// FibonacciBackoff returns a Fibonacci-based backoff duration.
type FibonacciBackoff = backoff.FibonacciBackoff

// RetryConfig holds the configuration for retry operations.
type RetryConfig = backoff.RetryConfig

// ErrMaxRetriesExceeded is returned when the maximum number of retries is exceeded.
var ErrMaxRetriesExceeded = backoff.ErrMaxRetriesExceeded

// Execute executes the function with retry logic.
func Execute(ctx context.Context, config RetryConfig, fn func(ctx context.Context) (any, error)) (any, error) {
	return backoff.Execute(ctx, config, fn)
}

// Convenience functions

// WithFixedRetry executes with a fixed backoff retry strategy.
func WithFixedRetry(ctx context.Context, interval time.Duration, maxRetries int, fn func(ctx context.Context) (any, error)) (any, error) {
	return Execute(ctx, RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      interval,
			MaxRetryCount: maxRetries,
		},
	}, fn)
}

// WithExponentialRetry executes with an exponential backoff retry strategy.
func WithExponentialRetry(ctx context.Context, initial, max time.Duration, maxRetries int, fn func(ctx context.Context) (any, error)) (any, error) {
	return Execute(ctx, RetryConfig{
		Backoff: &ExponentialBackoff{
			InitialInterval: initial,
			MaxInterval:     max,
			MaxRetryCount:   maxRetries,
		},
	}, fn)
}
