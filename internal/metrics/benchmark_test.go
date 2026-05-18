package metrics

import (
	"sync"
	"testing"
)

// =============================================================================
// SimpleCounter Benchmarks
// =============================================================================

func BenchmarkSimpleCounter_Inc(b *testing.B) {
	c := &SimpleCounter{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

func BenchmarkSimpleCounter_Inc_Parallel(b *testing.B) {
	c := &SimpleCounter{}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkSimpleCounter_Add(b *testing.B) {
	c := &SimpleCounter{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Add(1)
	}
}

func BenchmarkSimpleCounter_Add_Parallel(b *testing.B) {
	c := &SimpleCounter{}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Add(1)
		}
	})
}

func BenchmarkSimpleCounter_Value(b *testing.B) {
	c := &SimpleCounter{}
	c.Add(1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Value()
	}
}

func BenchmarkSimpleCounter_Value_Parallel(b *testing.B) {
	c := &SimpleCounter{}
	c.Add(1000)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Value()
		}
	})
}

func BenchmarkSimpleCounter_NoAlloc(b *testing.B) {
	c := &SimpleCounter{}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

func BenchmarkSimpleCounter_NoAlloc_Parallel(b *testing.B) {
	c := &SimpleCounter{}
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

// =============================================================================
// SimpleGauge Benchmarks
// =============================================================================

func BenchmarkSimpleGauge_Set(b *testing.B) {
	g := &SimpleGauge{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Set(int64(i))
	}
}

func BenchmarkSimpleGauge_Set_Parallel(b *testing.B) {
	g := &SimpleGauge{}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			g.Set(int64(i))
			i++
		}
	})
}

func BenchmarkSimpleGauge_Inc(b *testing.B) {
	g := &SimpleGauge{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Inc()
	}
}

func BenchmarkSimpleGauge_Inc_Parallel(b *testing.B) {
	g := &SimpleGauge{}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Inc()
		}
	})
}

func BenchmarkSimpleGauge_Dec(b *testing.B) {
	g := &SimpleGauge{}
	g.Set(1000000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Dec()
	}
}

func BenchmarkSimpleGauge_Dec_Parallel(b *testing.B) {
	g := &SimpleGauge{}
	g.Set(10000000)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Dec()
		}
	})
}

func BenchmarkSimpleGauge_Add(b *testing.B) {
	g := &SimpleGauge{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Add(1)
	}
}

func BenchmarkSimpleGauge_Add_Parallel(b *testing.B) {
	g := &SimpleGauge{}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Add(1)
		}
	})
}

func BenchmarkSimpleGauge_Value(b *testing.B) {
	g := &SimpleGauge{}
	g.Set(1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Value()
	}
}

func BenchmarkSimpleGauge_Value_Parallel(b *testing.B) {
	g := &SimpleGauge{}
	g.Set(1000)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Value()
		}
	})
}

func BenchmarkSimpleGauge_NoAlloc(b *testing.B) {
	g := &SimpleGauge{}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Inc()
	}
}

func BenchmarkSimpleGauge_NoAlloc_Parallel(b *testing.B) {
	g := &SimpleGauge{}
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Inc()
		}
	})
}

// =============================================================================
// SimpleHistogram Benchmarks
// =============================================================================

func BenchmarkSimpleHistogram_Observe(b *testing.B) {
	h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.Observe(float64(i%10) * 0.1)
	}
}

func BenchmarkSimpleHistogram_Observe_Parallel(b *testing.B) {
	h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			h.Observe(float64(i%10) * 0.1)
			i++
		}
	})
}

func BenchmarkSimpleHistogram_Snapshot(b *testing.B) {
	h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
	// Populate some data
	for i := 0; i < 1000; i++ {
		h.Observe(float64(i%10) * 0.1)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.Snapshot()
	}
}

func BenchmarkSimpleHistogram_Snapshot_Parallel(b *testing.B) {
	h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
	// Populate some data
	for i := 0; i < 1000; i++ {
		h.Observe(float64(i%10) * 0.1)
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.Snapshot()
		}
	})
}

func BenchmarkSimpleHistogram_NoAlloc(b *testing.B) {
	h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.Observe(float64(i%10) * 0.1)
	}
}

func BenchmarkSimpleHistogram_NoAlloc_Parallel(b *testing.B) {
	h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			h.Observe(float64(i%10) * 0.1)
			i++
		}
	})
}

// =============================================================================
// Registry Benchmarks
// =============================================================================

func BenchmarkRegistry_Gather(b *testing.B) {
	r := NewRegistry()
	// Use a mock collector for benchmarking
	r.MustRegister(&mockCollector{count: 10})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Gather()
	}
}

