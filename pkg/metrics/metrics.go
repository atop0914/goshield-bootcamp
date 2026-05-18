// Package metrics provides a zero-dependency metrics collection and
// Prometheus-compatible exposition system for GoShield resilience patterns.
//
// This is the public API. For internal details, see internal/metrics.
//
// Example:
//
//	registry := metrics.NewRegistry()
//
//	// Register a circuit breaker for metrics collection
//	registry.RegisterBreaker(myBreaker)
//
//	// Serve metrics via HTTP
//	http.Handle("/metrics", metrics.Handler(registry))
package metrics

import (
	"net/http"

	intmetrics "github.com/atop0914/goshield/internal/metrics"
)

// Collector is the interface for collecting metrics.
type Collector = intmetrics.Collector

// Registry collects metrics from multiple collectors.
type Registry struct {
	internal *intmetrics.Registry
}

// NewRegistry creates a new metrics registry with the default "goshield" namespace.
func NewRegistry() *Registry {
	return &Registry{internal: intmetrics.NewRegistry()}
}

// NewRegistryWithNamespace creates a registry with a custom namespace prefix.
func NewRegistryWithNamespace(ns string) *Registry {
	return &Registry{internal: intmetrics.NewRegistryWithNamespace(ns)}
}

// Register adds a collector to the registry.
func (r *Registry) Register(c Collector) {
	r.internal.Register(c)
}

// MustRegister adds a collector and panics on nil.
func (r *Registry) MustRegister(c Collector) {
	r.internal.MustRegister(c)
}

// Gather collects all metrics from registered collectors.
func (r *Registry) Gather() []intmetrics.MetricSample {
	return r.internal.Gather()
}

// PrometheusText returns metrics in Prometheus text exposition format.
func (r *Registry) PrometheusText() string {
	return r.internal.PrometheusTextFormat()
}

// Handler returns an http.Handler that serves metrics in Prometheus text format.
func Handler(registry *Registry) http.Handler {
	return intmetrics.HTTPHandler(registry.internal)
}

// Label is a key-value pair for metric labels.
type Label = intmetrics.Label

// Labels creates a sorted slice of labels from key-value pairs.
func Labels(pairs ...string) []Label {
	return intmetrics.Labels(pairs...)
}

// RegisterBreaker registers a circuit breaker for metrics collection.
// It uses the breaker's GetMetrics() and State() methods.
func (r *Registry) RegisterBreaker(name string, stateGetter func() int, metricsGetter func() (float64, float64, uint32, uint64, uint64, uint64, uint64, uint64)) {
	r.internal.Register(&intmetrics.BreakerCollector{
		Name:       name,
		GetState:   stateGetter,
		GetMetrics: metricsGetter,
	})
}

// RegisterRateLimiter registers a rate limiter for metrics collection.
func (r *Registry) RegisterRateLimiter(name string, rateGetter func() float64, burstGetter func() int) {
	r.internal.Register(&intmetrics.RateLimiterCollector{
		Name:     name,
		GetRate:  rateGetter,
		GetBurst: burstGetter,
	})
}

// RegisterBulkhead registers a bulkhead for metrics collection.
func (r *Registry) RegisterBulkhead(name string, metricsGetter func() (int64, int64, int64, int64, int64)) {
	r.internal.Register(&intmetrics.BulkheadCollector{
		Name:       name,
		GetMetrics: metricsGetter,
	})
}

// RegisterRetry registers a retry tracker for metrics collection.
func (r *Registry) RegisterRetry(name string, metricsGetter func() (uint64, uint64, uint64, uint64)) {
	r.internal.Register(&intmetrics.RetryCollector{
		Name:       name,
		GetMetrics: metricsGetter,
	})
}

// RegisterTimeout registers a timeout tracker for metrics collection.
func (r *Registry) RegisterTimeout(name string, metricsGetter func() (uint64, uint64, uint64)) {
	r.internal.Register(&intmetrics.TimeoutCollector{
		Name:       name,
		GetMetrics: metricsGetter,
	})
}
