// Package config provides configuration management for GoShield.
//
// It supports loading configuration from JSON files, environment variable
// overrides, validation, and hot-reload via file watching.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the top-level GoShield configuration.
type Config struct {
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
	Adaptive       *AdaptiveConfig       `json:"adaptive,omitempty"`
	Retry          *RetryConfig          `json:"retry,omitempty"`
	RateLimiter    *RateLimiterConfig    `json:"rate_limiter,omitempty"`
	Timeout        *TimeoutConfig        `json:"timeout,omitempty"`
	Bulkhead       *BulkheadConfig       `json:"bulkhead,omitempty"`
	Metrics        *MetricsConfig        `json:"metrics,omitempty"`
	HTTP           *HTTPConfig           `json:"http,omitempty"`
}

// CircuitBreakerConfig configures the standard circuit breaker.
type CircuitBreakerConfig struct {
	Name                  string `json:"name"`
	MaxRequests           uint32 `json:"max_requests"`
	TimeoutSeconds        int    `json:"timeout_seconds"`
	FailureRateThreshold  uint8  `json:"failure_rate_threshold"`
	SlowCallRateThreshold uint8  `json:"slow_call_rate_threshold"`
	SlowCallDurationMs    int    `json:"slow_call_duration_ms"`
	MinimumNumberOfCalls  uint32 `json:"minimum_number_of_calls"`
	SlidingWindowSize     uint32 `json:"sliding_window_size"`
	SlidingWindowType     string `json:"sliding_window_type"` // "count" or "time"
}

// AdaptiveConfig configures the adaptive circuit breaker.
type AdaptiveConfig struct {
	Name                    string  `json:"name"`
	FailureRateEMAAlpha     float64 `json:"failure_rate_ema_alpha"`
	LatencyEMAAlpha         float64 `json:"latency_ema_alpha"`
	MinAdaptiveThreshold    float64 `json:"min_adaptive_threshold"`
	MaxAdaptiveThreshold    float64 `json:"max_adaptive_threshold"`
	ConsecutiveFailureLimit int     `json:"consecutive_failure_limit"`
	SlowCallMultiplier      float64 `json:"slow_call_multiplier"`
	MinSlowCallDurationMs   int     `json:"min_slow_call_duration_ms"`
	MaxSlowCallDurationMs   int     `json:"max_slow_call_duration_ms"`
	TimeoutMultiplier       float64 `json:"timeout_multiplier"`
	MaxTimeoutSeconds       int     `json:"max_timeout_seconds"`
	TimeoutSeconds          int     `json:"timeout_seconds"`
	FailureRateThreshold    uint8   `json:"failure_rate_threshold"`
	SlowCallRateThreshold   uint8   `json:"slow_call_rate_threshold"`
	MinimumNumberOfCalls    uint32  `json:"minimum_number_of_calls"`
	MaxRequests             uint32  `json:"max_requests"`
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries    int     `json:"max_retries"`
	Strategy      string  `json:"strategy"` // "fixed", "exponential", "exponential_random", "fibonacci"
	IntervalMs    int     `json:"interval_ms"`
	MaxIntervalMs int     `json:"max_interval_ms"`
	Multiplier    float64 `json:"multiplier"`
	MaxDurationMs int     `json:"max_duration_ms"`
}

// RateLimiterConfig configures rate limiting.
type RateLimiterConfig struct {
	Type  string  `json:"type"` // "token_bucket" or "sliding_window"
	Rate  float64 `json:"rate"` // requests per second
	Burst int     `json:"burst"`
}

// TimeoutConfig configures timeout behavior.
type TimeoutConfig struct {
	DurationMs int `json:"duration_ms"`
}

// BulkheadConfig configures bulkhead behavior.
type BulkheadConfig struct {
	MaxConcurrent     int `json:"max_concurrent"`
	MaxWaitDurationMs int `json:"max_wait_duration_ms"`
}

