// Package timeout provides timeout enforcement for operations.
package timeout

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var (
	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("operation timed out")
)

// Config holds the configuration for timeout operations.
type Config struct {
	// Duration is the maximum time to wait for an operation.
	Duration time.Duration
	// OnTimeout is called when an operation times out.
	OnTimeout func(duration time.Duration)
}

// Timeout wraps Execute with metrics tracking.
type Timeout struct {
	config Config

	totalCalls    uint64
	totalTimeouts uint64
	totalSuccess  uint64
}

// New creates a new Timeout with the given configuration.
func New(config Config) *Timeout {
	return &Timeout{config: config}
}

// Execute executes the function with a timeout and tracks metrics.
func (t *Timeout) Execute(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	atomic.AddUint64(&t.totalCalls, 1)

	result, err := Execute(ctx, t.config, fn)
	if err != nil {
		if errors.Is(err, ErrTimeout) {
			atomic.AddUint64(&t.totalTimeouts, 1)
		}
		return result, err
	}

	atomic.AddUint64(&t.totalSuccess, 1)
	return result, err
}

// GetMetrics returns total calls, total timeouts, total successes.
func (t *Timeout) GetMetrics() (totalCalls, totalTimeouts, totalSuccesses uint64) {
	return atomic.LoadUint64(&t.totalCalls),
		atomic.LoadUint64(&t.totalTimeouts),
		atomic.LoadUint64(&t.totalSuccess)
}

// Execute executes the function with a timeout (stateless version).
func Execute(ctx context.Context, config Config, fn func(ctx context.Context) (any, error)) (any, error) {
	if config.Duration <= 0 {
		return fn(ctx)
	}

	ctx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	type result struct {
		value any
		err   error
	}

	ch := make(chan result, 1)
	go func() {
		value, err := fn(ctx)
		ch <- result{value, err}
	}()

	select {
	case res := <-ch:
		return res.value, res.err
	case <-ctx.Done():
		if config.OnTimeout != nil {
			go config.OnTimeout(config.Duration)
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, ctx.Err()
	}
}
