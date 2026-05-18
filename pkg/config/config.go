// Package config provides configuration management for GoShield.
//
// It supports loading configuration from JSON files, environment variable
// overrides, validation, and hot-reload via file watching.
//
// Usage:
//
//	// Load from file
//	cfg, err := config.LoadFile("goshield.json")
//
//	// Apply environment overrides
//	cfg.ApplyEnvOverrides()
//
//	// Validate
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use with hot reload
//	w, _ := config.NewWatcher("goshield.json", func(cfg *config.Config) {
//	    // Reconfigure your components
//	})
//	w.Start()
//	defer w.Stop()
package config

import (
	"time"

	internal "github.com/atop0914/goshield/internal/config"
)

// Config is the top-level GoShield configuration.
type Config = internal.Config

// CircuitBreakerConfig configures the standard circuit breaker.
type CircuitBreakerConfig = internal.CircuitBreakerConfig

// AdaptiveConfig configures the adaptive circuit breaker.
type AdaptiveConfig = internal.AdaptiveConfig

// RetryConfig configures retry behavior.
type RetryConfig = internal.RetryConfig

// RateLimiterConfig configures rate limiting.
type RateLimiterConfig = internal.RateLimiterConfig

// TimeoutConfig configures timeout behavior.
type TimeoutConfig = internal.TimeoutConfig

// BulkheadConfig configures bulkhead behavior.
type BulkheadConfig = internal.BulkheadConfig

// MetricsConfig configures the metrics system.
type MetricsConfig = internal.MetricsConfig

// HTTPConfig configures HTTP middleware defaults.
type HTTPConfig = internal.HTTPConfig

// DefaultConfig returns a Config with sensible defaults.
var DefaultConfig = internal.DefaultConfig

// LoadFile reads a JSON configuration file and returns a Config.
var LoadFile = internal.LoadFile

// LoadBytes parses JSON data and returns a Config.
var LoadBytes = internal.LoadBytes

// PresetConservative returns a config with conservative settings.
var PresetConservative = internal.PresetConservative

// PresetAggressive returns a config with aggressive settings.
var PresetAggressive = internal.PresetAggressive

// Watcher watches a config file for changes and triggers a reload callback.
type Watcher = internal.Watcher

// WatcherOption configures a Watcher.
type WatcherOption = internal.WatcherOption

// NewWatcher creates a new file watcher for the given config path.
var NewWatcher = internal.NewWatcher

// WithInterval sets the polling interval. Default is 5 seconds.
var WithInterval = internal.WithInterval

// WithOnError sets the error callback.
var WithOnError = internal.WithOnError

// Re-export time.Duration for convenience
type Duration = time.Duration