// MetricsConfig configures the metrics system.
type MetricsConfig struct {
	Enabled   bool   `json:"enabled"`
	Namespace string `json:"namespace"`
	Endpoint  string `json:"endpoint"` // HTTP endpoint path, e.g. "/metrics"
}

// HTTPConfig configures HTTP middleware defaults.
type HTTPConfig struct {
	Enabled     bool   `json:"enabled"`
	Addr        string `json:"addr"`         // listen address, e.g. ":8080"
	MetricsAddr string `json:"metrics_addr"` // metrics listen address
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		CircuitBreaker: &CircuitBreakerConfig{
			Name:                  "default",
			MaxRequests:           1,
			TimeoutSeconds:        60,
			FailureRateThreshold:  50,
			SlowCallRateThreshold: 100,
			SlowCallDurationMs:    0,
			MinimumNumberOfCalls:  10,
			SlidingWindowSize:     100,
			SlidingWindowType:     "count",
		},
		Adaptive: &AdaptiveConfig{
			Name:                    "adaptive",
			FailureRateEMAAlpha:     0.5,
			LatencyEMAAlpha:         0.3,
			MinAdaptiveThreshold:    20.0,
			MaxAdaptiveThreshold:    80.0,
			ConsecutiveFailureLimit: 5,
			SlowCallMultiplier:      3.0,
			MinSlowCallDurationMs:   100,
			MaxSlowCallDurationMs:   5000,
			TimeoutMultiplier:       2.0,
			MaxTimeoutSeconds:       300,
			TimeoutSeconds:          60,
			FailureRateThreshold:    50,
			SlowCallRateThreshold:   100,
			MinimumNumberOfCalls:    10,
			MaxRequests:             1,
		},
		Retry: &RetryConfig{
			MaxRetries:    3,
			Strategy:      "exponential",
			IntervalMs:    1000,
			MaxIntervalMs: 30000,
			Multiplier:    2.0,
			MaxDurationMs: 60000,
		},
		RateLimiter: &RateLimiterConfig{
			Type:  "token_bucket",
			Rate:  100,
			Burst: 200,
		},
		Timeout: &TimeoutConfig{
			DurationMs: 5000,
		},
		Bulkhead: &BulkheadConfig{
			MaxConcurrent:     10,
			MaxWaitDurationMs: 5000,
		},
		Metrics: &MetricsConfig{
			Enabled:   true,
			Namespace: "goshield",
			Endpoint:  "/metrics",
		},
		HTTP: &HTTPConfig{
			Enabled:     true,
			Addr:        ":8080",
			MetricsAddr: ":9090",
		},
	}
}

// LoadFile reads a JSON configuration file and returns a Config.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file %s: %w", path, err)
	}
	return LoadBytes(data)
}

// LoadBytes parses JSON data and returns a Config.
func LoadBytes(data []byte) (*Config, error) {
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parse json: %w", err)
	}
	return cfg, nil
}

