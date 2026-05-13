package benchmarks

import (
    "context"
    "testing"
    "time"
    
    "github.com/atop0914/goshield/internal/backoff"
    "github.com/atop0914/goshield/internal/breaker"
    "github.com/atop0914/goshield/internal/bulkhead"
    "github.com/atop0914/goshield/internal/ratelimit"
    "github.com/atop0914/goshield/internal/timeout"
    "github.com/atop0914/goshield/pkg/resilience"
)

func BenchmarkCircuitBreaker_Closed(b *testing.B) {
    cb := breaker.New(breaker.Config{
        Name:                 "bench",
        FailureRateThreshold: 50,
        MinimumNumberOfCalls: 1000000,
    })
    
    ctx := context.Background()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        cb.Execute(ctx, func(ctx context.Context) (any, error) {
            return "ok", nil
        })
    }
}

func BenchmarkRateLimiter_TokenBucket(b *testing.B) {
    tb := ratelimit.NewTokenBucket(1000000, 1000000)
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        tb.Allow()
    }
}

func BenchmarkBulkhead(b *testing.B) {
    bh := bulkhead.New(bulkhead.Config{
        MaxConcurrent: 1000000,
    })
    
    ctx := context.Background()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        bh.Execute(ctx, func(ctx context.Context) (any, error) {
            return "ok", nil
        })
    }
}

func BenchmarkTimeout(b *testing.B) {
    cfg := timeout.Config{
        Duration: 10 * time.Second,
    }
    
    ctx := context.Background()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        timeout.Execute(ctx, cfg, func(ctx context.Context) (any, error) {
            return "ok", nil
        })
    }
}

func BenchmarkRetry_Success(b *testing.B) {
    cfg := backoff.RetryConfig{
        Backoff: &backoff.FixedBackoff{
            Interval:   1 * time.Millisecond,
            MaxRetryCount: 3,
        },
    }
    
    ctx := context.Background()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        backoff.Execute(ctx, cfg, func(ctx context.Context) (any, error) {
            return "ok", nil
        })
    }
}

func BenchmarkCompositeExecutor(b *testing.B) {
    executor := resilience.NewExecutor(
        resilience.WithCircuitBreaker(breaker.Config{
            Name:                 "bench",
            FailureRateThreshold: 50,
            MinimumNumberOfCalls: 1000000,
        }),
        resilience.WithRetry(backoff.RetryConfig{
            Backoff: &backoff.FixedBackoff{
                Interval:   1 * time.Millisecond,
                MaxRetryCount: 3,
            },
        }),
        resilience.WithRateLimiter(ratelimit.NewTokenBucket(1000000, 1000000)),
        resilience.WithTimeout(10*time.Second),
        resilience.WithBulkhead(bulkhead.Config{MaxConcurrent: 1000000}),
    )
    
    ctx := context.Background()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        executor.Execute(ctx, func(ctx context.Context) (any, error) {
            return "ok", nil
        })
    }
}
