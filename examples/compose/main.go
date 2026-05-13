package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"
    
    "github.com/atop0914/goshield/internal/backoff"
    "github.com/atop0914/goshield/internal/breaker"
    "github.com/atop0914/goshield/internal/bulkhead"
    "github.com/atop0914/goshield/internal/ratelimit"
    "github.com/atop0914/goshield/pkg/resilience"
)

func main() {
    // Create a composite executor with multiple resilience patterns
    executor := resilience.NewExecutor(
        // Circuit breaker: trip after 50% failure rate
        resilience.WithCircuitBreaker(breaker.Config{
            Name:                  "api-breaker",
            FailureRateThreshold:  50,
            MinimumNumberOfCalls:  5,
            SlidingWindowSize:     10,
            Timeout:               30 * time.Second,
            OnStateChange: func(name string, from, to breaker.State) {
                log.Printf("[CircuitBreaker] %s: %v -> %v", name, from, to)
            },
        }),
        
        // Retry: exponential backoff, max 3 retries
        resilience.WithRetry(backoff.RetryConfig{
            Backoff: &backoff.ExponentialBackoff{
                InitialInterval: 100 * time.Millisecond,
                MaxInterval:     5 * time.Second,
                MaxRetryCount:   3,
                Multiplier:      2.0,
            },
            RetryOn: func(err error) bool {
                // Only retry on transient errors
                return !errors.Is(err, context.Canceled)
            },
            OnRetry: func(retry int, err error) {
                log.Printf("[Retry] attempt %d: %v", retry+1, err)
            },
        }),
        
        // Rate limiter: 100 requests per second
        resilience.WithRateLimiter(ratelimit.NewTokenBucket(100, 200)),
        
        // Timeout: 5 seconds per request
        resilience.WithTimeout(5*time.Second),
        
        // Bulkhead: max 50 concurrent requests
        resilience.WithBulkhead(bulkhead.Config{
            MaxConcurrent:   50,
            MaxWaitDuration: 10 * time.Second,
        }),
    )
    
    // Simulate API calls
    for i := 0; i < 10; i++ {
        result, err := executor.Execute(context.Background(), func(ctx context.Context) (any, error) {
            // Simulate an API call
            return callExternalAPI(ctx, i)
        })
        
        if err != nil {
            log.Printf("Call %d failed: %v", i, err)
            continue
        }
        
        fmt.Printf("Call %d succeeded: %v\n", i, result)
    }
}

func callExternalAPI(ctx context.Context, id int) (any, error) {
    // Simulate some failures
    if id%3 == 0 {
        return nil, errors.New("service unavailable")
    }
    return fmt.Sprintf("response-%d", id), nil
}
