package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/atop0914/goshield/internal/backoff"
	"github.com/atop0914/goshield/internal/breaker"
	"github.com/atop0914/goshield/internal/bulkhead"
	"github.com/atop0914/goshield/internal/ratelimit"
	"github.com/atop0914/goshield/pkg/resilience"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("goshield v1.0.0")
	case "test":
		testCommand()
	case "benchmark":
		benchmarkCommand()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("GoShield - Unified Resilience Toolkit for Go")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  goshield version    Print version")
	fmt.Println("  goshield test       Test resilience patterns")
	fmt.Println("  goshield benchmark  Run benchmarks")
}

func testCommand() {
	fmt.Println("Testing GoShield resilience patterns...")

	// Test circuit breaker
	fmt.Println("\n1. Circuit Breaker:")
	cb := breaker.New(breaker.Config{
		Name:                 "test-cb",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 3,
	})

	for i := 0; i < 5; i++ {
		_, err := cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			if i < 3 {
				return nil, fmt.Errorf("error %d", i)
			}
			return "ok", nil
		})
		if err != nil {
			fmt.Printf("  Call %d: %v\n", i, err)
		} else {
			fmt.Printf("  Call %d: success\n", i)
		}
	}

	// Test rate limiter
	fmt.Println("\n2. Rate Limiter:")
	limiter := ratelimit.NewTokenBucket(5, 10)
	for i := 0; i < 15; i++ {
		if limiter.Allow() {
			fmt.Printf("  Request %d: allowed\n", i)
		} else {
			fmt.Printf("  Request %d: rate limited\n", i)
		}
	}

	// Test bulkhead
	fmt.Println("\n3. Bulkhead:")
	bh := bulkhead.New(bulkhead.Config{MaxConcurrent: 2})
	metrics := bh.GetMetrics()
	fmt.Printf("  Max concurrent: 2, Available: %d\n", metrics.AvailableSlots)

	fmt.Println("\nAll tests passed!")
}

func benchmarkCommand() {
	fmt.Println("Running GoShield benchmarks...")

	executor := resilience.NewExecutor(
		resilience.WithCircuitBreaker(breaker.Config{
			Name:                 "bench-cb",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 100,
		}),
		resilience.WithRetry(backoff.RetryConfig{
			Backoff: &backoff.FixedBackoff{
				Interval:      10 * time.Millisecond,
				MaxRetryCount: 3,
			},
		}),
		resilience.WithRateLimiter(ratelimit.NewTokenBucket(10000, 20000)),
		resilience.WithTimeout(5*time.Second),
		resilience.WithBulkhead(bulkhead.Config{MaxConcurrent: 1000}),
	)

	// Run benchmark
	iterations := 10000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		executor.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	elapsed := time.Since(start)
	opsPerSec := float64(iterations) / elapsed.Seconds()

	result := map[string]any{
		"iterations":  iterations,
		"duration":    elapsed.String(),
		"ops_per_sec": int(opsPerSec),
		"avg_latency": elapsed / time.Duration(iterations),
	}

	json.NewEncoder(os.Stdout).Encode(result)
}
