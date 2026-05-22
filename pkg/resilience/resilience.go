// Package resilience provides unified resilience patterns for Go applications.
//
// GoShield provides composable resilience decorators inspired by Resilience4j (Java)
// and Polly (.NET), bringing the same level of resilience tooling to Go.
//
// Supported patterns:
//   - Circuit Breaker: Prevent cascading failures
//   - Retry: Automatic retries with configurable backoff
//   - Rate Limiter: Control request throughput
//   - Timeout: Enforce time limits on operations
//   - Bulkhead: Limit concurrent executions
//   - Fallback: Provide fallback responses on failure
package resilience

import (
	"context"
	"time"
)

// Result represents the outcome of a resilience-wrapped operation.
type Result struct {
	// Success indicates whether the operation completed successfully.
	Success bool
	// Error is the error returned by the operation, if any.
	Error error
	// Duration is how long the operation took.
	Duration time.Duration
	// Metadata contains additional information about the execution.
	Metadata map[string]any
}

// ExecuteFunc is the function signature for operations wrapped by resilience decorators.
type ExecuteFunc func(ctx context.Context) (any, error)

// FallbackFunc provides a fallback value when the primary operation fails.
type FallbackFunc func(ctx context.Context, err error) (any, error)

// Decorator wraps an ExecuteFunc with resilience behavior.
type Decorator interface {
	// Wrap returns a new ExecuteFunc that applies the resilience pattern.
	Wrap(next ExecuteFunc) ExecuteFunc
	// Name returns the name of this decorator for metrics/logging.
	Name() string
}

// Chain composes multiple decorators into a single decorator.
// Decorators are applied in order: the first decorator is the outermost wrapper.
func Chain(decorators ...Decorator) Decorator {
	return &chainDecorator{decorators: decorators}
}

type chainDecorator struct {
	decorators []Decorator
}

func (c *chainDecorator) Name() string {
	return "chain"
}

func (c *chainDecorator) Wrap(next ExecuteFunc) ExecuteFunc {
	// Apply decorators in reverse order so the first one is outermost
	wrapped := next
	for i := len(c.decorators) - 1; i >= 0; i-- {
		wrapped = c.decorators[i].Wrap(wrapped)
	}
	return wrapped
}

// State represents the state of a resilience component.
type State int

const (
	// StateClosed is the normal operating state.
	StateClosed State = iota
	// StateOpen means the component is blocking operations.
	StateOpen
	// StateHalfOpen means the component is testing if it can return to normal.
	StateHalfOpen
	// StateDisabled means the component is not active.
	StateDisabled
	// StateForcedOpen means the component is manually forced open.
	StateForcedOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	case StateDisabled:
		return "DISABLED"
	case StateForcedOpen:
		return "FORCED_OPEN"
	default:
		return "UNKNOWN"
	}
}
