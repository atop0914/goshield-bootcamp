# GoShield 🛡️

**Unified Resilience Toolkit for Go**

[![Go Reference](https://pkg.go.dev/badge/github.com/atop0914/goshield.svg)](https://pkg.go.dev/github.com/atop0914/goshield)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

GoShield provides composable resilience decorators inspired by [Resilience4j](https://github.com/resilience4j/resilience4j) (Java) and [Polly](https://github.com/App-vNext/Polly) (.NET), bringing the same level of resilience tooling to Go.

## Features

- 🔌 **Circuit Breaker** - Prevent cascading failures with configurable thresholds
- 🔄 **Retry** - Automatic retries with multiple backoff strategies (fixed, exponential, fibonacci)
- ⚡ **Rate Limiter** - Token bucket and sliding window implementations
- ⏱️ **Timeout** - Enforce time limits on operations
- 🚧 **Bulkhead** - Limit concurrent executions to prevent resource exhaustion
- 🔀 **Fallback** - Provide fallback responses when operations fail
- 📊 **Prometheus Metrics** - Built-in observability for all patterns
- 🌐 **HTTP Middleware** - Ready-to-use middleware for net/http
- 🔗 **Composable** - Chain patterns together for complex resilience strategies
- ⚙️ **Configuration** - JSON config with env overrides, validation, and hot-reload

## Installation

```bash
go get github.com/atop0914/goshield
```

## Quick Start

### Circuit Breaker

```go
cb := breaker.New(breaker.Config{
    Name:                 "my-service",
    FailureRateThreshold: 50,
    MinimumNumberOfCalls: 10,
    Timeout:              30 * time.Second,
})

result, err := cb.Execute(ctx, func(ctx context.Context) (any, error) {
    return myService.Call(ctx)
})
```

### Retry with Backoff

```go
result, err := backoff.Execute(ctx, backoff.RetryConfig{
    Backoff: &backoff.ExponentialBackoff{
        InitialInterval: 100 * time.Millisecond,
        MaxInterval:     10 * time.Second,
        MaxRetries:      5,
    },
    RetryOn: func(err error) bool {
        return !errors.Is(err, context.Canceled)
    },
}, func(ctx context.Context) (any, error) {
    return myService.Call(ctx)
})
```

### Rate Limiter

```go
// Token bucket: 100 requests/sec, burst 200
limiter := ratelimit.NewTokenBucket(100, 200)

if !limiter.Allow() {
    return ErrRateLimited
}

// Or with context waiting
if err := limiter.Wait(ctx); err != nil {
    return err
}
```

### HTTP Middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/resource", handler)

// Apply resilience middleware chain
handler := middleware.Chain(
    middleware.CircuitBreaker("api", breakerCfg),
    middleware.RateLimit(limiter),
    middleware.Timeout(5*time.Second),
    middleware.Bulkhead(bulkheadCfg),
)(mux)

http.ListenAndServe(":8080", handler)
```

### Composite Executor

```go
executor := resilience.NewExecutor(
    resilience.WithCircuitBreaker(breaker.Config{...}),
    resilience.WithRetry(backoff.RetryConfig{...}),
    resilience.WithRateLimiter(ratelimit.NewTokenBucket(100, 200)),
    resilience.WithTimeout(5*time.Second),
    resilience.WithBulkhead(bulkhead.Config{MaxConcurrent: 50}),
)

result, err := executor.Execute(ctx, func(ctx context.Context) (any, error) {
    return myService.Call(ctx)
})
```

## Backoff Strategies

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `FixedBackoff` | Fixed interval between retries | Simple retry scenarios |
| `ExponentialBackoff` | Exponentially increasing intervals | Most common, prevents thundering herd |
| `ExponentialRandomBackoff` | Exponential with jitter | Distributed systems |
| `FibonacciBackoff` | Fibonacci-based intervals | Gradual backoff |

## HTTP Middleware

```go
// Individual middleware
handler := middleware.CircuitBreaker("api", cfg)(next)
handler := middleware.RateLimit(limiter)(next)
handler := middleware.Timeout(5*time.Second)(next)
handler := middleware.Bulkhead(cfg)(next)
handler := middleware.Retry(3, 100*time.Millisecond)(next)

// Or chain them together
handler := middleware.Chain(
    middleware.CircuitBreaker("api", cbCfg),
    middleware.RateLimit(limiter),
    middleware.Timeout(5*time.Second),
    middleware.Bulkhead(bhCfg),
)(mux)
```

## Prometheus Metrics

GoShield provides a **zero-dependency** metrics system with Prometheus-compatible exposition format.

### Setup

```go
import "github.com/atop0914/goshield/pkg/metrics"

// Create a registry
registry := metrics.NewRegistry()

// Register your resilience patterns
registry.RegisterBreaker(cb.Name(), 
    func() int { return int(cb.State()) },
    func() (float64, float64, uint32, uint64, uint64, uint64, uint64, uint64) {
        m := cb.GetMetrics()
        return m.FailureRate, m.SlowCallRate, m.TotalCalls,
            m.TotalSuccesses, m.TotalFailures, m.TotalRejected,
            m.TotalSlowCalls, m.StateTransitions
    },
)

registry.RegisterRateLimiter("api-limiter", limiter.Rate, limiter.Burst)

registry.RegisterBulkhead("db-pool", func() (int64, int64, int64, int64, int64) {
    return bh.GetMetricsForCollection()
})

// Serve via HTTP
http.Handle("/metrics", metrics.Handler(registry))
```

### Exposed Metrics

```
# Circuit Breaker
goshield_breaker_state{name="api",state="closed"} 1
goshield_breaker_failure_rate{name="api"} 10
goshield_breaker_slow_call_rate{name="api"} 5
goshield_breaker_calls_total{name="api"} 100
goshield_breaker_successes_total{name="api"} 90
goshield_breaker_failures_total{name="api"} 10
goshield_breaker_rejected_total{name="api"} 2
goshield_breaker_slow_calls_total{name="api"} 5
goshield_breaker_state_transitions_total{name="api"} 3

# Rate Limiter
goshield_ratelimit_rate{name="api"} 100
goshield_ratelimit_burst{name="api"} 200

# Bulkhead
goshield_bulkhead_available_permits{name="db"} 40
goshield_bulkhead_max_concurrent{name="db"} 50
goshield_bulkhead_running{name="db"} 10
goshield_bulkhead_executions_total{name="db"} 1000
goshield_bulkhead_rejections_total{name="db"} 5

# Retry
goshield_retry_attempts_total{name="api"} 100
goshield_retry_successes_total{name="api"} 80
goshield_retry_retries_total{name="api"} 20

# Timeout
goshield_timeout_calls_total{name="api"} 100
goshield_timeout_timeouts_total{name="api"} 5
goshield_timeout_successes_total{name="api"} 95
```

## Configuration

GoShield provides a **zero-dependency** configuration system with JSON file loading, environment variable overrides, validation, and hot-reload support.

### Load from JSON File

```go
import "github.com/atop0914/goshield/pkg/config"

cfg, err := config.LoadFile("goshield.json")
if err != nil {
    log.Fatal(err)
}

// Validate
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Use config values
cb := breaker.New(breaker.Config{
    Name:                 cfg.CircuitBreaker.Name,
    FailureRateThreshold: cfg.CircuitBreaker.FailureRateThreshold,
    Timeout:              cfg.CircuitBreaker.Timeout(),
    MinimumNumberOfCalls: cfg.CircuitBreaker.MinimumNumberOfCalls,
    SlidingWindowSize:    cfg.CircuitBreaker.SlidingWindowSize,
})
```

### Environment Variable Overrides

All config values can be overridden via environment variables with the prefix `GOSHIELD_`:

```bash
GOSHIELD_CIRCUIT_BREAKER_TIMEOUT_SECONDS=30
GOSHIELD_RATE_LIMITER_RATE=500
GOSHIELD_TIMEOUT_DURATION_MS=10000
GOSHIELD_BULKHEAD_MAX_CONCURRENT=50
GOSHIELD_METRICS_ENABLED=false
GOSHIELD_HTTP_ADDR=:9090
```

```go
cfg := config.DefaultConfig()
cfg.ApplyEnvOverrides()
```

### Hot Reload

Watch a config file for live changes:

```go
w, err := config.NewWatcher("goshield.json", func(cfg *config.Config) {
    // Reconfigure your components with new settings
    limiter.SetRate(cfg.RateLimiter.Rate)
    breaker.SetTimeout(cfg.CircuitBreaker.Timeout())
})
w.Start()
defer w.Stop()
```

### Presets

```go
// Conservative: lower thresholds, more sensitive to failures
cfg := config.PresetConservative()

// Aggressive: higher throughput, less sensitive
cfg := config.PresetAggressive()

// Custom
cfg := config.DefaultConfig()
cfg.RateLimiter.Rate = 1000
cfg.CircuitBreaker.FailureRateThreshold = 30
```

### JSON Config Structure

See [`examples/config/goshield.json`](examples/config/goshield.json) for a complete example.

## Comparison with Existing Solutions

| Feature | GoShield | sony/gobreaker | eapache/go-resiliency | juju/ratelimit |
|---------|----------|----------------|----------------------|----------------|
| Circuit Breaker | ✅ | ✅ | ✅ | ❌ |
| Retry | ✅ | ❌ | ✅ | ❌ |
| Rate Limiter | ✅ | ❌ | ❌ | ✅ |
| Timeout | ✅ | ❌ | ❌ | ❌ |
| Bulkhead | ✅ | ❌ | ✅ | ❌ |
| Fallback | ✅ | ❌ | ❌ | ❌ |
| HTTP Middleware | ✅ | ❌ | ❌ | ❌ |
| Prometheus | ✅ | ❌ | ❌ | ❌ |
| Composable | ✅ | ❌ | ❌ | ❌ |

## License

MIT License - see [LICENSE](LICENSE) for details.
