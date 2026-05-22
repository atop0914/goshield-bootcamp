package metrics

import (
	"fmt"
	"net/http"
	"time"
)

// HTTPHandler returns an http.Handler that serves metrics in Prometheus text format.
func HTTPHandler(registry *Registry) http.Handler {
	return &metricsHandler{registry: registry}
}

type metricsHandler struct {
	registry *Registry
}

func (h *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body := h.registry.PrometheusTextFormat()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if body == "" {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

// CollectorFunc is an adapter to allow the use of ordinary functions as Collectors.
type CollectorFunc func() []MetricSample

// Collect calls the function.
func (f CollectorFunc) Collect() []MetricSample {
	return f()
}

// MultiCollector combines multiple collectors into one.
type MultiCollector struct {
	collectors []Collector
}

// NewMultiCollector creates a collector that delegates to multiple sub-collectors.
func NewMultiCollector(collectors ...Collector) *MultiCollector {
	return &MultiCollector{collectors: collectors}
}

// Collect returns metrics from all sub-collectors.
func (mc *MultiCollector) Collect() []MetricSample {
	var all []MetricSample
	for _, c := range mc.collectors {
		all = append(all, c.Collect()...)
	}
	return all
}

// Snapshot represents a point-in-time view of all metrics.
type Snapshot struct {
	Timestamp time.Time
	Metrics   []MetricSample
}

// TakeSnapshot gathers all metrics and returns a Snapshot.
func (r *Registry) TakeSnapshot() Snapshot {
	return Snapshot{
		Timestamp: time.Now(),
		Metrics:   r.Gather(),
	}
}
