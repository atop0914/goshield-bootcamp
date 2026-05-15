// Package breaker provides the circuit breaker pattern for Go applications.
//
// The circuit breaker prevents cascading failures by monitoring the failure rate
// of operations and temporarily blocking requests when the failure rate exceeds
// a configured threshold.
//
// Example:
//
//    cb := breaker.New(breaker.Config{
//        Name:                 "my-service",
//        FailureRateThreshold: 50,
//        MinimumNumberOfCalls: 10,
//        Timeout:              30 * time.Second,
//    })
//
//    result, err := cb.Execute(ctx, func(ctx context.Context) (any, error) {
//        return myService.Call(ctx)
//    })
package breaker

import (
    "context"
    "time"
    
    "github.com/atop0914/goshield/internal/breaker"
)

// State represents the circuit breaker state.
type State = breaker.State

const (
    StateClosed     = breaker.StateClosed
    StateOpen       = breaker.StateOpen
    StateHalfOpen   = breaker.StateHalfOpen
    StateDisabled   = breaker.StateDisabled
    StateForcedOpen = breaker.StateForcedOpen
)

// Config holds the configuration for a circuit breaker.
type Config struct {
    // Name is the identifier for this circuit breaker.
    Name string
    // MaxRequests is the maximum number of requests allowed in half-open state.
    MaxRequests uint32
    // Interval is the cyclic period of the closed state.
    Interval time.Duration
    // Timeout is the duration of the open state before transitioning to half-open.
    Timeout time.Duration
    // FailureRateThreshold is the failure rate threshold (0-100) to trip the breaker.
    FailureRateThreshold uint8
    // SlowCallRateThreshold is the slow call rate threshold (0-100).
    SlowCallRateThreshold uint8
    // SlowCallDuration defines the duration above which a call is considered slow.
    SlowCallDuration time.Duration
    // MinimumNumberOfCalls is the minimum number of calls before calculating failure rate.
    MinimumNumberOfCalls uint32
    // SlidingWindowSize is the size of the sliding window.
    SlidingWindowSize uint32
    // SlidingWindowType determines if the window is count-based or time-based.
    SlidingWindowType SlidingWindowType
    // OnStateChange is called when the state changes.
    OnStateChange func(name string, from, to State)
    // IsSuccessful determines if an error should be counted as a failure.
    IsSuccessful func(err error) bool
}

// SlidingWindowType determines the type of sliding window.
type SlidingWindowType = breaker.SlidingWindowType

const (
    SlidingWindowCount = breaker.SlidingWindowCount
    SlidingWindowTime  = breaker.SlidingWindowTime
)

// CircuitBreaker is the public interface for the circuit breaker.
type CircuitBreaker struct {
    internal *breaker.CircuitBreaker
}

// New creates a new CircuitBreaker with the given configuration.
func New(config Config) *CircuitBreaker {
    internalConfig := breaker.Config{
        Name:                  config.Name,
        MaxRequests:           config.MaxRequests,
        Interval:              config.Interval,
        Timeout:               config.Timeout,
        FailureRateThreshold:  config.FailureRateThreshold,
        SlowCallRateThreshold: config.SlowCallRateThreshold,
        SlowCallDuration:      config.SlowCallDuration,
        MinimumNumberOfCalls:  config.MinimumNumberOfCalls,
        SlidingWindowSize:     config.SlidingWindowSize,
        SlidingWindowType:     config.SlidingWindowType,
        OnStateChange:         config.OnStateChange,
        IsSuccessful:          config.IsSuccessful,
    }
    
    return &CircuitBreaker{
        internal: breaker.New(internalConfig),
    }
}

// Name returns the name of the circuit breaker.
func (cb *CircuitBreaker) Name() string {
    return cb.internal.Name()
}

// State returns the current state.
func (cb *CircuitBreaker) State() State {
    return cb.internal.State()
}

// Execute wraps a function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
    return cb.internal.Execute(ctx, fn)
}

// Reset resets the circuit breaker to its initial state.
func (cb *CircuitBreaker) Reset() {
    cb.internal.Reset()
}

// ForceOpen forces the circuit breaker to the open state.
func (cb *CircuitBreaker) ForceOpen() {
    cb.internal.ForceOpen()
}

// Metrics returns the current metrics.
type Metrics = breaker.Metrics

// GetMetrics returns the current metrics.
func (cb *CircuitBreaker) GetMetrics() Metrics {
    return cb.internal.GetMetrics()
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(name string) Config {
    internal := breaker.DefaultConfig(name)
    return Config{
        Name:                  internal.Name,
        MaxRequests:           internal.MaxRequests,
        Interval:              internal.Interval,
        Timeout:               internal.Timeout,
        FailureRateThreshold:  internal.FailureRateThreshold,
        SlowCallRateThreshold: internal.SlowCallRateThreshold,
        SlowCallDuration:      internal.SlowCallDuration,
        MinimumNumberOfCalls:  internal.MinimumNumberOfCalls,
        SlidingWindowSize:     internal.SlidingWindowSize,
        SlidingWindowType:     internal.SlidingWindowType,
    }
}

// AdaptiveConfig holds configuration for the adaptive circuit breaker.
type AdaptiveConfig = breaker.AdaptiveConfig

// AdaptiveParams holds the current adaptive parameters.
type AdaptiveParams = breaker.AdaptiveParams

// AdaptiveBreaker extends the circuit breaker with adaptive threshold adjustment.
type AdaptiveBreaker struct {
    internal *breaker.AdaptiveBreaker
}

// NewAdaptive creates a new adaptive circuit breaker.
func NewAdaptive(config AdaptiveConfig) *AdaptiveBreaker {
    return &AdaptiveBreaker{
        internal: breaker.NewAdaptive(config),
    }
}

// DefaultAdaptiveConfig returns sensible defaults for the adaptive circuit breaker.
func DefaultAdaptiveConfig(name string) AdaptiveConfig {
    return breaker.DefaultAdaptiveConfig(name)
}

// Name returns the name of the adaptive circuit breaker.
func (ab *AdaptiveBreaker) Name() string {
    return ab.internal.Name()
}

// State returns the current state.
func (ab *AdaptiveBreaker) State() State {
    return ab.internal.State()
}

// Execute wraps a function with adaptive circuit breaker protection.
func (ab *AdaptiveBreaker) Execute(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
    return ab.internal.Execute(ctx, fn)
}

// UpdateEMAs recalculates exponential moving averages.
func (ab *AdaptiveBreaker) UpdateEMAs() {
    ab.internal.UpdateEMAs()
}

// GetAdaptiveParams returns the current adaptive parameters.
func (ab *AdaptiveBreaker) GetAdaptiveParams() AdaptiveParams {
    return ab.internal.GetAdaptiveParams()
}

// TripCount returns the number of times the breaker has tripped.
func (ab *AdaptiveBreaker) TripCount() uint32 {
    return ab.internal.TripCount()
}

// Reset resets the adaptive breaker to its initial state.
func (ab *AdaptiveBreaker) Reset() {
    ab.internal.Reset()
}

// ForceOpen forces the breaker into the open state.
func (ab *AdaptiveBreaker) ForceOpen() {
    ab.internal.ForceOpen()
}

// GetMetrics returns the underlying breaker's metrics.
func (ab *AdaptiveBreaker) GetMetrics() Metrics {
    return ab.internal.GetMetrics()
}
