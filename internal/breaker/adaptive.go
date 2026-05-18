package breaker

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// AdaptiveConfig holds configuration for the adaptive circuit breaker.
type AdaptiveConfig struct {
	// Base circuit breaker config
	Base Config

	// FailureRateEMAAlpha is the smoothing factor for failure rate EMA (0, 1].
	// Higher values weight recent observations more. Default: 0.5
	FailureRateEMAAlpha float64

	// LatencyEMAAlpha is the smoothing factor for latency EMA (0, 1]. Default: 0.3
	LatencyEMAAlpha float64

	// MinFailureRateThreshold is the lower bound for adaptive threshold (percent). Default: 20
	MinFailureRateThreshold float64

	// MaxFailureRateThreshold is the upper bound for adaptive threshold (percent). Default: 80
	MaxFailureRateThreshold float64

	// ConsecutiveFailureLimit triggers immediate open when this many consecutive
	// failures occur, regardless of the failure rate. Default: 0 (disabled)
	ConsecutiveFailureLimit uint32

	// SlowCallMultiplier adjusts SlowCallDuration dynamically.
	// SlowCallDuration = latencyEMA * SlowCallMultiplier.
	// 0 means disabled (use base config's SlowCallDuration). Default: 0
	SlowCallMultiplier float64

	// MinSlowCallDuration is the lower bound for dynamically computed SlowCallDuration.
	// Only used when SlowCallMultiplier > 0. Default: 100ms
	MinSlowCallDuration time.Duration

	// MaxSlowCallDuration is the upper bound for dynamically computed SlowCallDuration.
	// Only used when SlowCallMultiplier > 0. Default: 30s
	MaxSlowCallDuration time.Duration

	// TimeoutMultiplier adjusts the open-state timeout dynamically.
	// timeout = baseTimeout * timeoutMultiplier^(tripCount-1).
	// 0 means disabled (use base config's Timeout). Default: 0
	TimeoutMultiplier float64

	// MaxTimeout is the upper bound for dynamically computed timeout.
	// Only used when TimeoutMultiplier > 0. Default: 5 minutes
	MaxTimeout time.Duration

	// OnAdaptiveChange is called when adaptive parameters change.
	OnAdaptiveChange func(name string, params AdaptiveParams)
}

// AdaptiveParams holds the current adaptive parameters for inspection.
type AdaptiveParams struct {
	FailureRateEMA      float64
	LatencyEMA          time.Duration
	AdaptiveThreshold   float64
	ConsecutiveFailures uint32
	TripCount           uint32
	EffectiveTimeout    time.Duration
	EffectiveSlowCall   time.Duration
}

// DefaultAdaptiveConfig returns sensible defaults for the adaptive circuit breaker.
func DefaultAdaptiveConfig(name string) AdaptiveConfig {
	return AdaptiveConfig{
		Base:                    DefaultConfig(name),
		FailureRateEMAAlpha:     0.5,
		LatencyEMAAlpha:         0.3,
		MinFailureRateThreshold: 20,
		MaxFailureRateThreshold: 80,
		ConsecutiveFailureLimit: 0,
		SlowCallMultiplier:      0,
		MinSlowCallDuration:     100 * time.Millisecond,
		MaxSlowCallDuration:     30 * time.Second,
		TimeoutMultiplier:       0,
		MaxTimeout:              5 * time.Minute,
	}
}

// AdaptiveBreaker extends the circuit breaker with adaptive threshold adjustment.
type AdaptiveBreaker struct {
	config AdaptiveConfig
	inner  *CircuitBreaker
	mutex  sync.RWMutex

	// EMA state
	failureRateEMA float64
	latencyEMA     float64 // nanoseconds
	emaInitialized bool

	// Consecutive failure tracking
	consecutiveFailures uint32

	// Trip tracking (incremented via OnStateChange callback)
	tripCount atomic.Uint32

	// Adaptive threshold (moves between min and max based on history)
	currentThreshold float64

	// Window tracking for EMA updates
	windowSuccesses uint32
	windowFailures  uint32
	windowLatency   time.Duration
	windowCalls     uint32
}