// ApplyEnvOverrides applies environment variable overrides to the config.
// Environment variables follow the pattern GOSHIELD_<SECTION>_<KEY>.
// Examples:
//
//	GOSHIELD_CIRCUIT_BREAKER_TIMEOUT_SECONDS=30
//	GOSHIELD_RATE_LIMITER_RATE=500
//	GOSHIELD_TIMEOUT_DURATION_MS=10000
//	GOSHIELD_BULKHEAD_MAX_CONCURRENT=50
//	GOSHIELD_METRICS_ENABLED=false
//	GOSHIELD_HTTP_ADDR=:9090
func (c *Config) ApplyEnvOverrides() {
	prefix := "GOSHIELD_"

	if c.CircuitBreaker != nil {
		applyEnvInt(prefix+"CIRCUIT_BREAKER_TIMEOUT_SECONDS", &c.CircuitBreaker.TimeoutSeconds)
		applyEnvUint8(prefix+"CIRCUIT_BREAKER_FAILURE_RATE_THRESHOLD", &c.CircuitBreaker.FailureRateThreshold)
		applyEnvUint8(prefix+"CIRCUIT_BREAKER_SLOW_CALL_RATE_THRESHOLD", &c.CircuitBreaker.SlowCallRateThreshold)
		applyEnvUint32(prefix+"CIRCUIT_BREAKER_MINIMUM_NUMBER_OF_CALLS", &c.CircuitBreaker.MinimumNumberOfCalls)
		applyEnvUint32(prefix+"CIRCUIT_BREAKER_SLIDING_WINDOW_SIZE", &c.CircuitBreaker.SlidingWindowSize)
		applyEnvUint32(prefix+"CIRCUIT_BREAKER_MAX_REQUESTS", &c.CircuitBreaker.MaxRequests)
	}

	if c.Adaptive != nil {
		applyEnvFloat64(prefix+"ADAPTIVE_FAILURE_RATE_EMA_ALPHA", &c.Adaptive.FailureRateEMAAlpha)
		applyEnvFloat64(prefix+"ADAPTIVE_LATENCY_EMA_ALPHA", &c.Adaptive.LatencyEMAAlpha)
		applyEnvFloat64(prefix+"ADAPTIVE_MIN_ADAPTIVE_THRESHOLD", &c.Adaptive.MinAdaptiveThreshold)
		applyEnvFloat64(prefix+"ADAPTIVE_MAX_ADAPTIVE_THRESHOLD", &c.Adaptive.MaxAdaptiveThreshold)
		applyEnvInt(prefix+"ADAPTIVE_CONSECUTIVE_FAILURE_LIMIT", &c.Adaptive.ConsecutiveFailureLimit)
		applyEnvFloat64(prefix+"ADAPTIVE_SLOW_CALL_MULTIPLIER", &c.Adaptive.SlowCallMultiplier)
		applyEnvFloat64(prefix+"ADAPTIVE_TIMEOUT_MULTIPLIER", &c.Adaptive.TimeoutMultiplier)
		applyEnvInt(prefix+"ADAPTIVE_MAX_TIMEOUT_SECONDS", &c.Adaptive.MaxTimeoutSeconds)
	}

	if c.Retry != nil {
		applyEnvInt(prefix+"RETRY_MAX_RETRIES", &c.Retry.MaxRetries)
		applyEnvInt(prefix+"RETRY_INTERVAL_MS", &c.Retry.IntervalMs)
		applyEnvInt(prefix+"RETRY_MAX_INTERVAL_MS", &c.Retry.MaxIntervalMs)
		applyEnvFloat64(prefix+"RETRY_MULTIPLIER", &c.Retry.Multiplier)
		applyEnvInt(prefix+"RETRY_MAX_DURATION_MS", &c.Retry.MaxDurationMs)
	}

	if c.RateLimiter != nil {
		applyEnvFloat64(prefix+"RATE_LIMITER_RATE", &c.RateLimiter.Rate)
		applyEnvInt(prefix+"RATE_LIMITER_BURST", &c.RateLimiter.Burst)
	}

	if c.Timeout != nil {
		applyEnvInt(prefix+"TIMEOUT_DURATION_MS", &c.Timeout.DurationMs)
	}

	if c.Bulkhead != nil {
		applyEnvInt(prefix+"BULKHEAD_MAX_CONCURRENT", &c.Bulkhead.MaxConcurrent)
		applyEnvInt(prefix+"BULKHEAD_MAX_WAIT_DURATION_MS", &c.Bulkhead.MaxWaitDurationMs)
	}

	if c.Metrics != nil {
		applyEnvBool(prefix+"METRICS_ENABLED", &c.Metrics.Enabled)
		applyEnvString(prefix+"METRICS_NAMESPACE", &c.Metrics.Namespace)
		applyEnvString(prefix+"METRICS_ENDPOINT", &c.Metrics.Endpoint)
	}

	if c.HTTP != nil {
		applyEnvBool(prefix+"HTTP_ENABLED", &c.HTTP.Enabled)
		applyEnvString(prefix+"HTTP_ADDR", &c.HTTP.Addr)
		applyEnvString(prefix+"HTTP_METRICS_ADDR", &c.HTTP.MetricsAddr)
	}
}

