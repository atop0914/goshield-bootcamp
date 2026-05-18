package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Token Bucket Benchmarks
// =============================================================================

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkTokenBucket_Allow_Parallel(b *testing.B) {
	tb := NewTokenBucket(10000000, 10000000)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow()
		}
	})
}

func BenchmarkTokenBucket_AllowN(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.AllowN(1)
	}
}

func BenchmarkTokenBucket_AllowN_Parallel(b *testing.B) {
	tb := NewTokenBucket(10000000, 10000000)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.AllowN(1)
		}
	})
}

func BenchmarkTokenBucket_Reserve(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r := tb.Reserve()
		_ = r
	}
}

func BenchmarkTokenBucket_Reserve_Parallel(b *testing.B) {
	tb := NewTokenBucket(10000000, 10000000)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := tb.Reserve()
			_ = r
		}
	})
}

func BenchmarkTokenBucket_Contention(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", goroutines), func(b *testing.B) {
			tb := NewTokenBucket(10000000, 10000000)
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

func BenchmarkTokenBucket_Throughput(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000)
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.Allow()
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkTokenBucket_Throughput_Parallel(b *testing.B) {
	tb := NewTokenBucket(10000000, 10000000)
	start := time.Now()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow()
		}
	})

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkTokenBucket_NoAlloc(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkTokenBucket_NoAlloc_Parallel(b *testing.B) {
	tb := NewTokenBucket(10000000, 10000000)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow()
		}
	})
}

func BenchmarkTokenBucket_ConcurrentAccess(b *testing.B) {
	tb := NewTokenBucket(10000000, 10000000)
	var wg sync.WaitGroup
	numGoroutines := 100
	opsPerGoroutine := b.N / numGoroutines

	b.ResetTimer()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				tb.Allow()
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// Sliding Window Benchmarks
// =============================================================================

func BenchmarkSlidingWindow_Allow(b *testing.B) {
	sw := NewSlidingWindow(1000000, 1*time.Minute)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sw.Allow()
	}
}

func BenchmarkSlidingWindow_Allow_Parallel(b *testing.B) {
	sw := NewSlidingWindow(10000000, 1*time.Minute)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sw.Allow()
		}
	})
}

func BenchmarkSlidingWindow_Reserve(b *testing.B) {
	sw := NewSlidingWindow(1000000, 1*time.Minute)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r := sw.Reserve()
		_ = r
	}
}

func BenchmarkSlidingWindow_Reserve_Parallel(b *testing.B) {
	sw := NewSlidingWindow(10000000, 1*time.Minute)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := sw.Reserve()
			_ = r
		}
	})
}

func BenchmarkSlidingWindow_Contention(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", goroutines), func(b *testing.B) {
			sw := NewSlidingWindow(10000000, 1*time.Minute)
			b.SetParallelism(goroutines)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					sw.Allow()
				}
			})
		})
	}
}

func BenchmarkSlidingWindow_Throughput(b *testing.B) {
	sw := NewSlidingWindow(1000000, 1*time.Minute)
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sw.Allow()
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec/1e6, "Mops/s")
}

func BenchmarkSlidingWindow_NoAlloc(b *testing.B) {
	sw := NewSlidingWindow(1000000, 1*time.Minute)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sw.Allow()
	}
}

// =============================================================================
// Wait Benchmarks
// =============================================================================

func BenchmarkTokenBucket_Wait(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.Wait(ctx)
	}
}

func BenchmarkTokenBucket_Wait_Parallel(b *testing.B) {
	tb := NewTokenBucket(10000000, 10000000)
	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Wait(ctx)
		}
	})
}

func BenchmarkSlidingWindow_Wait(b *testing.B) {
	sw := NewSlidingWindow(1000000, 1*time.Minute)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sw.Wait(ctx)
	}
}

func BenchmarkSlidingWindow_Wait_Parallel(b *testing.B) {
	sw := NewSlidingWindow(10000000, 1*time.Minute)
	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sw.Wait(ctx)
		}
	})
}

// =============================================================================
// Comparison Benchmarks
// =============================================================================

func BenchmarkRateLimiters_Comparison(b *testing.B) {
	b.Run("TokenBucket", func(b *testing.B) {
		tb := NewTokenBucket(1000000, 1000000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tb.Allow()
		}
	})

	b.Run("SlidingWindow", func(b *testing.B) {
		sw := NewSlidingWindow(1000000, 1*time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sw.Allow()
		}
	})
}

func BenchmarkRateLimiters_Comparison_Parallel(b *testing.B) {
	b.Run("TokenBucket", func(b *testing.B) {
		tb := NewTokenBucket(10000000, 10000000)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tb.Allow()
			}
		})
	})

	b.Run("SlidingWindow", func(b *testing.B) {
		sw := NewSlidingWindow(10000000, 1*time.Minute)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				sw.Allow()
			}
		})
	})
}

// =============================================================================
// Edge Case Benchmarks
// =============================================================================

func BenchmarkTokenBucket_SmallBurst(b *testing.B) {
	tb := NewTokenBucket(1000, 10)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkTokenBucket_LargeBurst(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkSlidingWindow_SmallLimit(b *testing.B) {
	sw := NewSlidingWindow(10, 1*time.Minute)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sw.Allow()
	}
}

func BenchmarkSlidingWindow_LargeLimit(b *testing.B) {
	sw := NewSlidingWindow(1000000, 1*time.Minute)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sw.Allow()
	}
}
