// Package main demonstrates how to use GoShield's metrics system
// with Prometheus-compatible exposition.
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/atop0914/goshield/internal/breaker"
	"github.com/atop0914/goshield/internal/bulkhead"
	"github.com/atop0914/goshield/internal/metrics"
	"github.com/atop0914/goshield/internal/ratelimit"
)

func main() {
	// Create a metrics registry
	registry := metrics.NewRegistry()

	// Create and register a circuit breaker
	cb := breaker.New(breaker.Config{
		Name:                 "payment-service",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 5,
		Timeout:              10 * time.Second,
	})

	registry.Register(&metrics.BreakerCollector{
		Name:     cb.Name(),
		GetState: func() int { return int(cb.State()) },
		GetMetrics: func() (float64, float64, uint32, uint64, uint64, uint64, uint64, uint64) {
			m := cb.GetMetrics()
			return m.FailureRate, m.SlowCallRate, m.TotalCalls,
				m.TotalSuccesses, m.TotalFailures, m.TotalRejected,
				m.TotalSlowCalls, m.StateTransitions
		},
	})

	// Create and register a rate limiter
	limiter := ratelimit.NewTokenBucket(100, 200)
	registry.Register(&metrics.RateLimiterCollector{
		Name:     "api-limiter",
		GetRate:  limiter.Rate,
		GetBurst: limiter.Burst,
	})

	// Create and register a bulkhead
	bh := bulkhead.New(bulkhead.Config{MaxConcurrent: 50})
	registry.Register(&metrics.BulkheadCollector{
		Name: "db-pool",
		GetMetrics: func() (int64, int64, int64, int64, int64) {
			return bh.GetMetricsForCollection()
		},
	})

	// Simulate some traffic
	go func() {
		for i := 0; i < 100; i++ {
			cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
				if i%10 == 0 {
					return nil, fmt.Errorf("simulated error")
				}
				return "ok", nil
			})
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Serve metrics via HTTP
	http.Handle("/metrics", metrics.HTTPHandler(registry))

	// Also serve a simple health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("Metrics server listening on :9090")
	fmt.Println("  GET /metrics  — Prometheus metrics")
	fmt.Println("  GET /health   — Health check")
	fmt.Println()
	fmt.Println("Example curl:")
	fmt.Println("  curl http://localhost:9090/metrics")

	if err := http.ListenAndServe(":9090", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
