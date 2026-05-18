package timeout

import (
	"context"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Timeout Benchmarks
// =============================================================================

func BenchmarkTimeout_Execute(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Execute(ctx, cfg, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkTimeout_Execute_Parallel(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
	}

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkTimeout_Execute_NoDuration(b *testing.B) {
	cfg := Config{
		Duration: 0,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Execute(ctx, cfg, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkTimeout_Execute_NoDuration_Parallel(b *testing.B) {
	cfg := Config{
		Duration: 0,
	}

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkTimeout_Throughput(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
	}

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Execute(ctx, cfg, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkTimeout_Throughput_Parallel(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
	}

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkTimeout_NoAlloc(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Execute(ctx, cfg, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkTimeout_NoAlloc_Parallel(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkTimeout_ConcurrentAccess(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
	}

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
				Execute(ctx, cfg, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// Timeout Struct Benchmarks
// =============================================================================

func BenchmarkTimeout_Struct_Execute(b *testing.B) {
	t := New(Config{
		Duration: 10 * time.Second,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkTimeout_Struct_Execute_Parallel(b *testing.B) {
	t := New(Config{
		Duration: 10 * time.Second,
	})

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkTimeout_Struct_GetMetrics(b *testing.B) {
	t := New(Config{
		Duration: 10 * time.Second,
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		t.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t.GetMetrics()
	}
}

func BenchmarkTimeout_Struct_GetMetrics_Parallel(b *testing.B) {
	t := New(Config{
		Duration: 10 * time.Second,
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		t.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.GetMetrics()
		}
	})
}

// =============================================================================
// Edge Case Benchmarks
// =============================================================================

func BenchmarkTimeout_SmallDuration(b *testing.B) {
	cfg := Config{
		Duration: 1 * time.Microsecond,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Execute(ctx, cfg, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkTimeout_LargeDuration(b *testing.B) {
	cfg := Config{
		Duration: 1 * time.Hour,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Execute(ctx, cfg, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkTimeout_WithCallback(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
		OnTimeout: func(duration time.Duration) {
			// No-op callback
		},
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Execute(ctx, cfg, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkTimeout_WithCallback_Parallel(b *testing.B) {
	cfg := Config{
		Duration: 10 * time.Second,
		OnTimeout: func(duration time.Duration) {
			// No-op callback
		},
	}

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

// =============================================================================
// Comparison Benchmarks
// =============================================================================

func BenchmarkTimeout_Comparison(b *testing.B) {
	b.Run("NoTimeout", func(b *testing.B) {
		cfg := Config{Duration: 0}
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})

	b.Run("WithTimeout", func(b *testing.B) {
		cfg := Config{Duration: 10 * time.Second}
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkTimeout_Comparison_Parallel(b *testing.B) {
	b.Run("NoTimeout", func(b *testing.B) {
		cfg := Config{Duration: 0}
		ctx := context.Background()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Execute(ctx, cfg, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		})
	})

	b.Run("WithTimeout", func(b *testing.B) {
		cfg := Config{Duration: 10 * time.Second}
		ctx := context.Background()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Execute(ctx, cfg, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		})
	})
}
