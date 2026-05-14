// Package metrics provides a zero-dependency metrics collection and
// Prometheus-compatible exposition system for GoShield resilience patterns.
//
// It defines collector interfaces that all resilience patterns implement,
// a central registry for collecting metrics, and a Prometheus text format
// exporter that can be served via HTTP.
//
// Example:
//
//	registry := metrics.NewRegistry()
//	registry.MustRegister(myBreaker)
//	registry.MustRegister(myRateLimiter)
//
//	http.Handle("/metrics", metrics.HTTPHandler(registry))
package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MetricType represents the type of a Prometheus metric.
type MetricType int

const (
	// Counter is a monotonically increasing value.
	Counter MetricType = iota
	// Gauge is a value that can go up and down.
	Gauge
	// Histogram tracks the distribution of values.
	Histogram
	// Summary tracks quantiles of values.
	Summary
)

func (t MetricType) String() string {
	switch t {
	case Counter:
		return "counter"
	case Gauge:
		return "gauge"
	case Histogram:
		return "histogram"
	case Summary:
		return "summary"
	default:
		return "unknown"
	}
}

// Label is a key-value pair for metric labels.
type Label struct {
	Name  string
	Value string
}

// Labels creates a sorted slice of labels.
func Labels(pairs ...string) []Label {
	if len(pairs)%2 != 0 {
		panic("metrics.Labels requires even number of arguments")
	}
	labels := make([]Label, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		labels = append(labels, Label{Name: pairs[i], Value: pairs[i+1]})
	}
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})
	return labels
}

// MetricSample represents a single metric data point.
type MetricSample struct {
	Name      string
	Labels    []Label
	Value     float64
	Type      MetricType
	Help      string
	Timestamp time.Time
}

// Collector is the interface that resilience patterns implement to expose metrics.
type Collector interface {
	// Collect returns all metric samples from this collector.
	Collect() []MetricSample
}

// Registry collects metrics from multiple collectors.
type Registry struct {
	mu         sync.RWMutex
	collectors []Collector
	namespace  string
}

// NewRegistry creates a new metrics registry.
func NewRegistry() *Registry {
	return &Registry{
		namespace: "goshield",
	}
}

// NewRegistryWithNamespace creates a registry with a custom namespace prefix.
func NewRegistryWithNamespace(ns string) *Registry {
	return &Registry{
		namespace: ns,
	}
}

// Register adds a collector to the registry.
func (r *Registry) Register(c Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors = append(r.collectors, c)
}

// MustRegister adds a collector and panics on nil (for convenience).
func (r *Registry) MustRegister(c Collector) {
	if c == nil {
		panic("metrics: Register called with nil collector")
	}
	r.Register(c)
}

// Gather collects all metrics from registered collectors.
func (r *Registry) Gather() []MetricSample {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []MetricSample
	for _, c := range r.collectors {
		samples := c.Collect()
		// Apply namespace prefix
		for i := range samples {
			if r.namespace != "" {
				samples[i].Name = r.namespace + "_" + samples[i].Name
			}
		}
		all = append(all, samples...)
	}
	return all
}

// PrometheusTextFormat returns metrics in Prometheus text exposition format.
// This is compatible with Prometheus scraping and can be served via HTTP.
func (r *Registry) PrometheusTextFormat() string {
	samples := r.Gather()
	if len(samples) == 0 {
		return ""
	}

	var sb strings.Builder
	seen := make(map[string]bool)

	for _, s := range samples {
		// Write HELP and TYPE only once per metric name
		if !seen[s.Name] {
			seen[s.Name] = true
			if s.Help != "" {
				sb.WriteString(fmt.Sprintf("# HELP %s %s\n", s.Name, s.Help))
			}
			sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", s.Name, s.Type))
		}

		// Write the sample
		sb.WriteString(s.Name)
		if len(s.Labels) > 0 {
			sb.WriteByte('{')
			for i, l := range s.Labels {
				if i > 0 {
					sb.WriteByte(',')
				}
				sb.WriteString(fmt.Sprintf("%s=%q", l.Name, l.Value))
			}
			sb.WriteByte('}')
		}
		sb.WriteString(fmt.Sprintf(" %g", s.Value))
		if !s.Timestamp.IsZero() {
			sb.WriteString(fmt.Sprintf(" %d", s.Timestamp.UnixMilli()))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// SimpleCounter is a basic counter metric for use by collectors.
type SimpleCounter struct {
	value uint64
}

// Inc increments the counter by 1.
func (c *SimpleCounter) Inc() {
	atomicAddUint64(&c.value, 1)
}

// Add increments the counter by n.
func (c *SimpleCounter) Add(n uint64) {
	atomicAddUint64(&c.value, n)
}

// Value returns the current counter value.
func (c *SimpleCounter) Value() uint64 {
	return atomicLoadUint64(&c.value)
}

// SimpleGauge is a basic gauge metric for use by collectors.
type SimpleGauge struct {
	value int64
}

// Set sets the gauge to v.
func (g *SimpleGauge) Set(v int64) {
	atomicStoreInt64(&g.value, v)
}

// Inc increments the gauge by 1.
func (g *SimpleGauge) Inc() {
	atomicAddInt64(&g.value, 1)
}

// Dec decrements the gauge by 1.
func (g *SimpleGauge) Dec() {
	atomicAddInt64(&g.value, -1)
}

// Add adds delta to the gauge.
func (g *SimpleGauge) Add(delta int64) {
	atomicAddInt64(&g.value, delta)
}

// Value returns the current gauge value.
func (g *SimpleGauge) Value() int64 {
	return atomicLoadInt64(&g.value)
}

// HistogramBucket represents a histogram bucket with an upper bound.
type HistogramBucket struct {
	UpperBound float64
	Count      uint64
}

// SimpleHistogram is a basic histogram metric for use by collectors.
type SimpleHistogram struct {
	mu      sync.Mutex
	buckets []HistogramBucket
	sum     float64
	count   uint64
}

// NewSimpleHistogram creates a histogram with default buckets.
func NewSimpleHistogram(upperBounds []float64) *SimpleHistogram {
	buckets := make([]HistogramBucket, len(upperBounds))
	for i, ub := range upperBounds {
		buckets[i] = HistogramBucket{UpperBound: ub}
	}
	return &SimpleHistogram{buckets: buckets}
}

// Observe records a value in the histogram.
func (h *SimpleHistogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sum += v
	h.count++
	for i := range h.buckets {
		if v <= h.buckets[i].UpperBound {
			h.buckets[i].Count++
		}
	}
}

// Snapshot returns a copy of the histogram state.
func (h *SimpleHistogram) Snapshot() (sum float64, count uint64, buckets []HistogramBucket) {
	h.mu.Lock()
	defer h.mu.Unlock()
	buckets = make([]HistogramBucket, len(h.buckets))
	copy(buckets, h.buckets)
	return h.sum, h.count, buckets
}
