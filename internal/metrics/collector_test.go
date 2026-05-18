package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestRegistryEmpty(t *testing.T) {
	r := NewRegistry()
	text := r.PrometheusTextFormat()
	if text != "" {
		t.Errorf("expected empty output, got: %s", text)
	}
}

func TestRegistryNamespace(t *testing.T) {
	r := NewRegistryWithNamespace("myapp")
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{
			{Name: "test_counter", Value: 42, Type: Counter},
		}
	}))
	text := r.PrometheusTextFormat()
	if !strings.Contains(text, "myapp_test_counter") {
		t.Errorf("expected namespace prefix, got: %s", text)
	}
}

func TestCounterCollector(t *testing.T) {
	r := NewRegistryWithNamespace("") // No namespace for this test
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{
			{
				Name:      "requests_total",
				Labels:    []Label{{Name: "method", Value: "GET"}},
				Value:     100,
				Type:      Counter,
				Help:      "Total number of requests.",
				Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		}
	}))

	text := r.PrometheusTextFormat()
	if !strings.Contains(text, "# HELP requests_total Total number of requests.") {
		t.Errorf("missing HELP line, got: %s", text)
	}
	if !strings.Contains(text, "# TYPE requests_total counter") {
		t.Errorf("missing TYPE line, got: %s", text)
	}
	if !strings.Contains(text, `requests_total{method="GET"} 100`) {
		t.Errorf("missing metric value, got: %s", text)
	}
}

func TestGaugeCollector(t *testing.T) {
	r := NewRegistryWithNamespace("")
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{
			{
				Name:  "temperature",
				Value: 23.5,
				Type:  Gauge,
				Help:  "Current temperature.",
			},
		}
	}))

	text := r.PrometheusTextFormat()
	if !strings.Contains(text, "# TYPE temperature gauge") {
		t.Errorf("missing TYPE line for gauge, got: %s", text)
	}
	if !strings.Contains(text, "temperature 23.5") {
		t.Errorf("missing gauge value, got: %s", text)
	}
}

func TestMultipleLabels(t *testing.T) {
	r := NewRegistry()
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{
			{
				Name:   "http_requests",
				Labels: []Label{{Name: "method", Value: "GET"}, {Name: "status", Value: "200"}},
				Value:  50,
				Type:   Counter,
			},
		}
	}))

	text := r.PrometheusTextFormat()
	if !strings.Contains(text, `http_requests{method="GET",status="200"} 50`) {
		t.Errorf("expected multi-label metric, got: %s", text)
	}
}

func TestMultipleSamplesSameName(t *testing.T) {
	r := NewRegistryWithNamespace("")
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{
			{
				Name:   "http_requests",
				Labels: []Label{{Name: "status", Value: "200"}},
				Value:  50,
				Type:   Counter,
				Help:   "Total HTTP requests.",
			},
			{
				Name:   "http_requests",
				Labels: []Label{{Name: "status", Value: "500"}},
				Value:  5,
				Type:   Counter,
			},
		}
	}))

	text := r.PrometheusTextFormat()
	// HELP and TYPE should appear only once
	helpCount := strings.Count(text, "# HELP http_requests")
	typeCount := strings.Count(text, "# TYPE http_requests")
	if helpCount != 1 {
		t.Errorf("expected 1 HELP line, got %d, output: %s", helpCount, text)
	}
	if typeCount != 1 {
		t.Errorf("expected 1 TYPE line, got %d, output: %s", typeCount, text)
	}
}

func TestSimpleCounter(t *testing.T) {
	c := &SimpleCounter{}
	if c.Value() != 0 {
		t.Error("expected initial value 0")
	}
	c.Inc()
	if c.Value() != 1 {
		t.Errorf("expected 1, got %d", c.Value())
	}
	c.Add(5)
	if c.Value() != 6 {
		t.Errorf("expected 6, got %d", c.Value())
	}
}

func TestSimpleGauge(t *testing.T) {
	g := &SimpleGauge{}
	if g.Value() != 0 {
		t.Error("expected initial value 0")
	}
	g.Set(10)
	if g.Value() != 10 {
		t.Errorf("expected 10, got %d", g.Value())
	}
	g.Inc()
	if g.Value() != 11 {
		t.Errorf("expected 11, got %d", g.Value())
	}
	g.Dec()
	if g.Value() != 10 {
		t.Errorf("expected 10, got %d", g.Value())
	}
	g.Add(-5)
	if g.Value() != 5 {
		t.Errorf("expected 5, got %d", g.Value())
	}
}

func TestSimpleHistogram(t *testing.T) {
	h := NewSimpleHistogram([]float64{0.1, 0.5, 1.0, 5.0})

	h.Observe(0.05)
	h.Observe(0.3)
	h.Observe(0.8)
	h.Observe(2.0)
	h.Observe(10.0)

	sum, count, buckets := h.Snapshot()
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
	expectedSum := 0.05 + 0.3 + 0.8 + 2.0 + 10.0
	if sum != expectedSum {
		t.Errorf("expected sum %f, got %f", expectedSum, sum)
	}

	// Check buckets
	expected := []struct {
		ub    float64
		count uint64
	}{
		{0.1, 1},   // 0.05
		{0.5, 2},   // 0.05, 0.3
		{1.0, 3},   // 0.05, 0.3, 0.8
		{5.0, 4},   // 0.05, 0.3, 0.8, 2.0
	}

	for i, e := range expected {
		if buckets[i].UpperBound != e.ub {
			t.Errorf("bucket %d: expected ub %f, got %f", i, e.ub, buckets[i].UpperBound)
		}
		if buckets[i].Count != e.count {
			t.Errorf("bucket %d: expected count %d, got %d", i, e.count, buckets[i].Count)
		}
	}
}

