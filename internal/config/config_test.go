package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.CircuitBreaker == nil {
		t.Error("CircuitBreaker is nil")
	}
	if cfg.Retry == nil {
		t.Error("Retry is nil")
	}
	if cfg.RateLimiter == nil {
		t.Error("RateLimiter is nil")
	}
	if cfg.Timeout == nil {
		t.Error("Timeout is nil")
	}
	if cfg.Bulkhead == nil {
		t.Error("Bulkhead is nil")
	}
	if cfg.Metrics == nil {
		t.Error("Metrics is nil")
	}
	if cfg.HTTP == nil {
		t.Error("HTTP is nil")
	}

	// Check some default values
	if cfg.CircuitBreaker.FailureRateThreshold != 50 {
		t.Errorf("expected FailureRateThreshold=50, got %d", cfg.CircuitBreaker.FailureRateThreshold)
	}
	if cfg.RateLimiter.Rate != 100 {
		t.Errorf("expected RateLimiter.Rate=100, got %f", cfg.RateLimiter.Rate)
	}
	if cfg.Timeout.DurationMs != 5000 {
		t.Errorf("expected Timeout.DurationMs=5000, got %d", cfg.Timeout.DurationMs)
	}
}

func TestLoadBytes(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "empty JSON uses defaults",
			json:    "{}",
			wantErr: false,
		},
		{
			name: "full config",
			json: `{
				"circuit_breaker": {
					"name": "test",
					"max_requests": 5,
					"timeout_seconds": 30,
					"failure_rate_threshold": 60,
					"sliding_window_size": 200,
					"sliding_window_type": "count"
				},
				"retry": {
					"max_retries": 5,
					"strategy": "exponential",
					"interval_ms": 500,
					"multiplier": 2.5
				},
				"rate_limiter": {
					"type": "token_bucket",
					"rate": 1000,
					"burst": 2000
				},
				"timeout": {
					"duration_ms": 10000
				},
				"bulkhead": {
					"max_concurrent": 20
				}
			}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    "{invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadBytes([]byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("config is nil")
			}
		})
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()

	cfg := DefaultConfig()
	cfg.CircuitBreaker.Name = "from-file"
	data, _ := json.MarshalIndent(cfg, "", "  ")
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded.CircuitBreaker.Name != "from-file" {
		t.Errorf("expected name 'from-file', got %q", loaded.CircuitBreaker.Name)
	}

	// Non-existent file
	_, err = LoadFile(filepath.Join(dir, "nope.json"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := DefaultConfig()

	// Set env vars
	t.Setenv("GOSHIELD_CIRCUIT_BREAKER_TIMEOUT_SECONDS", "30")
	t.Setenv("GOSHIELD_RATE_LIMITER_RATE", "500.5")
	t.Setenv("GOSHIELD_TIMEOUT_DURATION_MS", "10000")
	t.Setenv("GOSHIELD_BULKHEAD_MAX_CONCURRENT", "50")
	t.Setenv("GOSHIELD_METRICS_ENABLED", "false")
	t.Setenv("GOSHIELD_HTTP_ADDR", ":9090")
	t.Setenv("GOSHIELD_ADAPTIVE_FAILURE_RATE_EMA_ALPHA", "0.7")
	t.Setenv("GOSHIELD_RETRY_MAX_RETRIES", "10")

	cfg.ApplyEnvOverrides()

	if cfg.CircuitBreaker.TimeoutSeconds != 30 {
		t.Errorf("expected TimeoutSeconds=30, got %d", cfg.CircuitBreaker.TimeoutSeconds)
	}
	if cfg.RateLimiter.Rate != 500.5 {
		t.Errorf("expected Rate=500.5, got %f", cfg.RateLimiter.Rate)
	}
	if cfg.Timeout.DurationMs != 10000 {
		t.Errorf("expected DurationMs=10000, got %d", cfg.Timeout.DurationMs)
	}
	if cfg.Bulkhead.MaxConcurrent != 50 {
		t.Errorf("expected MaxConcurrent=50, got %d", cfg.Bulkhead.MaxConcurrent)
	}
	if cfg.Metrics.Enabled != false {
		t.Error("expected Metrics.Enabled=false")
	}
	if cfg.HTTP.Addr != ":9090" {
		t.Errorf("expected HTTP.Addr=:9090, got %q", cfg.HTTP.Addr)
	}
	if cfg.Adaptive.FailureRateEMAAlpha != 0.7 {
		t.Errorf("expected FailureRateEMAAlpha=0.7, got %f", cfg.Adaptive.FailureRateEMAAlpha)
	}
	if cfg.Retry.MaxRetries != 10 {
		t.Errorf("expected MaxRetries=10, got %d", cfg.Retry.MaxRetries)
	}
}

func TestApplyEnvOverrides_Partial(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Retry = nil // No retry config

	t.Setenv("GOSHIELD_RETRY_MAX_RETRIES", "10")
	cfg.ApplyEnvOverrides()
	// Should not panic

	if cfg.Retry != nil {
		t.Error("expected Retry to remain nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{
			name:    "default is valid",
			mutate:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "failure_rate > 100",
			mutate: func(c *Config) {
				c.CircuitBreaker.FailureRateThreshold = 101
			},
			wantErr: true,
		},
		{
			name: "slow_call_rate > 100",
			mutate: func(c *Config) {
				c.CircuitBreaker.SlowCallRateThreshold = 150
			},
			wantErr: true,
		},
		{
			name: "zero sliding window size",
			mutate: func(c *Config) {
				c.CircuitBreaker.SlidingWindowSize = 0
			},
			wantErr: true,
		},
		{
			name: "invalid sliding window type",
			mutate: func(c *Config) {
				c.CircuitBreaker.SlidingWindowType = "invalid"
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			mutate: func(c *Config) {
				c.CircuitBreaker.TimeoutSeconds = -1
			},
			wantErr: true,
		},
		{
			name: "adaptive alpha out of range",
			mutate: func(c *Config) {
				c.Adaptive.FailureRateEMAAlpha = 1.5
			},
			wantErr: true,
		},
		{
			name: "adaptive min > max threshold",
			mutate: func(c *Config) {
				c.Adaptive.MinAdaptiveThreshold = 90
				c.Adaptive.MaxAdaptiveThreshold = 50
			},
			wantErr: true,
		},
		{
			name: "adaptive timeout_multiplier < 1",
			mutate: func(c *Config) {
				c.Adaptive.TimeoutMultiplier = 0.5
			},
			wantErr: true,
		},
		{
			name: "negative max_retries",
			mutate: func(c *Config) {
				c.Retry.MaxRetries = -1
			},
			wantErr: true,
		},
		{
			name: "invalid retry strategy",
			mutate: func(c *Config) {
				c.Retry.Strategy = "unknown"
			},
			wantErr: true,
		},
		{
			name: "zero rate",
			mutate: func(c *Config) {
				c.RateLimiter.Rate = 0
			},
			wantErr: true,
		},
		{
			name: "invalid rate limiter type",
			mutate: func(c *Config) {
				c.RateLimiter.Type = "leaky_bucket"
			},
			wantErr: true,
		},
		{
			name: "zero timeout duration",
			mutate: func(c *Config) {
				c.Timeout.DurationMs = 0
			},
			wantErr: true,
		},
		{
			name: "zero bulkhead concurrent",
			mutate: func(c *Config) {
				c.Bulkhead.MaxConcurrent = 0
			},
			wantErr: true,
		},
		{
			name: "multiple errors",
			mutate: func(c *Config) {
				c.CircuitBreaker.FailureRateThreshold = 200
				c.Retry.MaxRetries = -1
				c.RateLimiter.Rate = -1
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestValidate_NilSections(t *testing.T) {
	cfg := &Config{} // All sections nil
	if err := cfg.Validate(); err != nil {
		t.Errorf("nil sections should be valid: %v", err)
	}
}

func TestDeepCopy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CircuitBreaker.Name = "original"
	cfg.RateLimiter.Rate = 999

	cp := cfg.DeepCopy()
	cp.CircuitBreaker.Name = "copy"
	cp.RateLimiter.Rate = 111

	if cfg.CircuitBreaker.Name != "original" {
		t.Error("deep copy modified original")
	}
	if cfg.RateLimiter.Rate != 999 {
		t.Error("deep copy modified original rate")
	}
	if cp.CircuitBreaker.Name != "copy" {
		t.Error("copy name not updated")
	}
}

func TestToJSON(t *testing.T) {
	cfg := DefaultConfig()
	data, err := cfg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	// Round-trip
	loaded, err := LoadBytes(data)
	if err != nil {
		t.Fatalf("LoadBytes: %v", err)
	}
	if loaded.CircuitBreaker.FailureRateThreshold != cfg.CircuitBreaker.FailureRateThreshold {
		t.Error("round-trip mismatch")
	}
}

func TestPresets(t *testing.T) {
	c := PresetConservative()
	if c.CircuitBreaker.FailureRateThreshold != 30 {
		t.Errorf("conservative FailureRateThreshold=30, got %d", c.CircuitBreaker.FailureRateThreshold)
	}
	if err := c.Validate(); err != nil {
		t.Errorf("conservative preset invalid: %v", err)
	}

	a := PresetAggressive()
	if a.CircuitBreaker.FailureRateThreshold != 80 {
		t.Errorf("aggressive FailureRateThreshold=80, got %d", a.CircuitBreaker.FailureRateThreshold)
	}
	if err := a.Validate(); err != nil {
		t.Errorf("aggressive preset invalid: %v", err)
	}
}

func TestDurationHelpers(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.CircuitBreaker.Timeout() != 60*time.Second {
		t.Error("CB timeout mismatch")
	}
	if cfg.Retry.Interval() != 1000*time.Millisecond {
		t.Error("retry interval mismatch")
	}
	if cfg.Timeout.Duration() != 5000*time.Millisecond {
		t.Error("timeout duration mismatch")
	}
	if cfg.Bulkhead.MaxWaitDuration() != 5000*time.Millisecond {
		t.Error("bulkhead wait duration mismatch")
	}
	if cfg.Adaptive.MinSlowCallDuration() != 100*time.Millisecond {
		t.Error("adaptive min slow call duration mismatch")
	}
	if cfg.Adaptive.MaxTimeout() != 300*time.Second {
		t.Error("adaptive max timeout mismatch")
	}
}

func TestWatcher(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	cfg.CircuitBreaker.Name = "v1"
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	var received *Config
	onChange := func(c *Config) {
		received = c
	}

	w, err := NewWatcher(path, onChange, WithInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	w.Start()
	if !w.IsRunning() {
		t.Error("expected watcher to be running")
	}

	// Sleep briefly to ensure initial mod time is recorded
	time.Sleep(100 * time.Millisecond)

	// Modify the file with different content (different size ensures detection)
	cfg.CircuitBreaker.Name = "v2"
	cfg.RateLimiter.Rate = 999
	data, _ = json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(path, data, 0644)

	// Wait for detection
	time.Sleep(300 * time.Millisecond)

	w.Stop()
	if w.IsRunning() {
		t.Error("expected watcher to be stopped")
	}

	if received == nil {
		t.Fatal("onChange was not called")
	}
	if received.CircuitBreaker.Name != "v2" {
		t.Errorf("expected name 'v2', got %q", received.CircuitBreaker.Name)
	}
}

func TestWatcher_InvalidFile(t *testing.T) {
	_, err := NewWatcher("/nonexistent/config.json", func(c *Config) {})
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestWatcher_NilCallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("{}"), 0644)

	_, err := NewWatcher(path, nil)
	if err == nil {
		t.Error("expected error for nil callback")
	}
}

func TestWatcher_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("{}"), 0644)

	var reloadErr error
	w, _ := NewWatcher(path, func(c *Config) {}, WithInterval(50*time.Millisecond), WithOnError(func(err error) {
		reloadErr = err
	}))
	w.Start()

	// Write invalid JSON
	os.WriteFile(path, []byte("{bad"), 0644)
	time.Sleep(200 * time.Millisecond)
	w.Stop()

	if reloadErr == nil {
		t.Error("expected error callback for invalid JSON")
	}
}

func TestEnvOverridesIgnoredInvalid(t *testing.T) {
	cfg := DefaultConfig()
	t.Setenv("GOSHIELD_CIRCUIT_BREAKER_TIMEOUT_SECONDS", "notanumber")
	cfg.ApplyEnvOverrides()
	// Should keep default value
	if cfg.CircuitBreaker.TimeoutSeconds != 60 {
		t.Errorf("expected default 60, got %d", cfg.CircuitBreaker.TimeoutSeconds)
	}
}
