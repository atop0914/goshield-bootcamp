package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/atop0914/goshield/internal/backoff"
	"github.com/atop0914/goshield/internal/breaker"
	"github.com/atop0914/goshield/internal/bulkhead"
	"github.com/atop0914/goshield/internal/ratelimit"
	"github.com/atop0914/goshield/internal/timeout"
	"github.com/atop0914/goshield/pkg/resilience"
)

// =============================================================================
// Single-threaded Benchmarks (Baseline)
// =============================================================================

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
			Interval:      1 * time.Millisecond,
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
				Interval:      1 * time.Millisecond,
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

func BenchmarkAdaptiveBreaker_Closed(b *testing.B) {
	ab := breaker.NewAdaptive(breaker.AdaptiveConfig{
		Base: breaker.Config{
			Name:                 "bench-adaptive",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkAdaptiveBreaker_WithUpdateEMAs(b *testing.B) {
	ab := breaker.NewAdaptive(breaker.AdaptiveConfig{
		Base: breaker.Config{
			Name:                 "bench-adaptive",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
		if i%100 == 0 {
			ab.UpdateEMAs()
		}
	}
}

// =============================================================================
// Parallel Benchmarks (Multi-Goroutine Contention)
// =============================================================================

func BenchmarkCircuitBreaker_Parallel(b *testing.B) {
	cb := breaker.New(breaker.Config{
		Name:                 "bench-parallel",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkRateLimiter_TokenBucket_Parallel(b *testing.B) {
	tb := ratelimit.NewTokenBucket(10000000, 10000000)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow()
		}
	})
}

func BenchmarkBulkhead_Parallel(b *testing.B) {
	bh := bulkhead.New(bulkhead.Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bh.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkAdaptiveBreaker_Parallel(b *testing.B) {
	ab := breaker.NewAdaptive(breaker.AdaptiveConfig{
		Base: breaker.Config{
			Name:                 "bench-adaptive-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ab.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkCompositeExecutor_Parallel(b *testing.B) {
	executor := resilience.NewExecutor(
		resilience.WithCircuitBreaker(breaker.Config{
			Name:                 "bench-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		}),
		resilience.WithRetry(backoff.RetryConfig{
			Backoff: &backoff.FixedBackoff{
				Interval:      1 * time.Millisecond,
				MaxRetryCount: 3,
			},
		}),
		resilience.WithRateLimiter(ratelimit.NewTokenBucket(10000000, 10000000)),
		resilience.WithTimeout(10*time.Second),
		resilience.WithBulkhead(bulkhead.Config{MaxConcurrent: 1000000}),
	)

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			executor.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

// =============================================================================
// Contention Benchmarks (Realistic High-Concurrency Scenarios)
// =============================================================================

func BenchmarkCircuitBreaker_Contention(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", goroutines), func(b *testing.B) {
			cb := breaker.New(breaker.Config{
				Name:                 "bench-contention",
				FailureRateThreshold: 50,
				MinimumNumberOfCalls: 1000000,
			})

			ctx := context.Background()
			b.SetParallelism(goroutines)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					cb.Execute(ctx, func(ctx context.Context) (any, error) {
						return "ok", nil
					})
				}
			})
		})
	}
}

func BenchmarkRateLimiter_Contention(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", goroutines), func(b *testing.B) {
			tb := ratelimit.NewTokenBucket(10000000, 10000000)
			b.SetParallelism(goroutines)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					tb.Allow()
				}
			})
		})
	}
}

func BenchmarkBulkhead_Contention(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", goroutines), func(b *testing.B) {
			bh := bulkhead.New(bulkhead.Config{
				MaxConcurrent: 1000000,
			})

			ctx := context.Background()
			b.SetParallelism(goroutines)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					bh.Execute(ctx, func(ctx context.Context) (any, error) {
						return "ok", nil
					})
				}
			})
		})
	}
}

// =============================================================================
// Allocation-Free Path Benchmarks
// =============================================================================

func BenchmarkCircuitBreaker_NoAlloc(b *testing.B) {
	cb := breaker.New(breaker.Config{
		Name:                 "bench-noalloc",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkRateLimiter_NoAlloc(b *testing.B) {
	tb := ratelimit.NewTokenBucket(1000000, 1000000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkBulkhead_NoAlloc(b *testing.B) {
	bh := bulkhead.New(bulkhead.Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bh.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

// =============================================================================
// Backoff Strategy Benchmarks
// =============================================================================

func BenchmarkBackoff_Fixed(b *testing.B) {
	bo := &backoff.FixedBackoff{
		Interval:      1 * time.Millisecond,
		MaxRetryCount: 5,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 5)
	}
}

func BenchmarkBackoff_Exponential(b *testing.B) {
	bo := &backoff.ExponentialBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
		Multiplier:      2.0,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 10)
	}
}

func BenchmarkBackoff_ExponentialRandom(b *testing.B) {
	bo := &backoff.ExponentialRandomBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
		Multiplier:      2.0,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 10)
	}
}

func BenchmarkBackoff_Fibonacci(b *testing.B) {
	bo := &backoff.FibonacciBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 10)
	}
}

// =============================================================================
// State Transition Benchmarks
// =============================================================================

func BenchmarkCircuitBreaker_StateTransition(b *testing.B) {
	cb := breaker.New(breaker.Config{
		Name:                 "bench-transition",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 2,
		Timeout:              1 * time.Microsecond,
	})

	ctx := context.Background()
	failFn := func(ctx context.Context) (any, error) {
		return nil, fmt.Errorf("fail")
	}
	successFn := func(ctx context.Context) (any, error) {
		return "ok", nil
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Trip the breaker
		cb.Execute(ctx, failFn)
		cb.Execute(ctx, failFn)
		// Wait for timeout
		time.Sleep(2 * time.Microsecond)
		// Recover
		cb.Execute(ctx, successFn)
	}
}

