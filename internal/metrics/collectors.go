// Package metrics - collectors.go provides adapter types that make
// resilience patterns implement the Collector interface.
//
// These adapters avoid circular imports by living in the metrics package
// and referencing the internal pattern packages.
package metrics

import (
	"fmt"
	"time"
)

// BreakerCollector adapts a circuit breaker to the Collector interface.
// It uses a function-based approach to avoid circular imports.
type BreakerCollector struct {
	Name string
	// State returns the current state as an int.
	GetState func() int
	// GetMetrics returns failure rate, slow call rate, total calls, successes, failures, rejected, slow calls, state transitions.
	GetMetrics func() (failureRate, slowCallRate float64, totalCalls uint32, totalSuccesses, totalFailures, totalRejected, totalSlowCalls, stateTransitions uint64)
}

// Collect implements Collector for circuit breaker metrics.
func (bc *BreakerCollector) Collect() []MetricSample {
	samples := make([]MetricSample, 0, 8)
	ts := time.Now()

	failureRate, slowCallRate, totalCalls, totalSuccesses, totalFailures, totalRejected, totalSlowCalls, stateTransitions := bc.GetMetrics()

	state := bc.GetState()

	nameLabel := Label{Name: "name", Value: bc.Name}

	samples = append(samples, MetricSample{
		Name:      "breaker_state",
		Labels:    []Label{nameLabel, {Name: "state", Value: stateString(state)}},
		Value:     1,
		Type:      Gauge,
		Help:      "Current state of the circuit breaker (1 = active state).",
		Timestamp: ts,
	})

	samples = append(samples, MetricSample{
		Name:      "breaker_failure_rate",
		Labels:    []Label{nameLabel},
		Value:     failureRate,
		Type:      Gauge,
		Help:      "Current failure rate percentage in the sliding window.",
		Timestamp: ts,
	})

	samples = append(samples, MetricSample{
		Name:      "breaker_slow_call_rate",
		Labels:    []Label{nameLabel},
		Value:     slowCallRate,
		Type:      Gauge,
		Help:      "Current slow call rate percentage in the sliding window.",
		Timestamp: ts,
	})

	samples = append(samples, MetricSample{
		Name:      "breaker_calls_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalCalls),
		Type:      Gauge,
		Help:      "Current window call count.",
		Timestamp: ts,
	})

	samples = append(samples, MetricSample{
		Name:      "breaker_successes_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalSuccesses),
		Type:      Counter,
		Help:      "Total successful calls since creation.",
		Timestamp: ts,
	})

	samples = append(samples, MetricSample{
		Name:      "breaker_failures_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalFailures),
		Type:      Counter,
		Help:      "Total failed calls since creation.",
		Timestamp: ts,
	})

	samples = append(samples, MetricSample{
		Name:      "breaker_rejected_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalRejected),
		Type:      Counter,
		Help:      "Total rejected calls (open/half-open overflow).",
		Timestamp: ts,
	})

	samples = append(samples, MetricSample{
		Name:      "breaker_slow_calls_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalSlowCalls),
		Type:      Counter,
		Help:      "Total slow calls since creation.",
		Timestamp: ts,
	})

	samples = append(samples, MetricSample{
		Name:      "breaker_state_transitions_total",
		Labels:    []Label{nameLabel},
		Value:     float64(stateTransitions),
		Type:      Counter,
		Help:      "Total state transitions.",
		Timestamp: ts,
	})

	return samples
}

func stateString(state int) string {
	switch state {
	case 0:
		return "closed"
	case 1:
		return "open"
	case 2:
		return "half_open"
	case 3:
		return "disabled"
	case 4:
		return "forced_open"
	default:
		return fmt.Sprintf("unknown_%d", state)
	}
}

// RateLimiterCollector adapts a rate limiter to the Collector interface.
type RateLimiterCollector struct {
	Name string
	// Rate returns the configured rate (tokens/sec or requests/window).
	GetRate func() float64
	// Burst returns the configured burst size.
	GetBurst func() int
}

// Collect implements Collector for rate limiter metrics.
func (rc *RateLimiterCollector) Collect() []MetricSample {
	ts := time.Now()
	nameLabel := Label{Name: "name", Value: rc.Name}

	return []MetricSample{
		{
			Name:      "ratelimit_rate",
			Labels:    []Label{nameLabel},
			Value:     rc.GetRate(),
			Type:      Gauge,
			Help:      "Configured rate (tokens per second or requests per window).",
			Timestamp: ts,
		},
		{
			Name:      "ratelimit_burst",
			Labels:    []Label{nameLabel},
			Value:     float64(rc.GetBurst()),
			Type:      Gauge,
			Help:      "Configured burst size.",
			Timestamp: ts,
		},
	}
}

