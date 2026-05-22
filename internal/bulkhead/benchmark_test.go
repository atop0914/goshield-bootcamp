package bulkhead

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Bulkhead Benchmarks
// =============================================================================

func BenchmarkBulkhead_Execute(b *testing.B) {
	bh := New(Config{
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

func BenchmarkBulkhead_Execute_Parallel(b *testing.B) {
	bh := New(Config{
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

func BenchmarkBulkhead_Contention(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", goroutines), func(b *testing.B) {
			bh := New(Config{
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

func BenchmarkBulkhead_Throughput(b *testing.B) {
	bh := New(Config{
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

func BenchmarkBulkhead_Throughput_Parallel(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bh.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkBulkhead_NoAlloc(b *testing.B) {
	bh := New(Config{
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

func BenchmarkBulkhead_NoAlloc_Parallel(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bh.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkBulkhead_ConcurrentAccess(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 1000000,
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
				bh.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// GetMetrics Benchmarks
// =============================================================================

func BenchmarkBulkhead_GetMetrics(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		bh.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bh.GetMetrics()
	}
}

func BenchmarkBulkhead_GetMetrics_Parallel(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		bh.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bh.GetMetrics()
		}
	})
}

func BenchmarkBulkhead_GetMetricsForCollection(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		bh.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bh.GetMetricsForCollection()
	}
}

func BenchmarkBulkhead_GetMetricsForCollection_Parallel(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		bh.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bh.GetMetricsForCollection()
		}
	})
}

// =============================================================================
// Edge Case Benchmarks
// =============================================================================

func BenchmarkBulkhead_SmallConcurrent(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 10,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bh.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkBulkhead_LargeConcurrent(b *testing.B) {
	bh := New(Config{
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

func BenchmarkBulkhead_WithWaitDuration(b *testing.B) {
	bh := New(Config{
		MaxConcurrent:   1000000,
		MaxWaitDuration: 1 * time.Second,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bh.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkBulkhead_WithWaitDuration_Parallel(b *testing.B) {
	bh := New(Config{
		MaxConcurrent:   1000000,
		MaxWaitDuration: 1 * time.Second,
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

// =============================================================================
// Mixed Operations Benchmarks
// =============================================================================

func BenchmarkBulkhead_MixedOperations(b *testing.B) {
	bh := New(Config{
		MaxConcurrent: 1000000,
	})

	ctx := context.Background()
	var ops uint64
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			op := ops % 100
			if op == 0 {
				// 1% reads
				bh.GetMetrics()
			} else {
				// 99% writes
				bh.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
			ops++
		}
	})
}