// =============================================================================
// Concurrent Access Patterns
// =============================================================================

func BenchmarkCircuitBreaker_ReadHeavy(b *testing.B) {
	cb := breaker.New(breaker.Config{
		Name:                 "bench-readheavy",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of reads (State()) and writes (Execute())
			cb.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkCircuitBreaker_MixedOperations(b *testing.B) {
	cb := breaker.New(breaker.Config{
		Name:                 "bench-mixed",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	var ops uint64
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			op := atomic.AddUint64(&ops, 1)
			if op%100 == 0 {
				// 1% reads
				cb.GetMetrics()
			} else {
				// 99% writes
				cb.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		}
	})
}

// =============================================================================
// Throughput Benchmarks
// =============================================================================

func BenchmarkCircuitBreaker_Throughput(b *testing.B) {
	cb := breaker.New(breaker.Config{
		Name:                 "bench-throughput",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkRateLimiter_Throughput(b *testing.B) {
	tb := ratelimit.NewTokenBucket(1000000, 1000000)
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.Allow()
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkBulkhead_Throughput(b *testing.B) {
	bh := bulkhead.New(bulkhead.Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bh.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

// =============================================================================
// Memory Pool Benchmarks
// =============================================================================

func BenchmarkCircuitBreaker_WithPool(b *testing.B) {
	pool := sync.Pool{
		New: func() any {
			return &breaker.CircuitBreaker{}
		},
	}

	cb := breaker.New(breaker.Config{
		Name:                 "bench-pool",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = pool.Get()
		cb.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
		pool.Put(cb)
	}
}

// =============================================================================
// Comparison Benchmarks
// =============================================================================

func BenchmarkAllPatterns_Single(b *testing.B) {
	b.Run("CircuitBreaker", func(b *testing.B) {
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
	})

	b.Run("AdaptiveBreaker", func(b *testing.B) {
		ab := breaker.NewAdaptive(breaker.AdaptiveConfig{
			Base: breaker.Config{
				Name:                 "bench-adaptive",
				FailureRateThreshold: 50,
				MinimumNumberOfCalls: 1000000,
			},
		})
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ab.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})

	b.Run("RateLimiter", func(b *testing.B) {
		tb := ratelimit.NewTokenBucket(1000000, 1000000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tb.Allow()
		}
	})

	b.Run("Bulkhead", func(b *testing.B) {
		bh := bulkhead.New(bulkhead.Config{MaxConcurrent: 1000000})
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bh.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})

	b.Run("Timeout", func(b *testing.B) {
		cfg := timeout.Config{Duration: 10 * time.Second}
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			timeout.Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})

	b.Run("Retry", func(b *testing.B) {
		cfg := backoff.RetryConfig{
			Backoff: &backoff.FixedBackoff{
				Interval:      1 * time.Millisecond,
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
	})

	b.Run("Composite", func(b *testing.B) {
		executor := resilience.NewExecutor(
			resilience.WithCircuitBreaker(breaker.Config{
				Name:                 "bench",
				FailureRateThreshold: 50,
				MinimumNumberOfCalls: 1000000,
			}),
			resilience.WithRetry(backoff.RetryConfig{
				Backoff: &backoff.FixedBackoff{
					Interval:      1 * time.Millisecond,
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
	})
}

func BenchmarkAllPatterns_Parallel(b *testing.B) {
	b.Run("CircuitBreaker", func(b *testing.B) {
		cb := breaker.New(breaker.Config{
			Name:                 "bench-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		})
		ctx := context.Background()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		})
	})

	b.Run("AdaptiveBreaker", func(b *testing.B) {
		ab := breaker.NewAdaptive(breaker.AdaptiveConfig{
			Base: breaker.Config{
				Name:                 "bench-adaptive-parallel",
				FailureRateThreshold: 50,
				MinimumNumberOfCalls: 1000000,
			},
		})
		ctx := context.Background()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ab.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		})
	})

	b.Run("RateLimiter", func(b *testing.B) {
		tb := ratelimit.NewTokenBucket(10000000, 10000000)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tb.Allow()
			}
		})
	})

	b.Run("Bulkhead", func(b *testing.B) {
		bh := bulkhead.New(bulkhead.Config{MaxConcurrent: 1000000})
		ctx := context.Background()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				bh.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		})
	})

	b.Run("Composite", func(b *testing.B) {
		executor := resilience.NewExecutor(
			resilience.WithCircuitBreaker(breaker.Config{
				Name:                 "bench-parallel",
				FailureRateThreshold: 50,
				MinimumNumberOfCalls: 1000000,
			}),
			resilience.WithRetry(backoff.RetryConfig{
				Backoff: &backoff.FixedBackoff{
					Interval:      1 * time.Millisecond,
					MaxRetryCount: 3,
				},
			}),
			resilience.WithRateLimiter(ratelimit.NewTokenBucket(10000000, 10000000)),
			resilience.WithTimeout(10*time.Second),
			resilience.WithBulkhead(bulkhead.Config{MaxConcurrent: 1000000}),
		)
		ctx := context.Background()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				executor.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		})
	})
}

// =============================================================================
// Benchmark Helpers
// =============================================================================

func BenchmarkHelper_CalculateOpsPerSec(b *testing.B) {
	// This benchmark demonstrates how to calculate ops/sec
	cb := breaker.New(breaker.Config{
		Name:                 "bench-helper",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
	b.ReportMetric(float64(elapsed.Nanoseconds())/float64(b.N), "ns/op")
}
