package main

import (
    "fmt"
    "log"
    "net/http"
    "time"
    
    "github.com/atop0914/goshield/internal/breaker"
    "github.com/atop0914/goshield/internal/bulkhead"
    "github.com/atop0914/goshield/internal/ratelimit"
    "github.com/atop0914/goshield/pkg/middleware"
)

func main() {
    // Create a simple mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, World!")
    })
    mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `{"data": "some value"}`)
    })
    
    // Configure circuit breaker
    cbConfig := breaker.Config{
        Name:                  "api-cb",
        FailureRateThreshold:  50,
        MinimumNumberOfCalls:  5,
        SlidingWindowSize:     10,
        Timeout:               30 * time.Second,
        OnStateChange: func(name string, from, to breaker.State) {
            log.Printf("Circuit breaker %s: %v -> %v", name, from, to)
        },
    }
    
    // Configure rate limiter (100 requests per second, burst 200)
    limiter := ratelimit.NewTokenBucket(100, 200)
    
    // Configure bulkhead (max 50 concurrent requests)
    bhConfig := bulkhead.Config{
        MaxConcurrent:   50,
        MaxWaitDuration: 5 * time.Second,
    }
    
    // Apply middleware chain
    handler := middleware.Chain(
        middleware.CircuitBreaker("api", cbConfig),
        middleware.RateLimit(limiter),
        middleware.Timeout(10*time.Second),
        middleware.Bulkhead(bhConfig),
    )(mux)
    
    // Start server
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", handler))
}