func BenchmarkRegistry_Gather_Parallel(b *testing.B) {
	r := NewRegistry()
	// Use a mock collector for benchmarking
	r.MustRegister(&mockCollector{count: 10})
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.Gather()
		}
	})
}

func BenchmarkRegistry_PrometheusTextFormat(b *testing.B) {
	r := NewRegistry()
	// Use a mock collector for benchmarking
	r.MustRegister(&mockCollector{count: 10})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.PrometheusTextFormat()
	}
}

func BenchmarkRegistry_PrometheusTextFormat_Parallel(b *testing.B) {
	r := NewRegistry()
	// Use a mock collector for benchmarking
	r.MustRegister(&mockCollector{count: 10})
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.PrometheusTextFormat()
		}
	})
}

func BenchmarkRegistry_Gather_MultipleCollectors(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 10; i++ {
		r.MustRegister(&mockCollector{count: 10})
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Gather()
	}
}

func BenchmarkRegistry_Gather_MultipleCollectors_Parallel(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 10; i++ {
		r.MustRegister(&mockCollector{count: 10})
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.Gather()
		}
	})
}

func BenchmarkRegistry_PrometheusTextFormat_MultipleCollectors(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 10; i++ {
		r.MustRegister(&mockCollector{count: 10})
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.PrometheusTextFormat()
	}
}

func BenchmarkRegistry_PrometheusTextFormat_MultipleCollectors_Parallel(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 10; i++ {
		r.MustRegister(&mockCollector{count: 10})
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.PrometheusTextFormat()
		}
	})
}

// =============================================================================
// Concurrent Access Benchmarks
// =============================================================================

func BenchmarkSimpleCounter_ConcurrentAccess(b *testing.B) {
	c := &SimpleCounter{}
	var wg sync.WaitGroup
	numGoroutines := 100
	opsPerGoroutine := b.N / numGoroutines

	b.ResetTimer()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				c.Inc()
			}
		}()
	}

	wg.Wait()
}

func BenchmarkSimpleGauge_ConcurrentAccess(b *testing.B) {
	g := &SimpleGauge{}
	var wg sync.WaitGroup
	numGoroutines := 100
	opsPerGoroutine := b.N / numGoroutines

	b.ResetTimer()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				g.Inc()
			}
		}()
	}

	wg.Wait()
}

func BenchmarkSimpleHistogram_ConcurrentAccess(b *testing.B) {
	h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
	var wg sync.WaitGroup
	numGoroutines := 100
	opsPerGoroutine := b.N / numGoroutines

	b.ResetTimer()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				h.Observe(float64(j%10) * 0.1)
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// Comparison Benchmarks
// =============================================================================

func BenchmarkMetricsTypes_Comparison(b *testing.B) {
	b.Run("Counter", func(b *testing.B) {
		c := &SimpleCounter{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Inc()
		}
	})

	b.Run("Gauge", func(b *testing.B) {
		g := &SimpleGauge{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			g.Inc()
		}
	})

	b.Run("Histogram", func(b *testing.B) {
		h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.Observe(float64(i%10) * 0.1)
		}
	})
}

func BenchmarkMetricsTypes_Comparison_Parallel(b *testing.B) {
	b.Run("Counter", func(b *testing.B) {
		c := &SimpleCounter{}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.Inc()
			}
		})
	})

	b.Run("Gauge", func(b *testing.B) {
		g := &SimpleGauge{}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				g.Inc()
			}
		})
	})

	b.Run("Histogram", func(b *testing.B) {
		h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				h.Observe(float64(i%10) * 0.1)
				i++
			}
		})
	})
}

// =============================================================================
// Edge Case Benchmarks
// =============================================================================

func BenchmarkSimpleHistogram_SmallBuckets(b *testing.B) {
	h := NewSimpleHistogram([]float64{1.0})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.Observe(float64(i%10) * 0.1)
	}
}

func BenchmarkSimpleHistogram_LargeBuckets(b *testing.B) {
	buckets := make([]float64, 100)
	for i := range buckets {
		buckets[i] = float64(i+1) * 0.1
	}
	h := NewSimpleHistogram(buckets)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.Observe(float64(i%100) * 0.1)
	}
}

func BenchmarkLabels(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Labels("method", "GET", "status", "200", "path", "/api/v1")
	}
}

func BenchmarkLabels_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Labels("method", "GET", "status", "200", "path", "/api/v1")
		}
	})
}

// =============================================================================
// Mock Collector for Registry Benchmarks
// =============================================================================

type mockCollector struct {
	count int
}

func (m *mockCollector) Collect() []MetricSample {
	samples := make([]MetricSample, m.count)
	for i := 0; i < m.count; i++ {
		samples[i] = MetricSample{
			Name:  "mock_metric",
			Value: float64(i),
			Type:  Counter,
			Help:  "Mock metric for benchmarking",
		}
	}
	return samples
}
