// Package fallback provides fallback functionality for failed operations.
//
// Example:
//
//	result, err := fallback.Execute(ctx, fallback.Config{
//	    Fallback: func(ctx context.Context, err error) (any, error) {
//	        return getCachedValue(), nil
//	    },
//	}, func(ctx context.Context) (any, error) {
//	    return myService.Call(ctx)
//	})
package fallback

import (
	"context"

	"github.com/atop0914/goshield/internal/fallback"
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
	return fallback.Execute(ctx, fallback.Config{
		Fallback:   config.Fallback,
		OnFallback: config.OnFallback,
	}, fn)
}