// Validate checks the configuration for correctness and returns an error if invalid.
func (c *Config) Validate() error {
	var errs []string

	if c.CircuitBreaker != nil {
		cb := c.CircuitBreaker
		if cb.FailureRateThreshold > 100 {
			errs = append(errs, "circuit_breaker.failure_rate_threshold must be <= 100")
		}
		if cb.SlowCallRateThreshold > 100 {
			errs = append(errs, "circuit_breaker.slow_call_rate_threshold must be <= 100")
		}
		if cb.SlidingWindowSize == 0 {
			errs = append(errs, "circuit_breaker.sliding_window_size must be > 0")
		}
		if cb.TimeoutSeconds <= 0 {
			errs = append(errs, "circuit_breaker.timeout_seconds must be > 0")
		}
		if cb.SlidingWindowType != "count" && cb.SlidingWindowType != "time" {
			errs = append(errs, "circuit_breaker.sliding_window_type must be 'count' or 'time'")
		}
	}

	if c.Adaptive != nil {
		a := c.Adaptive
		if a.FailureRateEMAAlpha < 0 || a.FailureRateEMAAlpha > 1 {
			errs = append(errs, "adaptive.failure_rate_ema_alpha must be between 0 and 1")
		}
		if a.LatencyEMAAlpha < 0 || a.LatencyEMAAlpha > 1 {
			errs = append(errs, "adaptive.latency_ema_alpha must be between 0 and 1")
		}
		if a.MinAdaptiveThreshold < 0 || a.MinAdaptiveThreshold > 100 {
			errs = append(errs, "adaptive.min_adaptive_threshold must be between 0 and 100")
		}
		if a.MaxAdaptiveThreshold < 0 || a.MaxAdaptiveThreshold > 100 {
			errs = append(errs, "adaptive.max_adaptive_threshold must be between 0 and 100")
		}
		if a.MinAdaptiveThreshold > a.MaxAdaptiveThreshold {
			errs = append(errs, "adaptive.min_adaptive_threshold must be <= max_adaptive_threshold")
		}
		if a.ConsecutiveFailureLimit < 0 {
			errs = append(errs, "adaptive.consecutive_failure_limit must be >= 0")
		}
		if a.TimeoutMultiplier < 1 {
			errs = append(errs, "adaptive.timeout_multiplier must be >= 1")
		}
		if a.MaxTimeoutSeconds <= 0 {
			errs = append(errs, "adaptive.max_timeout_seconds must be > 0")
		}
	}

	if c.Retry != nil {
		r := c.Retry
		if r.MaxRetries < 0 {
			errs = append(errs, "retry.max_retries must be >= 0")
		}
		if r.IntervalMs <= 0 {
			errs = append(errs, "retry.interval_ms must be > 0")
		}
		if r.Strategy != "fixed" && r.Strategy != "exponential" && r.Strategy != "exponential_random" && r.Strategy != "fibonacci" {
			errs = append(errs, "retry.strategy must be one of: fixed, exponential, exponential_random, fibonacci")
		}
		if r.Multiplier < 1 {
			errs = append(errs, "retry.multiplier must be >= 1")
		}
	}

	if c.RateLimiter != nil {
		rl := c.RateLimiter
		if rl.Rate <= 0 {
			errs = append(errs, "rate_limiter.rate must be > 0")
		}
		if rl.Burst <= 0 {
			errs = append(errs, "rate_limiter.burst must be > 0")
		}
		if rl.Type != "token_bucket" && rl.Type != "sliding_window" {
			errs = append(errs, "rate_limiter.type must be 'token_bucket' or 'sliding_window'")
		}
	}

	if c.Timeout != nil {
		if c.Timeout.DurationMs <= 0 {
			errs = append(errs, "timeout.duration_ms must be > 0")
		}
	}

	if c.Bulkhead != nil {
		if c.Bulkhead.MaxConcurrent <= 0 {
			errs = append(errs, "bulkhead.max_concurrent must be > 0")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// DeepCopy returns a deep copy of the config.
func (c *Config) DeepCopy() *Config {
	data, _ := json.Marshal(c)
	cp := &Config{}
	_ = json.Unmarshal(data, cp)
	return cp
}

// ToJSON serializes the config to indented JSON.
func (c *Config) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// PresetConservative returns a config with conservative settings (more sensitive to failures).
func PresetConservative() *Config {
	cfg := DefaultConfig()
	cfg.CircuitBreaker.FailureRateThreshold = 30
	cfg.CircuitBreaker.SlowCallRateThreshold = 50
	cfg.CircuitBreaker.MinimumNumberOfCalls = 5
	cfg.CircuitBreaker.TimeoutSeconds = 30
	cfg.Retry.MaxRetries = 5
	cfg.Retry.IntervalMs = 500
	cfg.RateLimiter.Rate = 50
	cfg.RateLimiter.Burst = 100
	cfg.Timeout.DurationMs = 3000
	cfg.Bulkhead.MaxConcurrent = 5
	return cfg
}

// PresetAggressive returns a config with aggressive settings (higher throughput, less sensitive).
func PresetAggressive() *Config {
	cfg := DefaultConfig()
	cfg.CircuitBreaker.FailureRateThreshold = 80
	cfg.CircuitBreaker.SlowCallRateThreshold = 100
	cfg.CircuitBreaker.MinimumNumberOfCalls = 20
	cfg.CircuitBreaker.TimeoutSeconds = 120
	cfg.Retry.MaxRetries = 2
	cfg.Retry.IntervalMs = 2000
	cfg.RateLimiter.Rate = 500
	cfg.RateLimiter.Burst = 1000
	cfg.Timeout.DurationMs = 10000
	cfg.Bulkhead.MaxConcurrent = 50
	return cfg
}

// env helpers

func applyEnvInt(key string, dest *int) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*dest = n
		}
	}
}

