// Package bulkhead provides the bulkhead pattern for limiting concurrent executions.
//
// The bulkhead pattern limits the number of concurrent executions to prevent
// a single service from consuming all available resources.
//
// Example:
//
//	bh := bulkhead.New(bulkhead.Config{
//	    MaxConcurrent:   10,
//	    MaxWaitDuration: 5 * time.Second,
//	})
//
//	result, err := bh.Execute(ctx, func(ctx context.Context) (any, error) {
//	    return myService.Call(ctx)
//	})
package bulkhead

import (
	"time"

	"github.com/atop0914/goshield/internal/bulkhead"
)

// ErrBulkheadFull is returned when the bulkhead is at capacity.
var ErrBulkheadFull = bulkhead.ErrBulkheadFull

// ErrBulkheadTimeout is returned when waiting for a slot times out.
var ErrBulkheadTimeout = bulkhead.ErrBulkheadTimeout

// Config holds the configuration for a bulkhead.
type Config struct {
	// MaxConcurrent is the maximum number of concurrent executions.
	MaxConcurrent int
	// MaxWaitDuration is the maximum time to wait for a slot.
	MaxWaitDuration time.Duration
	// OnCallRejected is called when a call is rejected.
	OnCallRejected func()
}

// Bulkhead limits the number of concurrent executions.
type Bulkhead = bulkhead.Bulkhead

// Metrics returns the current metrics of the bulkhead.
type Metrics = bulkhead.Metrics

// New creates a new Bulkhead with the given configuration.
func New(config Config) *Bulkhead {
	return bulkhead.New(bulkhead.Config{
		MaxConcurrent:   config.MaxConcurrent,
		MaxWaitDuration: config.MaxWaitDuration,
		OnCallRejected:  config.OnCallRejected,
	})
}
