// Package main demonstrates advanced GoShield composition patterns.
//
// This example shows:
//   - Multi-service circuit breaker isolation
//   - Adaptive breaker with EMA-based threshold adjustment
//   - Graceful degradation with fallback chains
//   - Full observability with metrics
//
// Usage:
//
//	go run ./examples/advanced
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/atop0914/goshield/internal/backoff"
	"github.com/atop0914/goshield/internal/breaker"
	"github.com/atop0914/goshield/internal/bulkhead"
	"github.com/atop0914/goshield/internal/metrics"
	"github.com/atop0914/goshield/internal/ratelimit"
	"github.com/atop0914/goshield/pkg/resilience"
)

// Simulated service errors
var (
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTimeout            = errors.New("request timeout")
)

// SimulateServiceCall simulates a downstream service call with variable reliability.
func SimulateServiceCall(failureRate float64) func(ctx context.Context) (any, error) {
	return func(ctx context.Context) (any, error) {
		// Simulate latency
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

		if rand.Float64() < failureRate {
			return nil, ErrServiceUnavailable
		}
		return map[string]string{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)}, nil
	}
}

func main() {
	ctx := context.Background()

	// === 1. Per-Service Circuit Breakers ===
	fmt.Println("=== Per-Service Circuit Breakers ===")

	// Critical payment service: tight thresholds
	paymentBreaker := breaker.New(breaker.Config{
		Name:                  "payment-service",
		FailureRateThreshold:  30, // Trip at 30% failure
		MinimumNumberOfCalls:  5,
		SlidingWindowSize:     10,
		Timeout:               5 * time.Second,
		SlowCallDuration:      500 * time.Millisecond,
		SlowCallRateThreshold: 40,
		OnStateChange: func(name string, from, to breaker.State) {
			log.Printf("[BREAKER] %s: %v -> %v", name, from, to)
		},
	})

	// Analytics service: relaxed thresholds (non-critical)
	analyticsBreaker := breaker.New(breaker.Config{
		Name:                 "analytics-service",
		FailureRateThreshold: 70, // More tolerant
		MinimumNumberOfCalls: 20,
		SlidingWindowSize:    50,
		Timeout:              60 * time.Second,
	})

	// Simulate traffic
	for i := 0; i < 20; i++ {
		_, _ = paymentBreaker.Execute(ctx, SimulateServiceCall(0.2))
		_, _ = analyticsBreaker.Execute(ctx, SimulateServiceCall(0.5))
	}

	fmt.Printf("Payment breaker state: %v\n", paymentBreaker.State())
	fmt.Printf("Analytics breaker state: %v\n", analyticsBreaker.State())

	// === 2. Adaptive Circuit Breaker ===
	fmt.Println("\n=== Adaptive Circuit Breaker ===")

	ab := breaker.NewAdaptive(breaker.AdaptiveConfig{
		Base: breaker.Config{
			Name:                 "adaptive-service",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 5,
			Timeout:              10 * time.Second,
		},
		FailureRateEMAAlpha:     0.5,
		LatencyEMAAlpha:         0.3,
		MinFailureRateThreshold: 20,
		MaxFailureRateThreshold: 80,
		ConsecutiveFailureLimit: 5,
		SlowCallMultiplier:      2.0,
		TimeoutMultiplier:       1.5,
		MaxTimeout:              2 * time.Minute,
		OnAdaptiveChange: func(name string, params breaker.AdaptiveParams) {
			fmt.Printf("  [Adaptive] %s: threshold=%.1f%% latencyEMA=%v consecutive=%d\n",
				name, params.AdaptiveThreshold, params.LatencyEMA, params.ConsecutiveFailures)
		},
	})

	// Phase 1: Normal operation (low failure rate)
	fmt.Println("Phase 1: Normal operation")
	for i := 0; i < 15; i++ {
		_, _ = ab.Execute(ctx, SimulateServiceCall(0.1))
	}
	ab.UpdateEMAs()
	params := ab.GetAdaptiveParams()
	fmt.Printf("  Threshold: %.1f%%, Latency EMA: %v\n", params.AdaptiveThreshold, params.LatencyEMA)

	// Phase 2: Increased failures
	fmt.Println("Phase 2: Increased failures")
	for i := 0; i < 15; i++ {
		_, _ = ab.Execute(ctx, SimulateServiceCall(0.6))
	}
	ab.UpdateEMAs()
	params = ab.GetAdaptiveParams()
	fmt.Printf("  Threshold: %.1f%%, Latency EMA: %v, TripCount: %d\n",
		params.AdaptiveThreshold, params.LatencyEMA, params.TripCount)

	// === 3. Composite Executor with Fallback ===
	fmt.Println("\n=== Composite Executor with Graceful Degradation ===")

	// Primary executor (strict)
	primaryExecutor := resilience.NewExecutor(
		resilience.WithCircuitBreaker(breaker.Config{
			Name:                 "primary",
			FailureRateThreshold: 40,
			MinimumNumberOfCalls: 3,
			Timeout:              5 * time.Second,
		}),
		resilience.WithRetry(backoff.RetryConfig{
			Backoff: &backoff.ExponentialBackoff{
				InitialInterval: 50 * time.Millisecond,
				MaxInterval:     500 * time.Millisecond,
				MaxRetryCount:   2,
			},
		}),
		resilience.WithTimeout(2*time.Second),
	)

	// Full executor with all patterns
	fullExecutor := resilience.NewExecutor(
		resilience.WithCircuitBreaker(breaker.Config{
			Name:                 "api-gateway",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 10,
			Timeout:              30 * time.Second,
		}),
		resilience.WithRetry(backoff.RetryConfig{
			Backoff: &backoff.ExponentialRandomBackoff{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     5 * time.Second,
				MaxRetryCount:   3,
				Multiplier:      2.0,
			},
			RetryOn: func(err error) bool {
				return !errors.Is(err, context.Canceled)
			},
			OnRetry: func(retry int, err error) {
				fmt.Printf("  [Retry] attempt %d: %v\n", retry+1, err)
			},
		}),
		resilience.WithRateLimiter(ratelimit.NewTokenBucket(100, 200)),
		resilience.WithTimeout(10*time.Second),
		resilience.WithBulkhead(bulkhead.Config{
			MaxConcurrent:   50,
			MaxWaitDuration: 5 * time.Second,
		}),
	)

	// Execute with graceful degradation
	degradationChain := []struct {
		name    string
		executor *resilience.Executor
		failureRate float64
	}{
		{"primary", primaryExecutor, 0.7},   // High failure rate
		{"full-resilience", fullExecutor, 0.2}, // Low failure rate
	}

	for _, svc := range degradationChain {
		fmt.Printf("Trying %s...\n", svc.name)
		result, err := svc.executor.Execute(ctx, SimulateServiceCall(svc.failureRate))
		if err != nil {
			fmt.Printf("  %s failed: %v\n", svc.name, err)
			continue
		}
		fmt.Printf("  %s succeeded: %v\n", svc.name, result)
		break
	}

	// === 4. Metrics Collection ===
	fmt.Println("\n=== Metrics ===")

	registry := metrics.NewRegistry()

	// Reuse the payment breaker from above
	registry.Register(&metrics.BreakerCollector{
		Name:     paymentBreaker.Name(),
		GetState: func() int { return int(paymentBreaker.State()) },
		GetMetrics: func() (float64, float64, uint32, uint64, uint64, uint64, uint64, uint64) {
			m := paymentBreaker.GetMetrics()
			return m.FailureRate, m.SlowCallRate, m.TotalCalls,
				m.TotalSuccesses, m.TotalFailures, m.TotalRejected,
				m.TotalSlowCalls, m.StateTransitions
		},
	})

	limiter := ratelimit.NewTokenBucket(100, 200)
	registry.Register(&metrics.RateLimiterCollector{
		Name:     "api-limiter",
		GetRate:  limiter.Rate,
		GetBurst: limiter.Burst,
	})

	bh := bulkhead.New(bulkhead.Config{MaxConcurrent: 50})
	registry.Register(&metrics.BulkheadCollector{
		Name: "db-pool",
		GetMetrics: func() (int64, int64, int64, int64, int64) {
			return bh.GetMetricsForCollection()
		},
	})

	// Generate some traffic for metrics
	for i := 0; i < 10; i++ {
		_, _ = paymentBreaker.Execute(ctx, SimulateServiceCall(0.3))
	}

	// Print metrics
	handler := metrics.HTTPHandler(registry)
	w := &mockResponseWriter{}
	req, _ := http.NewRequest("GET", "/metrics", nil)
	handler.ServeHTTP(w, req)
	fmt.Println(w.body)

	fmt.Println("\n=== Advanced example complete ===")
}

// mockResponseWriter captures HTTP response for demo purposes
type mockResponseWriter struct {
	header http.Header
	body   string
	code   int
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}
func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.body += string(b)
	return len(b), nil
}
func (m *mockResponseWriter) WriteHeader(code int) { m.code = code }
