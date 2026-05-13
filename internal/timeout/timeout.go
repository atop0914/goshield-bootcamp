// Package timeout provides timeout enforcement for operations.
package timeout

import (
    "context"
    "errors"
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

// Execute executes the function with a timeout.
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