// BulkheadCollector adapts a bulkhead to the Collector interface.
type BulkheadCollector struct {
	Name string
	// GetMetrics returns available slots, max concurrent, total executions, total rejections, current running.
	GetMetrics func() (available, maxConcurrent, totalExecutions, totalRejections, currentRunning int64)
}

// Collect implements Collector for bulkhead metrics.
func (bc *BulkheadCollector) Collect() []MetricSample {
	ts := time.Now()
	nameLabel := Label{Name: "name", Value: bc.Name}

	available, maxConcurrent, totalExecutions, totalRejections, currentRunning := bc.GetMetrics()

	return []MetricSample{
		{
			Name:      "bulkhead_available_permits",
			Labels:    []Label{nameLabel},
			Value:     float64(available),
			Type:      Gauge,
			Help:      "Number of available concurrent execution slots.",
			Timestamp: ts,
		},
		{
			Name:      "bulkhead_max_concurrent",
			Labels:    []Label{nameLabel},
			Value:     float64(maxConcurrent),
			Type:      Gauge,
			Help:      "Maximum concurrent executions allowed.",
			Timestamp: ts,
		},
		{
			Name:      "bulkhead_running",
			Labels:    []Label{nameLabel},
			Value:     float64(currentRunning),
			Type:      Gauge,
			Help:      "Currently running executions.",
			Timestamp: ts,
		},
		{
			Name:      "bulkhead_executions_total",
			Labels:    []Label{nameLabel},
			Value:     float64(totalExecutions),
			Type:      Counter,
			Help:      "Total successful executions.",
			Timestamp: ts,
		},
		{
			Name:      "bulkhead_rejections_total",
			Labels:    []Label{nameLabel},
			Value:     float64(totalRejections),
			Type:      Counter,
			Help:      "Total rejected executions (at capacity).",
			Timestamp: ts,
		},
	}
}

// RetryCollector tracks retry attempts.
type RetryCollector struct {
	Name string
	// GetMetrics returns total attempts, total successes, total retries, current attempt.
	GetMetrics func() (totalAttempts, totalSuccesses, totalRetries, currentAttempt uint64)
}

// Collect implements Collector for retry metrics.
func (rc *RetryCollector) Collect() []MetricSample {
	ts := time.Now()
	nameLabel := Label{Name: "name", Value: rc.Name}

	totalAttempts, totalSuccesses, totalRetries, _ := rc.GetMetrics()

	return []MetricSample{
		{
			Name:      "retry_attempts_total",
			Labels:    []Label{nameLabel},
			Value:     float64(totalAttempts),
			Type:      Counter,
			Help:      "Total retry attempts (including first try).",
			Timestamp: ts,
		},
		{
			Name:      "retry_successes_total",
			Labels:    []Label{nameLabel},
			Value:     float64(totalSuccesses),
			Type:      Counter,
			Help:      "Total successful calls (after retries).",
			Timestamp: ts,
		},
		{
			Name:      "retry_retries_total",
			Labels:    []Label{nameLabel},
			Value:     float64(totalRetries),
			Type:      Counter,
			Help:      "Total retry events (not counting first attempt).",
			Timestamp: ts,
		},
	}
}

// TimeoutCollector tracks timeout events.
type TimeoutCollector struct {
	Name string
	// GetMetrics returns total calls, total timeouts, total successes.
	GetMetrics func() (totalCalls, totalTimeouts, totalSuccesses uint64)
}

// Collect implements Collector for timeout metrics.
func (tc *TimeoutCollector) Collect() []MetricSample {
	ts := time.Now()
	nameLabel := Label{Name: "name", Value: tc.Name}

	totalCalls, totalTimeouts, totalSuccesses := tc.GetMetrics()

	return []MetricSample{
		{
			Name:      "timeout_calls_total",
			Labels:    []Label{nameLabel},
			Value:     float64(totalCalls),
			Type:      Counter,
			Help:      "Total calls through timeout wrapper.",
			Timestamp: ts,
		},
		{
			Name:      "timeout_timeouts_total",
			Labels:    []Label{nameLabel},
			Value:     float64(totalTimeouts),
			Type:      Counter,
			Help:      "Total calls that timed out.",
			Timestamp: ts,
		},
		{
			Name:      "timeout_successes_total",
			Labels:    []Label{nameLabel},
			Value:     float64(totalSuccesses),
			Type:      Counter,
			Help:      "Total calls that completed before timeout.",
			Timestamp: ts,
		},
	}
}