// NewAdaptive creates a new adaptive circuit breaker.
func NewAdaptive(config AdaptiveConfig) *AdaptiveBreaker {
	// Apply defaults
	if config.FailureRateEMAAlpha <= 0 || config.FailureRateEMAAlpha > 1 {
		config.FailureRateEMAAlpha = 0.5
	}
	if config.LatencyEMAAlpha <= 0 || config.LatencyEMAAlpha > 1 {
		config.LatencyEMAAlpha = 0.3
	}
	if config.MinFailureRateThreshold <= 0 {
		config.MinFailureRateThreshold = 20
	}
	if config.MaxFailureRateThreshold <= 0 {
		config.MaxFailureRateThreshold = 80
	}
	if config.MinSlowCallDuration == 0 {
		config.MinSlowCallDuration = 100 * time.Millisecond
	}
	if config.MaxSlowCallDuration == 0 {
		config.MaxSlowCallDuration = 30 * time.Second
	}
	if config.MaxTimeout == 0 {
		config.MaxTimeout = 5 * time.Minute
	}

	ab := &AdaptiveBreaker{
		config:          config,
		currentThreshold: float64(config.Base.FailureRateThreshold),
	}

	// Wrap the user's callback to also track trips
	originalCallback := config.Base.OnStateChange
	config.Base.OnStateChange = func(name string, from, to State) {
		if to == StateOpen {
			ab.tripCount.Add(1)
		}
		if originalCallback != nil {
			originalCallback(name, from, to)
		}
	}

	ab.inner = New(config.Base)
	return ab
}

func (ab *AdaptiveBreaker) Name() string {
	return ab.inner.Name()
}

// State returns the current breaker state.
func (ab *AdaptiveBreaker) State() State {
	return ab.inner.State()
}

// Execute runs the function with adaptive circuit breaker protection.
func (ab *AdaptiveBreaker) Execute(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	// Check consecutive failure limit before calling inner breaker
	ab.mutex.RLock()
	consFail := ab.consecutiveFailures
	consLimit := ab.config.ConsecutiveFailureLimit
	ab.mutex.RUnlock()

	if consLimit > 0 && consFail >= consLimit {
		// Force open if consecutive failures exceeded
		ab.inner.ForceOpen()
		return nil, ErrOpenState
	}

	start := time.Now()
	result, err := ab.inner.Execute(ctx, fn)
	duration := time.Since(start)

	// Don't count rejected calls as failures for adaptive tracking
	if err == ErrOpenState || err == ErrTooManyRequests || err == ErrForcedOpen {
		return result, err
	}

	// Track consecutive failures and update window
	ab.mutex.Lock()
	if err != nil {
		ab.consecutiveFailures++
		ab.windowFailures++
	} else {
		ab.consecutiveFailures = 0
		ab.windowSuccesses++
	}
	ab.windowLatency += duration
	ab.windowCalls++
	ab.mutex.Unlock()

	return result, err
}

// UpdateEMAs recalculates exponential moving averages based on accumulated window data.
// This should be called periodically (e.g., in a goroutine with a ticker) or after a batch.
func (ab *AdaptiveBreaker) UpdateEMAs() {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()

	if ab.windowCalls == 0 {
		return
	}

	// Calculate window failure rate
	var windowFailureRate float64
	total := ab.windowSuccesses + ab.windowFailures
	if total > 0 {
		windowFailureRate = float64(ab.windowFailures) / float64(total) * 100
	}

	// Calculate window average latency
	var windowAvgLatencyNs float64
	if ab.windowCalls > 0 {
		windowAvgLatencyNs = float64(ab.windowLatency.Nanoseconds()) / float64(ab.windowCalls)
	}

	// Update EMAs
	if !ab.emaInitialized {
		ab.failureRateEMA = windowFailureRate
		ab.latencyEMA = windowAvgLatencyNs
		ab.emaInitialized = true
	} else {
		alpha := ab.config.FailureRateEMAAlpha
		ab.failureRateEMA = alpha*windowFailureRate + (1-alpha)*ab.failureRateEMA

		latAlpha := ab.config.LatencyEMAAlpha
		ab.latencyEMA = latAlpha*windowAvgLatencyNs + (1-latAlpha)*ab.latencyEMA
	}

	// Reset window counters
	ab.windowSuccesses = 0
	ab.windowFailures = 0
	ab.windowLatency = 0
	ab.windowCalls = 0

	// Adapt the failure rate threshold based on EMA trend
	ab.adaptThreshold()
}

