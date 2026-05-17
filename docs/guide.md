# GoShield Advanced Usage Guide

This guide covers advanced patterns, real-world integration scenarios, and best practices for using GoShield in production.

## Table of Contents

- [Composition Patterns](#composition-patterns)
- [Custom Strategies](#custom-strategies)
- [Production Integration](#production-integration)
- [Configuration Management](#configuration-management)
- [Observability](#observability)
- [Error Handling](#error-handling)
- [Performance Tuning](#performance-tuning)

---

## Composition Patterns

### Layered Resilience with Executor

The `Executor` applies patterns in a specific order. Understanding this order is critical:

```
Request → Fallback → CircuitBreaker → Retry → RateLimiter → Timeout → Bulkhead → Your Function
```

- **Fallback** is outermost: catches any failure from inner patterns
- **CircuitBreaker** wraps Retry: a tripped breaker stops retries immediately
- **Retry** wraps RateLimiter: transient rate-limit errors can be retried
- **Bulkhead** is innermost: limits actual concurrent work

```go
executor := resilience.NewExecutor(
    resilience.WithFallback(fallback.Config{
        Fallback: func(ctx context.Context, err error) (any, error) {
            return getCachedResponse(), nil
        },
    }),
    resilience.WithCircuitBreaker(breaker.Config{
        Name:                 "payment-api",
        FailureRateThreshold: 50,
        MinimumNumberOfCalls: 10,
        Timeout:              30 * time.Second,
    }),
    resilience.WithRetry(backoff.RetryConfig{
        Backoff: &backoff.ExponentialBackoff{
            InitialInterval: 100 * time.Millisecond,
            MaxInterval:     5 * time.Second,
            MaxRetryCount:   3,
        },
    }),
    resilience.WithRateLimiter(ratelimit.NewTokenBucket(1000, 2000)),
    resilience.WithTimeout(10 * time.Second),
    resilience.WithBulkhead(bulkhead.Config{MaxConcurrent: 100}),
)

result, err := executor.Execute(ctx, func(ctx context.Context) (any, error) {
    return paymentService.Charge(ctx, amount)
})
```

### Per-Service Breakers

Use different circuit breaker configurations per downstream service:

```go
// Critical service: tight thresholds, fast failure
criticalBreaker := breaker.New(breaker.Config{
    Name:                 "critical-api",
    FailureRateThreshold: 30,
    MinimumNumberOfCalls: 5,
    SlidingWindowSize:    10,
    Timeout:              10 * time.Second,
})

// Non-critical service: relaxed thresholds
relaxedBreaker := breaker.New(breaker.Config{
    Name:                 "analytics-api",
    FailureRateThreshold: 70,
    MinimumNumberOfCalls: 20,
    SlidingWindowSize:    50,
    Timeout:              60 * time.Second,
})
```

### Adaptive Breaker for Variable Workloads

When your service experiences varying load patterns, the adaptive breaker automatically adjusts thresholds:

```go
ab := breaker.NewAdaptive(breaker.AdaptiveConfig{
    Base: breaker.Config{
        Name:                 "flaky-service",
        FailureRateThreshold: 50,
        MinimumNumberOfCalls: 10,
        Timeout:              30 * time.Second,
    },
    FailureRateEMAAlpha:     0.5,   // How quickly to react to failure changes
    LatencyEMAAlpha:         0.3,   // How quickly to react to latency changes
    MinFailureRateThreshold: 20,    // Never go below 20% threshold
    MaxFailureRateThreshold: 80,    // Never go above 80% threshold
    ConsecutiveFailureLimit: 5,     // Trip immediately after 5 consecutive failures
    SlowCallMultiplier:      2.0,   // Slow = 2× average latency
    TimeoutMultiplier:       1.5,   // Each trip increases timeout by 1.5×
    MaxTimeout:              5 * time.Minute,
    OnAdaptiveChange: func(name string, params breaker.AdaptiveParams) {
        log.Printf("[Adaptive] %s threshold=%.1f%% latencyEMA=%v",
            name, params.AdaptiveThreshold, params.LatencyEMA)
    },
})

// Periodically update EMAs (e.g., in a goroutine)
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        ab.UpdateEMAs()
    }
}()
```

---

## Custom Strategies

### Custom Retry Predicate

Control exactly which errors trigger retries:

```go
retryConfig := backoff.RetryConfig{
    Backoff: &backoff.ExponentialBackoff{
        InitialInterval: 200 * time.Millisecond,
        MaxInterval:     30 * time.Second,
        MaxRetryCount:   5,
        Multiplier:      2.0,
    },
    RetryOn: func(err error) bool {
        // Retry on transient errors
        var netErr net.Error
        if errors.As(err, &netErr) && netErr.Timeout() {
            return true
        }
        // Retry on specific HTTP status codes
        var httpErr *HTTPError
        if errors.As(err, &httpErr) {
            return httpErr.StatusCode == 503 || httpErr.StatusCode == 429
        }
        // Don't retry on context cancellation or permanent errors
        return !errors.Is(err, context.Canceled)
    },
    OnRetry: func(attempt int, err error) {
        log.Printf("Retry attempt %d: %v", attempt+1, err)
    },
}
```

### Custom Breaker Success Criteria

By default, any non-nil error counts as a failure. Customize this:

```go
cb := breaker.New(breaker.Config{
    Name:                 "grpc-service",
    FailureRateThreshold: 50,
    MinimumNumberOfCalls: 10,
    Timeout:              30 * time.Second,
    IsSuccessful: func(err error) bool {
        // Don't count "not found" as a failure
        if errors.Is(err, ErrNotFound) {
            return true
        }
        // Don't count client errors (4xx) as failures
        var httpErr *HTTPError
        if errors.As(err, &httpErr) && httpErr.StatusCode < 500 {
            return true
        }
        return err == nil
    },
})
```

### Fibonacci Backoff for Gradual Recovery

When you want a gentler backoff curve than exponential:

```go
retryConfig := backoff.RetryConfig{
    Backoff: &backoff.FibonacciBackoff{
        InitialInterval: 100 * time.Millisecond,
        MaxInterval:     10 * time.Second,
        MaxRetryCount:   8,
    },
}
// Intervals: 100ms, 100ms, 200ms, 300ms, 500ms, 800ms, 1.3s, 2.1s ...
```

### Exponential Random Backoff (Jitter)

Prevents thundering herd in distributed systems:

```go
retryConfig := backoff.RetryConfig{
    Backoff: &backoff.ExponentialRandomBackoff{
        InitialInterval: 100 * time.Millisecond,
        MaxInterval:     10 * time.Second,
        MaxRetryCount:   5,
        Multiplier:      2.0,
    },
}
// Each retry: random value between 0 and computed exponential interval
```

---

## Production Integration

### HTTP Client with Full Resilience

```go
func NewResilientHTTPClient(baseURL string) *ResilientClient {
    cb := breaker.New(breaker.Config{
        Name:                 "http-client",
        FailureRateThreshold: 50,
        MinimumNumberOfCalls: 10,
        Timeout:              30 * time.Second,
        SlowCallDuration:     2 * time.Second,
        SlowCallRateThreshold: 30,
    })

    limiter := ratelimit.NewTokenBucket(100, 200)

    executor := resilience.NewExecutor(
        resilience.WithCircuitBreaker(breaker.Config{...}),
        resilience.WithRetry(backoff.RetryConfig{
            Backoff: &backoff.ExponentialBackoff{
                InitialInterval: 100 * time.Millisecond,
                MaxInterval:     5 * time.Second,
                MaxRetryCount:   3,
            },
            RetryOn: func(err error) bool {
                // Only retry on 5xx and network errors
                return !errors.Is(err, context.Canceled)
            },
        }),
        resilience.WithRateLimiter(limiter),
        resilience.WithTimeout(10 * time.Second),
    )

    return &ResilientClient{executor: executor, client: &http.Client{}}
}

func (rc *ResilientClient) Get(ctx context.Context, url string) ([]byte, error) {
    result, err := rc.executor.Execute(ctx, func(ctx context.Context) (any, error) {
        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        resp, err := rc.client.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()
        return io.ReadAll(resp.Body)
    })
    if err != nil {
        return nil, err
    }
    return result.([]byte), nil
}
```

### Database Connection Pool Protection

```go
// Protect a database connection pool with bulkhead + circuit breaker
dbBulkhead := bulkhead.New(bulkhead.Config{
    MaxConcurrent:   50,              // Max 50 concurrent queries
    MaxWaitDuration: 5 * time.Second, // Fail fast if pool is full
    OnCallRejected: func() {
        log.Warn("database bulkhead full, request rejected")
        metrics.DBRejections.Inc()
    },
})

dbBreaker := breaker.New(breaker.Config{
    Name:                 "postgres",
    FailureRateThreshold: 60,
    MinimumNumberOfCalls: 20,
    Timeout:              15 * time.Second,
    IsSuccessful: func(err error) bool {
        // Don't count "no rows" as a failure
        if errors.Is(err, sql.ErrNoRows) {
            return true
        }
        return err == nil
    },
})

func QueryRow(ctx context.Context, query string, args ...any) (*sql.Row, error) {
    result, err := dbBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
        return dbBulkhead.Execute(ctx, func(ctx context.Context) (any, error) {
            return db.QueryRowContext(ctx, query, args...), nil
        })
    })
    if err != nil {
        return nil, err
    }
    return result.(*sql.Row), nil
}
```

### gRPC Interceptor

```go
func UnaryClientInterceptor(executor *resilience.Executor) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{},
        cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

        _, err := executor.Execute(ctx, func(ctx context.Context) (any, error) {
            return nil, invoker(ctx, method, req, reply, cc, opts...)
        })
        return err
    }
}

// Usage
conn, _ := grpc.Dial("localhost:50051",
    grpc.WithUnaryInterceptor(UnaryClientInterceptor(executor)),
)
```

---

## Configuration Management

### Multi-Environment Config

```go
func LoadConfig() (*config.Config, error) {
    // Start with defaults
    cfg := config.DefaultConfig()

    // Load from file (if exists)
    if _, err := os.Stat("goshield.json"); err == nil {
        cfg, err = config.LoadFile("goshield.json")
        if err != nil {
            return nil, fmt.Errorf("load config: %w", err)
        }
    }

    // Environment overrides (takes precedence)
    cfg.ApplyEnvOverrides()

    // Validate
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("validate config: %w", err)
    }

    return cfg, nil
}
```

### Hot-Reload with Component Reconfiguration

```go
func StartWithHotReload(cfgPath string) error {
    cfg, err := config.LoadFile(cfgPath)
    if err != nil {
        return err
    }

    // Create components from initial config
    cb := breaker.New(breaker.Config{
        Name:                 cfg.CircuitBreaker.Name,
        FailureRateThreshold: cfg.CircuitBreaker.FailureRateThreshold,
        Timeout:              cfg.CircuitBreaker.Timeout(),
    })
    limiter := ratelimit.NewTokenBucket(cfg.RateLimiter.Rate, cfg.RateLimiter.Burst)

    // Watch for changes
    watcher, err := config.NewWatcher(cfgPath, func(newCfg *config.Config) {
        log.Println("Config changed, reconfiguring...")
        // Note: Some changes require recreating components
        // Rate limiter can be updated in-place
        limiter.SetRate(newCfg.RateLimiter.Rate)
        // Circuit breaker threshold can be updated via ForceOpen/Reset if needed
    })
    if err != nil {
        return err
    }
    watcher.Start()
    defer watcher.Stop()

    // ... start your application
    return nil
}
```

---

## Observability

### Full Metrics Stack

```go
func SetupMetrics() (*metrics.Registry, http.Handler) {
    registry := metrics.NewRegistry()

    // Register all components
    registry.Register(&metrics.BreakerCollector{
        Name:     "payment-api",
        GetState: func() int { return int(paymentBreaker.State()) },
        GetMetrics: func() (float64, float64, uint32, uint64, uint64, uint64, uint64, uint64) {
            m := paymentBreaker.GetMetrics()
            return m.FailureRate, m.SlowCallRate, m.TotalCalls,
                m.TotalSuccesses, m.TotalFailures, m.TotalRejected,
                m.TotalSlowCalls, m.StateTransitions
        },
    })

    registry.Register(&metrics.RateLimiterCollector{
        Name:     "global-limiter",
        GetRate:  limiter.Rate,
        GetBurst: limiter.Burst,
    })

    registry.Register(&metrics.BulkheadCollector{
        Name: "db-pool",
        GetMetrics: func() (int64, int64, int64, int64, int64) {
            return dbBulkhead.GetMetricsForCollection()
        },
    })

    return registry, metrics.Handler(registry)
}

// In your main():
registry, metricsHandler := SetupMetrics()
http.Handle("/metrics", metricsHandler)
go http.ListenAndServe(":9090", nil)
```

### State Change Logging

```go
cb := breaker.New(breaker.Config{
    Name: "payment-api",
    OnStateChange: func(name string, from, to breaker.State) {
        log.Printf("[CircuitBreaker] %s: %s -> %s", name, from, to)

        // Alert on state changes
        switch to {
        case breaker.StateOpen:
            alert.Fire("circuit_breaker_open", map[string]string{
                "breaker": name,
                "from":    from.String(),
            })
        case breaker.StateClosed:
            alert.Resolve("circuit_breaker_open", name)
        }
    },
})
```

---

## Error Handling

### Error Type Checking

GoShield returns specific errors that you can check:

```go
result, err := executor.Execute(ctx, fn)
if err != nil {
    switch {
    case errors.Is(err, breaker.ErrOpenState):
        // Circuit breaker is open, service is down
        http.Error(w, "Service temporarily unavailable", 503)

    case errors.Is(err, breaker.ErrTooManyRequests):
        // Half-open state, too many concurrent probes
        http.Error(w, "Service temporarily unavailable", 503)

    case errors.Is(err, ratelimit.ErrRateLimitExceeded):
        // Rate limit exceeded
        http.Error(w, "Rate limit exceeded", 429)

    case errors.Is(err, bulkhead.ErrBulkheadFull):
        // Bulkhead at capacity
        http.Error(w, "Server busy", 503)

    case errors.Is(err, timeout.ErrTimeout):
        // Operation timed out
        http.Error(w, "Gateway timeout", 504)

    default:
        // Actual application error
        http.Error(w, "Internal error", 500)
    }
}
```

### Graceful Degradation Chain

```go
func GetData(ctx context.Context) (*Data, error) {
    // Try primary source
    data, err := primaryExecutor.Execute(ctx, func(ctx context.Context) (any, error) {
        return primaryDB.Query(ctx)
    })
    if err == nil {
        return data.(*Data), nil
    }

    // Fallback to replica
    data, err = replicaExecutor.Execute(ctx, func(ctx context.Context) (any, error) {
        return replicaDB.Query(ctx)
    })
    if err == nil {
        return data.(*Data), nil
    }

    // Final fallback to cache
    return getCachedData()
}
```

---

## Performance Tuning

### Zero-Allocation Hot Path

GoShield is designed for zero allocations in the hot path. Key tips:

1. **Reuse context values**: Avoid creating new context values per request
2. **Pre-create executors**: Don't create `resilience.NewExecutor` per request
3. **Use count-based windows**: Time-based windows have slightly higher overhead

```go
// ✅ Good: Create once, use many times
var executor = resilience.NewExecutor(
    resilience.WithCircuitBreaker(breaker.Config{...}),
    resilience.WithTimeout(5 * time.Second),
)

func handler(w http.ResponseWriter, r *http.Request) {
    result, err := executor.Execute(r.Context(), func(ctx context.Context) (any, error) {
        return fetchData(ctx)
    })
    // ...
}

// ❌ Bad: Creating per request
func handler(w http.ResponseWriter, r *http.Request) {
    executor := resilience.NewExecutor(...) // Allocates every time
    result, err := executor.Execute(...)
}
```

### Benchmarking Your Setup

```go
func BenchmarkExecutor(b *testing.B) {
    executor := resilience.NewExecutor(
        resilience.WithCircuitBreaker(breaker.Config{
            Name:                 "bench",
            FailureRateThreshold: 50,
            MinimumNumberOfCalls: 1000000, // Never trip during bench
        }),
        resilience.WithTimeout(10 * time.Second),
    )

    ctx := context.Background()
    fn := func(ctx context.Context) (any, error) { return "ok", nil }

    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        executor.Execute(ctx, fn)
    }
}
```

Expected benchmarks on modern hardware:

| Pattern | Ops/sec | Latency | Allocations |
|---------|---------|---------|-------------|
| Circuit Breaker | ~4.6M | 280 ns/op | 0 B/op |
| Adaptive Breaker | ~2.5M | 450 ns/op | 0 B/op |
| Rate Limiter | ~11.5M | 103 ns/op | 0 B/op |
| Bulkhead | ~8.9M | 146 ns/op | 0 B/op |
| Timeout | ~515K | 2.7 µs/op | 576 B/op |
| Composite | ~325K | 3.6 µs/op | 696 B/op |

---

## Testing

### Short Mode for CI

All container-dependent tests respect the `-short` flag:

```bash
# Run all tests (skip container tests)
go test -short ./...

# Run specific package tests
go test -v ./internal/breaker/...
```

### Mocking Resilience Patterns

For unit tests, you may want to disable resilience patterns:

```go
// Create a "no-op" breaker that never trips
testBreaker := breaker.New(breaker.Config{
    Name:                 "test",
    FailureRateThreshold: 100, // Never trip
    MinimumNumberOfCalls: math.MaxUint32, // Never calculate rate
    Timeout:              time.Nanosecond, // Instant recovery
})
```
