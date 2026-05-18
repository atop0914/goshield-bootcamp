// Package main demonstrates GoShield configuration management.
//
// Usage:
//
//	go run ./examples/config
//	GOSHIELD_RATE_LIMITER_RATE=500 go run ./examples/config
//	go run ./examples/config -file goshield.json
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/atop0914/goshield/internal/config"
)

func main() {
	filePath := flag.String("file", "", "path to JSON config file")
	preset := flag.String("preset", "", "config preset: conservative, moderate, aggressive")
	showJSON := flag.Bool("json", false, "dump config as JSON")
	flag.Parse()

	var cfg *config.Config
	var err error

	switch {
	case *filePath != "":
		fmt.Printf("Loading config from %s\n", *filePath)
		cfg, err = config.LoadFile(*filePath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	case *preset == "conservative":
		fmt.Println("Using conservative preset")
		cfg = config.PresetConservative()
	case *preset == "aggressive":
		fmt.Println("Using aggressive preset")
		cfg = config.PresetAggressive()
	default:
		fmt.Println("Using default config")
		cfg = config.DefaultConfig()
	}

	// Apply environment variable overrides
	fmt.Println("\nApplying environment overrides (GOSHIELD_*)...")
	cfg.ApplyEnvOverrides()

	// Validate
	fmt.Println("Validating config...")
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config validation failed: %v", err)
	}
	fmt.Println("✓ Config is valid")

	if *showJSON {
		data, err := cfg.ToJSON()
		if err != nil {
			log.Fatalf("Failed to serialize config: %v", err)
		}
		fmt.Println("\n" + string(data))
		return
	}

	// Print config summary
	fmt.Println("\n=== GoShield Configuration ===")
	fmt.Printf("Circuit Breaker: name=%q, threshold=%d%%, timeout=%v\n",
		cfg.CircuitBreaker.Name,
		cfg.CircuitBreaker.FailureRateThreshold,
		cfg.CircuitBreaker.Timeout())
	fmt.Printf("Adaptive:        ema_alpha=%.2f, threshold=[%.0f%%, %.0f%%], timeout_mult=%.1f\n",
		cfg.Adaptive.FailureRateEMAAlpha,
		cfg.Adaptive.MinAdaptiveThreshold,
		cfg.Adaptive.MaxAdaptiveThreshold,
		cfg.Adaptive.TimeoutMultiplier)
	fmt.Printf("Retry:           strategy=%s, max=%d, interval=%v, mult=%.1f\n",
		cfg.Retry.Strategy,
		cfg.Retry.MaxRetries,
		cfg.Retry.Interval(),
		cfg.Retry.Multiplier)
	fmt.Printf("Rate Limiter:    type=%s, rate=%.0f/s, burst=%d\n",
		cfg.RateLimiter.Type,
		cfg.RateLimiter.Rate,
		cfg.RateLimiter.Burst)
	fmt.Printf("Timeout:         %v\n", cfg.Timeout.Duration())
	fmt.Printf("Bulkhead:        max=%d, wait=%v\n",
		cfg.Bulkhead.MaxConcurrent,
		cfg.Bulkhead.MaxWaitDuration())
	fmt.Printf("Metrics:         enabled=%v, namespace=%q, endpoint=%q\n",
		cfg.Metrics.Enabled,
		cfg.Metrics.Namespace,
		cfg.Metrics.Endpoint)
	fmt.Printf("HTTP:            addr=%q, metrics=%q\n",
		cfg.HTTP.Addr,
		cfg.HTTP.MetricsAddr)

	// Demonstrate hot reload
	fmt.Println("\n--- Hot Reload Demo ---")
	fmt.Println("Watch a config file for live changes:")

	// Only run watcher demo if a file was provided
	if *filePath != "" {
		fmt.Printf("Watching %s for changes (press Ctrl+C to stop)...\n", *filePath)
		w, err := config.NewWatcher(*filePath, func(newCfg *config.Config) {
			fmt.Printf("🔄 Config reloaded! New rate_limit=%.0f\n", newCfg.RateLimiter.Rate)
		}, config.WithInterval(2000000000)) // 2s for demo
		if err != nil {
			log.Fatalf("Failed to create watcher: %v", err)
		}
		w.Start()
		fmt.Println("(Watcher started - modify the file to see hot reload)")
		// In a real app, you'd block here with a signal handler
		// For demo, we just stop immediately
		w.Stop()
		fmt.Println("Watcher stopped.")
	} else {
		fmt.Println("  w, _ := config.NewWatcher(\"goshield.json\", func(cfg *config.Config) {")
		fmt.Println("      // Reconfigure your components with new settings")
		fmt.Println("  })")
		fmt.Println("  w.Start()")
		fmt.Println("  defer w.Stop()")
	}

	// Show environment variable examples
	fmt.Println("\n--- Environment Variable Examples ---")
	fmt.Println("  GOSHIELD_CIRCUIT_BREAKER_TIMEOUT_SECONDS=30")
	fmt.Println("  GOSHIELD_RATE_LIMITER_RATE=500")
	fmt.Println("  GOSHIELD_TIMEOUT_DURATION_MS=10000")
	fmt.Println("  GOSHIELD_METRICS_ENABLED=false")
	fmt.Println("  GOSHIELD_HTTP_ADDR=:9090")

	_ = os.Environ() // avoid unused import
}
