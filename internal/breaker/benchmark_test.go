package breaker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// Circuit Breaker Benchmarks
// =============================================================================

func BenchmarkCircuitBreaker_Execute_Closed(b *testing.B) {
	cb := New(Config{
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

func BenchmarkCircuitBreaker_Execute_Parallel(b *testing.B) {
	cb := New(Config{
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

func BenchmarkCircuitBreaker_State(b *testing.B) {
	cb := New(Config{
		Name:                 "bench-state",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.State()
	}
}

func BenchmarkCircuitBreaker_State_Parallel(b *testing.B) {
	cb := New(Config{
		Name:                 "bench-state-parallel",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.State()
		}
	})
}

func BenchmarkCircuitBreaker_GetMetrics(b *testing.B) {
	cb := New(Config{
		Name:                 "bench-metrics",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		cb.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.GetMetrics()
	}
}

func BenchmarkCircuitBreaker_GetMetrics_Parallel(b *testing.B) {
	cb := New(Config{
		Name:                 "bench-metrics-parallel",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		cb.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.GetMetrics()
		}
	})
}

func BenchmarkCircuitBreaker_MixedReadWrite(b *testing.B) {
	cb := New(Config{
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

func BenchmarkCircuitBreaker_Contention(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", goroutines), func(b *testing.B) {
			cb := New(Config{
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

func BenchmarkCircuitBreaker_Throughput(b *testing.B) {
	cb := New(Config{
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

func BenchmarkCircuitBreaker_Throughput_Parallel(b *testing.B) {
	cb := New(Config{
		Name:                 "bench-throughput-parallel",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkCircuitBreaker_NoAlloc(b *testing.B) {
	cb := New(Config{
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

func BenchmarkCircuitBreaker_NoAlloc_Parallel(b *testing.B) {
	cb := New(Config{
		Name:                 "bench-noalloc-parallel",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkCircuitBreaker_StateTransition(b *testing.B) {
	cb := New(Config{
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

func BenchmarkCircuitBreaker_ConcurrentAccess(b *testing.B) {
	cb := New(Config{
		Name:                 "bench-concurrent",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 1000000,
	})

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 100
	opsPerGoroutine := b.N / numGoroutines

	b.ResetTimer()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				cb.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		}()
	}

	wg.Wait()
}
