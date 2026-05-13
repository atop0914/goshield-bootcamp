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

GoShield automatically exposes Prometheus metrics:

```
goshield_circuit_breaker_state{name="api"} 0
goshield_circuit_breaker_calls_total{name="api",result="success"} 100
goshield_circuit_breaker_calls_total{name="api",result="failure"} 5
goshield_circuit_breaker_failures_total{name="api"} 5
goshield_rate_limiter_requests_total{name="api",result="allowed"} 1000
goshield_bulkhead_active{name="api"} 10
```

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
