// Package fallback provides fallback functionality for failed operations.
package fallback

import (
    "context"
)

// Config holds the configuration for fallback operations.
type Config struct {
    // Fallback is the function to call when the primary operation fails.
    Fallback func(ctx context.Context, err error) (any, error)
    // OnFallback is called before the fallback is executed.
    OnFallback func(err error)
}

// Execute executes the function with fallback support.
func Execute(ctx context.Context, config Config, fn func(ctx context.Context) (any, error)) (any, error) {
    result, err := fn(ctx)
    if err == nil {
        return result, nil
    }
    
    if config.Fallback == nil {
        return nil, err
    }
    
    if config.OnFallback != nil {
        go config.OnFallback(err)
    }
    
    return config.Fallback(ctx, err)
}