// AdaptiveBreakerCollector adapts an adaptive circuit breaker to the Collector interface.
// It exposes both standard breaker metrics and adaptive-specific metrics.
type AdaptiveBreakerCollector struct {
	Name string
	// GetState returns the current state as an int.
	GetState func() int
	// GetMetrics returns failure rate, slow call rate, total calls, successes, failures, rejected, slow calls, state transitions.
	GetMetrics func() (failureRate, slowCallRate float64, totalCalls uint32, totalSuccesses, totalFailures, totalRejected, totalSlowCalls, stateTransitions uint64)
	// GetAdaptiveParams returns failure rate EMA, latency EMA (ns), adaptive threshold, consecutive failures, trip count.
	GetAdaptiveParams func() (failureRateEMA, adaptiveThreshold float64, latencyEMA int64, consecutiveFailures, tripCount uint32)
}

// Collect implements Collector for adaptive breaker metrics.
func (ac *AdaptiveBreakerCollector) Collect() []MetricSample {
	samples := make([]MetricSample, 0, 14)
	ts := time.Now()
	nameLabel := Label{Name: "name", Value: ac.Name}

	failureRate, slowCallRate, totalCalls, totalSuccesses, totalFailures, totalRejected, totalSlowCalls, stateTransitions := ac.GetMetrics()
	state := ac.GetState()

	// Standard breaker metrics
	samples = append(samples, MetricSample{
		Name:      "breaker_state",
		Labels:    []Label{nameLabel, {Name: "state", Value: stateString(state)}},
		Value:     1,
		Type:      Gauge,
		Help:      "Current state of the circuit breaker (1 = active state).",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_failure_rate",
		Labels:    []Label{nameLabel},
		Value:     failureRate,
		Type:      Gauge,
		Help:      "Current failure rate percentage in the sliding window.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_slow_call_rate",
		Labels:    []Label{nameLabel},
		Value:     slowCallRate,
		Type:      Gauge,
		Help:      "Current slow call rate percentage in the sliding window.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_calls_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalCalls),
		Type:      Gauge,
		Help:      "Current window call count.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_successes_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalSuccesses),
		Type:      Counter,
		Help:      "Total successful calls since creation.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_failures_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalFailures),
		Type:      Counter,
		Help:      "Total failed calls since creation.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_rejected_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalRejected),
		Type:      Counter,
		Help:      "Total rejected calls (open/half-open overflow).",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_slow_calls_total",
		Labels:    []Label{nameLabel},
		Value:     float64(totalSlowCalls),
		Type:      Counter,
		Help:      "Total slow calls since creation.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_state_transitions_total",
		Labels:    []Label{nameLabel},
		Value:     float64(stateTransitions),
		Type:      Counter,
		Help:      "Total state transitions.",
		Timestamp: ts,
	})

	// Adaptive-specific metrics
	failureRateEMA, adaptiveThreshold, latencyEMA, consecutiveFailures, tripCount := ac.GetAdaptiveParams()

	samples = append(samples, MetricSample{
		Name:      "breaker_adaptive_failure_rate_ema",
		Labels:    []Label{nameLabel},
		Value:     failureRateEMA,
		Type:      Gauge,
		Help:      "Exponential moving average of failure rate.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_adaptive_threshold",
		Labels:    []Label{nameLabel},
		Value:     adaptiveThreshold,
		Type:      Gauge,
		Help:      "Current adaptive failure rate threshold.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_adaptive_latency_ema_ns",
		Labels:    []Label{nameLabel},
		Value:     float64(latencyEMA),
		Type:      Gauge,
		Help:      "Exponential moving average of call latency in nanoseconds.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_adaptive_consecutive_failures",
		Labels:    []Label{nameLabel},
		Value:     float64(consecutiveFailures),
		Type:      Gauge,
		Help:      "Current consecutive failure count.",
		Timestamp: ts,
	})
	samples = append(samples, MetricSample{
		Name:      "breaker_adaptive_trip_count",
		Labels:    []Label{nameLabel},
		Value:     float64(tripCount),
		Type:      Counter,
		Help:      "Total number of times the adaptive breaker has tripped.",
		Timestamp: ts,
	})

	return samples
}
