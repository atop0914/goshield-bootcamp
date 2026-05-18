package breaker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Adaptive Breaker Benchmarks
// =============================================================================

func BenchmarkAdaptiveBreaker_Execute_Closed(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
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

func BenchmarkAdaptiveBreaker_Execute_Parallel(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
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

func BenchmarkAdaptiveBreaker_Execute_WithUpdateEMAs(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-emas",
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

func BenchmarkAdaptiveBreaker_Execute_WithUpdateEMAs_Parallel(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-emas-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ab.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
			if i%100 == 0 {
				ab.UpdateEMAs()
			}
			i++
		}
	})
}

func BenchmarkAdaptiveBreaker_Contention(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", goroutines), func(b *testing.B) {
			ab := NewAdaptive(AdaptiveConfig{
				Base: Config{
					Name:                 "bench-adaptive-contention",
					FailureRateThreshold: 50,
					MinimumNumberOfCalls: 1000000,
				},
			})

			ctx := context.Background()
			b.SetParallelism(goroutines)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					ab.Execute(ctx, func(ctx context.Context) (any, error) {
						return "ok", nil
					})
				}
			})
		})
	}
}

func BenchmarkAdaptiveBreaker_Throughput(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-throughput",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkAdaptiveBreaker_Throughput_Parallel(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-throughput-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	start := time.Now()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ab.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkAdaptiveBreaker_NoAlloc(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-noalloc",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkAdaptiveBreaker_NoAlloc_Parallel(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-noalloc-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ab.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkAdaptiveBreaker_ConcurrentAccess(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-concurrent",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
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
				ab.Execute(ctx, func(ctx context.Context) (any, error) {
					return "ok", nil
				})
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// UpdateEMAs Benchmarks
// =============================================================================

func BenchmarkAdaptiveBreaker_UpdateEMAs(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-emas",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	// Populate some data
	for i := 0; i < 1000; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.UpdateEMAs()
	}
}

func BenchmarkAdaptiveBreaker_UpdateEMAs_Parallel(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-emas-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	// Populate some data
	for i := 0; i < 1000; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ab.UpdateEMAs()
		}
	})
}

// =============================================================================
// GetAdaptiveParams Benchmarks
// =============================================================================

func BenchmarkAdaptiveBreaker_GetAdaptiveParams(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-params",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	// Populate some data
	for i := 0; i < 1000; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.GetAdaptiveParams()
	}
}

func BenchmarkAdaptiveBreaker_GetAdaptiveParams_Parallel(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-params-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	// Populate some data
	for i := 0; i < 1000; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ab.GetAdaptiveParams()
		}
	})
}

// =============================================================================
// GetMetrics Benchmarks
// =============================================================================

func BenchmarkAdaptiveBreaker_GetMetrics(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-metrics",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.GetMetrics()
	}
}

func BenchmarkAdaptiveBreaker_GetMetrics_Parallel(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-metrics-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ab.GetMetrics()
		}
	})
}

// =============================================================================
// Comparison Benchmarks
// =============================================================================

func BenchmarkAdaptiveVsRegular_Comparison(b *testing.B) {
	b.Run("Regular", func(b *testing.B) {
		cb := New(Config{
			Name:                 "bench-regular",
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

	b.Run("Adaptive", func(b *testing.B) {
		ab := NewAdaptive(AdaptiveConfig{
			Base: Config{
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
}

func BenchmarkAdaptiveVsRegular_Comparison_Parallel(b *testing.B) {
	b.Run("Regular", func(b *testing.B) {
		cb := New(Config{
			Name:                 "bench-regular-parallel",
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

	b.Run("Adaptive", func(b *testing.B) {
		ab := NewAdaptive(AdaptiveConfig{
			Base: Config{
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
}

// =============================================================================
// Edge Case Benchmarks
// =============================================================================

func BenchmarkAdaptiveBreaker_WithConsecutiveFailureLimit(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-consfail",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
		ConsecutiveFailureLimit: 10,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkAdaptiveBreaker_WithSlowCallMultiplier(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-slowcall",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
		},
		SlowCallMultiplier:  2.0,
		MinSlowCallDuration: 100 * time.Millisecond,
		MaxSlowCallDuration: 30 * time.Second,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkAdaptiveBreaker_WithTimeoutMultiplier(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-timeout",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
			Timeout:              60 * time.Second,
		},
		TimeoutMultiplier: 2.0,
		MaxTimeout:        5 * time.Minute,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkAdaptiveBreaker_WithAllFeatures(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-all",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
			Timeout:              60 * time.Second,
		},
		ConsecutiveFailureLimit: 10,
		SlowCallMultiplier:      2.0,
		MinSlowCallDuration:     100 * time.Millisecond,
		MaxSlowCallDuration:     30 * time.Second,
		TimeoutMultiplier:       2.0,
		MaxTimeout:              5 * time.Minute,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ab.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkAdaptiveBreaker_WithAllFeatures_Parallel(b *testing.B) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "bench-adaptive-all-parallel",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 1000000,
			Timeout:              60 * time.Second,
		},
		ConsecutiveFailureLimit: 10,
		SlowCallMultiplier:      2.0,
		MinSlowCallDuration:     100 * time.Millisecond,
		MaxSlowCallDuration:     30 * time.Second,
		TimeoutMultiplier:       2.0,
		MaxTimeout:              5 * time.Minute,
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