// adaptThreshold adjusts the failure rate threshold based on observed patterns.
// Called with mutex held.
func (ab *AdaptiveBreaker) adaptThreshold() {
	min := ab.config.MinFailureRateThreshold
	max := ab.config.MaxFailureRateThreshold
	base := float64(ab.config.Base.FailureRateThreshold)

	// Strategy: when the EMA failure rate is consistently low, tighten the threshold
	// (be more sensitive). When it's high, relax it (be more tolerant).
	if ab.failureRateEMA < min {
		// Low failure environment: tighten threshold
		ab.currentThreshold = min + (base-min)*0.3
	} else if ab.failureRateEMA > max {
		// High failure environment: relax threshold
		ab.currentThreshold = base + (max-base)*0.5
	} else {
		// Mid-range: interpolate
		ratio := (ab.failureRateEMA - min) / (max - min)
		ab.currentThreshold = base + (max-base)*ratio*0.3
	}

	// Clamp
	if ab.currentThreshold < min {
		ab.currentThreshold = min
	}
	if ab.currentThreshold > max {
		ab.currentThreshold = max
	}
}

// GetAdaptiveParams returns the current adaptive parameters.
func (ab *AdaptiveBreaker) GetAdaptiveParams() AdaptiveParams {
	ab.mutex.RLock()
	defer ab.mutex.RUnlock()

	params := AdaptiveParams{
		FailureRateEMA:      ab.failureRateEMA,
		LatencyEMA:          time.Duration(int64(ab.latencyEMA)),
		AdaptiveThreshold:   ab.currentThreshold,
		ConsecutiveFailures: ab.consecutiveFailures,
		TripCount:           ab.tripCount.Load(),
	}

	// Compute effective timeout
	trips := ab.tripCount.Load()
	if ab.config.TimeoutMultiplier > 0 && trips > 0 {
		multiplier := math.Pow(ab.config.TimeoutMultiplier, float64(trips-1))
		timeout := time.Duration(float64(ab.config.Base.Timeout) * multiplier)
		if timeout > ab.config.MaxTimeout {
			timeout = ab.config.MaxTimeout
		}
		params.EffectiveTimeout = timeout
	} else {
		params.EffectiveTimeout = ab.config.Base.Timeout
	}

	// Compute effective slow call duration
	if ab.config.SlowCallMultiplier > 0 {
		slowCall := time.Duration(ab.latencyEMA * ab.config.SlowCallMultiplier)
		if slowCall < ab.config.MinSlowCallDuration {
			slowCall = ab.config.MinSlowCallDuration
		}
		if slowCall > ab.config.MaxSlowCallDuration {
			slowCall = ab.config.MaxSlowCallDuration
		}
		params.EffectiveSlowCall = slowCall
	}

	return params
}

// Reset resets the adaptive breaker to its initial state.
func (ab *AdaptiveBreaker) Reset() {
	ab.mutex.Lock()
	ab.consecutiveFailures = 0
	ab.failureRateEMA = 0
	ab.latencyEMA = 0
	ab.emaInitialized = false
	ab.windowSuccesses = 0
	ab.windowFailures = 0
	ab.windowLatency = 0
	ab.windowCalls = 0
	ab.currentThreshold = float64(ab.config.Base.FailureRateThreshold)
	ab.mutex.Unlock()

	ab.tripCount.Store(0)
	ab.inner.Reset()
}

// GetMetrics returns the underlying breaker's metrics.
func (ab *AdaptiveBreaker) GetMetrics() Metrics {
	return ab.inner.GetMetrics()
}

// ForceOpen forces the breaker into the open state.
func (ab *AdaptiveBreaker) ForceOpen() {
	ab.inner.ForceOpen()
}

// TripCount returns the number of times the breaker has tripped to open.
func (ab *AdaptiveBreaker) TripCount() uint32 {
	return ab.tripCount.Load()
}