func TestMultiCollector(t *testing.T) {
	mc := NewMultiCollector(
		CollectorFunc(func() []MetricSample {
			return []MetricSample{{Name: "a", Value: 1, Type: Counter}}
		}),
		CollectorFunc(func() []MetricSample {
			return []MetricSample{{Name: "b", Value: 2, Type: Gauge}}
		}),
	)

	samples := mc.Collect()
	if len(samples) != 2 {
		t.Errorf("expected 2 samples, got %d", len(samples))
	}
}

func TestGatherWithMultipleCollectors(t *testing.T) {
	r := NewRegistry()
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{{Name: "col1", Value: 10, Type: Counter}}
	}))
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{{Name: "col2", Value: 20, Type: Gauge}}
	}))

	samples := r.Gather()
	if len(samples) != 2 {
		t.Errorf("expected 2 samples, got %d", len(samples))
	}
}

func TestTakeSnapshot(t *testing.T) {
	r := NewRegistry()
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{{Name: "snap", Value: 42, Type: Counter}}
	}))

	snap := r.TakeSnapshot()
	if snap.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if len(snap.Metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(snap.Metrics))
	}
}

func TestBreakerCollector(t *testing.T) {
	collector := &BreakerCollector{
		Name:     "test-breaker",
		GetState: func() int { return 0 }, // Closed
		GetMetrics: func() (float64, float64, uint32, uint64, uint64, uint64, uint64, uint64) {
			return 10.0, 5.0, 100, 90, 10, 2, 3, 5
		},
	}

	samples := collector.Collect()
	if len(samples) != 9 {
		t.Errorf("expected 9 samples, got %d", len(samples))
	}

	// Check breaker_state
	found := false
	for _, s := range samples {
		if s.Name == "breaker_state" {
			found = true
			if s.Value != 1 {
				t.Errorf("expected state value 1, got %f", s.Value)
			}
			hasClosed := false
			for _, l := range s.Labels {
				if l.Name == "state" && l.Value == "closed" {
					hasClosed = true
				}
			}
			if !hasClosed {
				t.Error("expected closed state label")
			}
		}
	}
	if !found {
		t.Error("breaker_state metric not found")
	}
}

func TestBulkheadCollector(t *testing.T) {
	collector := &BulkheadCollector{
		Name: "test-bulkhead",
		GetMetrics: func() (int64, int64, int64, int64, int64) {
			return 5, 10, 50, 3, 5
		},
	}

	samples := collector.Collect()
	if len(samples) != 5 {
		t.Errorf("expected 5 samples, got %d", len(samples))
	}

	for _, s := range samples {
		if len(s.Labels) != 1 {
			t.Errorf("expected 1 label for %s, got %d", s.Name, len(s.Labels))
		}
	}
}

func TestRateLimiterCollector(t *testing.T) {
	collector := &RateLimiterCollector{
		Name:     "test-ratelimit",
		GetRate:  func() float64 { return 100.0 },
		GetBurst: func() int { return 200 },
	}

	samples := collector.Collect()
	if len(samples) != 2 {
		t.Errorf("expected 2 samples, got %d", len(samples))
	}
}

func TestRetryCollector(t *testing.T) {
	collector := &RetryCollector{
		Name: "test-retry",
		GetMetrics: func() (uint64, uint64, uint64, uint64) {
			return 100, 80, 20, 0
		},
	}

	samples := collector.Collect()
	if len(samples) != 3 {
		t.Errorf("expected 3 samples, got %d", len(samples))
	}
}

func TestTimeoutCollector(t *testing.T) {
	collector := &TimeoutCollector{
		Name: "test-timeout",
		GetMetrics: func() (uint64, uint64, uint64) {
			return 100, 5, 95
		},
	}

	samples := collector.Collect()
	if len(samples) != 3 {
		t.Errorf("expected 3 samples, got %d", len(samples))
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    int
		expected string
	}{
		{0, "closed"},
		{1, "open"},
		{2, "half_open"},
		{3, "disabled"},
		{4, "forced_open"},
		{99, "unknown_99"},
	}

	for _, tt := range tests {
		result := stateString(tt.state)
		if result != tt.expected {
			t.Errorf("stateString(%d) = %s, want %s", tt.state, result, tt.expected)
		}
	}
}

func TestMustRegisterPanicsOnNil(t *testing.T) {
	r := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Error("expected panic on nil collector")
		}
	}()
	r.MustRegister(nil)
}

func TestMetricTypeString(t *testing.T) {
	tests := []struct {
		t        MetricType
		expected string
	}{
		{Counter, "counter"},
		{Gauge, "gauge"},
		{Histogram, "histogram"},
		{Summary, "summary"},
		{MetricType(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.t.String() != tt.expected {
			t.Errorf("MetricType(%d).String() = %s, want %s", tt.t, tt.t.String(), tt.expected)
		}
	}
}
