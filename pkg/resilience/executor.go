package resilience

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/atop0914/goshield/internal/breaker"
    "github.com/atop0914/goshield/internal/backoff"
    "github.com/atop0914/goshield/internal/ratelimit"
    "github.com/atop0914/goshield/internal/timeout"
    "github.com/atop0914/goshield/internal/bulkhead"
    "github.com/atop0914/goshield/internal/fallback"
)

// Executor combines multiple resilience patterns into a single executor.
type Executor struct {
    circuitBreaker *breaker.CircuitBreaker
    retryConfig    *backoff.RetryConfig
    rateLimiter    ratelimit.Limiter
    timeoutConfig  *timeout.Config
    bulkheadConfig *bulkhead.Bulkhead
    fallbackConfig *fallback.Config
}

// Option configures an Executor.
type Option func(*Executor)

// WithCircuitBreaker adds circuit breaker protection.
func WithCircuitBreaker(config breaker.Config) Option {
    return func(e *Executor) {
        e.circuitBreaker = breaker.New(config)
    }
}

// WithRetry adds retry with backoff.
func WithRetry(config backoff.RetryConfig) Option {
    return func(e *Executor) {
        e.retryConfig = &config
    }
}

// WithRateLimiter adds rate limiting.
func WithRateLimiter(limiter ratelimit.Limiter) Option {
    return func(e *Executor) {
        e.rateLimiter = limiter
    }
}

// WithTimeout adds timeout enforcement.
func WithTimeout(duration time.Duration) Option {
    return func(e *Executor) {
        e.timeoutConfig = &timeout.Config{Duration: duration}
    }
}

// WithBulkhead adds bulkhead protection.
func WithBulkhead(config bulkhead.Config) Option {
    return func(e *Executor) {
        e.bulkheadConfig = bulkhead.New(config)
    }
}

// WithFallback adds fallback support.
func WithFallback(config fallback.Config) Option {
    return func(e *Executor) {
        e.fallbackConfig = &config
    }
}

// NewExecutor creates a new Executor with the given options.
func NewExecutor(opts ...Option) *Executor {
    e := &Executor{}
    for _, opt := range opts {
        opt(e)
    }
    return e
}

// Patterns returns the names of the configured resilience patterns.
func (e *Executor) Patterns() []string {
    var patterns []string
    if e.fallbackConfig != nil {
        patterns = append(patterns, "fallback")
    }
    if e.circuitBreaker != nil {
        patterns = append(patterns, "circuit-breaker")
    }
    if e.retryConfig != nil {
        patterns = append(patterns, "retry")
    }
    if e.rateLimiter != nil {
        patterns = append(patterns, "rate-limiter")
    }
    if e.timeoutConfig != nil {
        patterns = append(patterns, "timeout")
    }
    if e.bulkheadConfig != nil {
        patterns = append(patterns, "bulkhead")
    }
    return patterns
}

// String returns a human-readable representation of the executor configuration.
func (e *Executor) String() string {
    patterns := e.Patterns()
    if len(patterns) == 0 {
        return "Executor(no patterns)"
    }
    return fmt.Sprintf("Executor(%s)", strings.Join(patterns, " → "))
}

// Execute executes the function with all configured resilience patterns.
func (e *Executor) Execute(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
    // Build the execution chain
    exec := fn
    
    // Apply fallback (outermost)
    if e.fallbackConfig != nil {
        inner := exec
        exec = func(ctx context.Context) (any, error) {
            return fallback.Execute(ctx, *e.fallbackConfig, inner)
        }
    }
    
    // Apply circuit breaker
    if e.circuitBreaker != nil {
        inner := exec
        exec = func(ctx context.Context) (any, error) {
            return e.circuitBreaker.Execute(ctx, inner)
        }
    }
    
    // Apply retry
    if e.retryConfig != nil {
        inner := exec
        exec = func(ctx context.Context) (any, error) {
            return backoff.Execute(ctx, *e.retryConfig, inner)
        }
    }
    
    // Apply rate limiter
    if e.rateLimiter != nil {
        inner := exec
        exec = func(ctx context.Context) (any, error) {
            if err := e.rateLimiter.Wait(ctx); err != nil {
                return nil, err
            }
            return inner(ctx)
        }
    }
    
    // Apply timeout
    if e.timeoutConfig != nil {
        inner := exec
        exec = func(ctx context.Context) (any, error) {
            return timeout.Execute(ctx, *e.timeoutConfig, inner)
        }
    }
    
    // Apply bulkhead (innermost)
    if e.bulkheadConfig != nil {
        inner := exec
        exec = func(ctx context.Context) (any, error) {
            return e.bulkheadConfig.Execute(ctx, inner)
        }
    }
    
    return exec(ctx)
}
