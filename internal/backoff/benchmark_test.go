package backoff

import (
	"context"
	"testing"
	"time"
)

// =============================================================================
// Backoff Strategy Benchmarks
// =============================================================================

func BenchmarkFixedBackoff_Next(b *testing.B) {
	bo := &FixedBackoff{
		Interval:      1 * time.Millisecond,
		MaxRetryCount: 5,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 5)
	}
}

func BenchmarkFixedBackoff_Next_Parallel(b *testing.B) {
	bo := &FixedBackoff{
		Interval:      1 * time.Millisecond,
		MaxRetryCount: 5,
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			bo.Next(i % 5)
			i++
		}
	})
}

func BenchmarkExponentialBackoff_Next(b *testing.B) {
	bo := &ExponentialBackoff{
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

func BenchmarkExponentialBackoff_Next_Parallel(b *testing.B) {
	bo := &ExponentialBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
		Multiplier:      2.0,
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			bo.Next(i % 10)
			i++
		}
	})
}

func BenchmarkExponentialRandomBackoff_Next(b *testing.B) {
	bo := &ExponentialRandomBackoff{
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

func BenchmarkExponentialRandomBackoff_Next_Parallel(b *testing.B) {
	bo := &ExponentialRandomBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
		Multiplier:      2.0,
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			bo.Next(i % 10)
			i++
		}
	})
}

func BenchmarkFibonacciBackoff_Next(b *testing.B) {
	bo := &FibonacciBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 10)
	}
}

func BenchmarkFibonacciBackoff_Next_Parallel(b *testing.B) {
	bo := &FibonacciBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			bo.Next(i % 10)
			i++
		}
	})
}

// =============================================================================
// Retry Benchmarks
// =============================================================================

func BenchmarkRetry_Execute_Success(b *testing.B) {
	cfg := RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 3,
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

func BenchmarkRetry_Execute_Success_Parallel(b *testing.B) {
	cfg := RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 3,
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

func BenchmarkRetry_Execute_WithRetries(b *testing.B) {
	cfg := RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 3,
		},
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		attempt := 0
		Execute(ctx, cfg, func(ctx context.Context) (any, error) {
			attempt++
			if attempt < 3 {
				return nil, context.DeadlineExceeded
			}
			return "ok", nil
		})
	}
}

func BenchmarkRetry_Execute_WithRetries_Parallel(b *testing.B) {
	cfg := RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 3,
		},
	}

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			attempt := 0
			Execute(ctx, cfg, func(ctx context.Context) (any, error) {
				attempt++
				if attempt < 3 {
					return nil, context.DeadlineExceeded
				}
				return "ok", nil
			})
		}
	})
}

// =============================================================================
// RetryTracker Benchmarks
// =============================================================================

func BenchmarkRetryTracker_Execute(b *testing.B) {
	rt := NewRetryTracker(RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 3,
		},
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
}

func BenchmarkRetryTracker_Execute_Parallel(b *testing.B) {
	rt := NewRetryTracker(RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 3,
		},
	})

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rt.Execute(ctx, func(ctx context.Context) (any, error) {
				return "ok", nil
			})
		}
	})
}

func BenchmarkRetryTracker_GetMetrics(b *testing.B) {
	rt := NewRetryTracker(RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 3,
		},
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		rt.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.GetMetrics()
	}
}

func BenchmarkRetryTracker_GetMetrics_Parallel(b *testing.B) {
	rt := NewRetryTracker(RetryConfig{
		Backoff: &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 3,
		},
	})

	ctx := context.Background()
	// Populate some metrics
	for i := 0; i < 1000; i++ {
		rt.Execute(ctx, func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rt.GetMetrics()
		}
	})
}

// =============================================================================
// Comparison Benchmarks
// =============================================================================

func BenchmarkBackoffStrategies_Comparison(b *testing.B) {
	b.Run("Fixed", func(b *testing.B) {
		bo := &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 5,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bo.Next(i % 5)
		}
	})

	b.Run("Exponential", func(b *testing.B) {
		bo := &ExponentialBackoff{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			MaxRetryCount:   10,
			Multiplier:      2.0,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bo.Next(i % 10)
		}
	})

	b.Run("ExponentialRandom", func(b *testing.B) {
		bo := &ExponentialRandomBackoff{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			MaxRetryCount:   10,
			Multiplier:      2.0,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bo.Next(i % 10)
		}
	})

	b.Run("Fibonacci", func(b *testing.B) {
		bo := &FibonacciBackoff{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			MaxRetryCount:   10,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bo.Next(i % 10)
		}
	})
}

func BenchmarkBackoffStrategies_Comparison_Parallel(b *testing.B) {
	b.Run("Fixed", func(b *testing.B) {
		bo := &FixedBackoff{
			Interval:      1 * time.Millisecond,
			MaxRetryCount: 5,
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				bo.Next(i % 5)
				i++
			}
		})
	})

	b.Run("Exponential", func(b *testing.B) {
		bo := &ExponentialBackoff{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			MaxRetryCount:   10,
			Multiplier:      2.0,
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				bo.Next(i % 10)
				i++
			}
		})
	})

	b.Run("ExponentialRandom", func(b *testing.B) {
		bo := &ExponentialRandomBackoff{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			MaxRetryCount:   10,
			Multiplier:      2.0,
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				bo.Next(i % 10)
				i++
			}
		})
	})

	b.Run("Fibonacci", func(b *testing.B) {
		bo := &FibonacciBackoff{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			MaxRetryCount:   10,
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				bo.Next(i % 10)
				i++
			}
		})
	})
}

// =============================================================================
// Edge Case Benchmarks
// =============================================================================

func BenchmarkFixedBackoff_SmallInterval(b *testing.B) {
	bo := &FixedBackoff{
		Interval:      1 * time.Microsecond,
		MaxRetryCount: 5,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 5)
	}
}

func BenchmarkFixedBackoff_LargeInterval(b *testing.B) {
	bo := &FixedBackoff{
		Interval:      1 * time.Second,
		MaxRetryCount: 5,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 5)
	}
}

func BenchmarkExponentialBackoff_SmallMultiplier(b *testing.B) {
	bo := &ExponentialBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
		Multiplier:      1.5,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 10)
	}
}

func BenchmarkExponentialBackoff_LargeMultiplier(b *testing.B) {
	bo := &ExponentialBackoff{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxRetryCount:   10,
		Multiplier:      3.0,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bo.Next(i % 10)
	}
}

// =============================================================================
// Fibonacci Function Benchmarks
// =============================================================================

func BenchmarkFibonacci(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fibonacci(i % 20)
	}
}

func BenchmarkFibonacci_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			fibonacci(i % 20)
			i++
		}
	})
}

func BenchmarkFibonacci_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fibonacci(5)
	}
}

func BenchmarkFibonacci_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fibonacci(20)
	}
}