func applyEnvUint8(key string, dest *uint8) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseUint(v, 10, 8); err == nil {
			*dest = uint8(n)
		}
	}
}

func applyEnvUint32(key string, dest *uint32) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil {
			*dest = uint32(n)
		}
	}
}

func applyEnvFloat64(key string, dest *float64) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			*dest = n
		}
	}
}

func applyEnvBool(key string, dest *bool) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseBool(v); err == nil {
			*dest = n
		}
	}
}

func applyEnvString(key string, dest *string) {
	if v := os.Getenv(key); v != "" {
		*dest = v
	}
}

// Duration helpers (convert config ms values to time.Duration)

func (c *CircuitBreakerConfig) Timeout() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}

func (c *CircuitBreakerConfig) SlowCallDuration() time.Duration {
	return time.Duration(c.SlowCallDurationMs) * time.Millisecond
}

func (r *RetryConfig) Interval() time.Duration {
	return time.Duration(r.IntervalMs) * time.Millisecond
}

func (r *RetryConfig) MaxInterval() time.Duration {
	return time.Duration(r.MaxIntervalMs) * time.Millisecond
}

func (r *RetryConfig) MaxDuration() time.Duration {
	return time.Duration(r.MaxDurationMs) * time.Millisecond
}

func (t *TimeoutConfig) Duration() time.Duration {
	return time.Duration(t.DurationMs) * time.Millisecond
}

func (b *BulkheadConfig) MaxWaitDuration() time.Duration {
	return time.Duration(b.MaxWaitDurationMs) * time.Millisecond
}

func (a *AdaptiveConfig) MinSlowCallDuration() time.Duration {
	return time.Duration(a.MinSlowCallDurationMs) * time.Millisecond
}

func (a *AdaptiveConfig) MaxSlowCallDuration() time.Duration {
	return time.Duration(a.MaxSlowCallDurationMs) * time.Millisecond
}

func (a *AdaptiveConfig) MaxTimeout() time.Duration {
	return time.Duration(a.MaxTimeoutSeconds) * time.Second
}

func (a *AdaptiveConfig) Timeout() time.Duration {
	return time.Duration(a.TimeoutSeconds) * time.Second
}
