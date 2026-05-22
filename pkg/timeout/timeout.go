// Package timeout provides timeout enforcement for operations.
//
// Example:
//
//	result, err := timeout.Execute(ctx, timeout.Config{
//	    Duration: 5 * time.Second,
//	}, func(ctx context.Context) (any, error) {
//	    return myService.Call(ctx)
//	})
package timeout

import (
	"context"
	"time"

	"github.com/atop0914/goshield/internal/timeout"
)

// ErrTimeout is returned when an operation times out.
var ErrTimeout = timeout.ErrTimeout

// Config holds the configuration for timeout operations.
type Config struct {
	// Duration is the maximum time to wait for an operation.
	Duration time.Duration
	// OnTimeout is called when an operation times out.
	OnTimeout func(duration time.Duration)
}

// Execute executes the function with a timeout.
func Execute(ctx context.Context, config Config, fn func(ctx context.Context) (any, error)) (any, error) {
	return timeout.Execute(ctx, timeout.Config{
		Duration:  config.Duration,
		OnTimeout: config.OnTimeout,
	}, fn)
}

// WithTimeout is a convenience function that wraps an operation with a timeout.
func WithTimeout(ctx context.Context, duration time.Duration, fn func(ctx context.Context) (any, error)) (any, error) {
	return Execute(ctx, Config{Duration: duration}, fn)
}
