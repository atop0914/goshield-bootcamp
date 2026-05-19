# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-05-19

### Added
- Circuit Breaker pattern with sliding window (count-based and time-based)
- Adaptive Circuit Breaker with EMA-based dynamic thresholds
  - Failure rate EMA with configurable smoothing factor
  - Latency-based slow call detection
  - Consecutive failure limit for burst detection
  - Exponential timeout backoff on repeated trips
- Retry with multiple backoff strategies (fixed, exponential, exponential random, fibonacci)
- Rate Limiter with token bucket and sliding window implementations
- Timeout enforcement for operations
- Bulkhead pattern for limiting concurrent executions
- Fallback pattern for providing alternative responses
- HTTP middleware for all resilience patterns (net/http compatible)
- Composable decorator chain for combining patterns
- Composite executor for easy multi-pattern usage
- Zero-dependency metrics system (internal/metrics)
  - Collector interface + Registry pattern
  - Prometheus text format output
  - HTTP handler (/metrics endpoint)
  - BreakerCollector / RateLimiterCollector / BulkheadCollector / RetryCollector / TimeoutCollector / AdaptiveBreakerCollector
  - SimpleCounter / SimpleGauge / SimpleHistogram atomic metric types
- Configuration management (internal/config)
  - JSON config file loading
  - Environment variable overrides (GOSHIELD_* prefix)
  - Configuration validation with range and type checks
  - Hot-reload watcher (file polling, no external deps)
  - Preset configurations (Conservative / Aggressive)
  - Deep copy and serialization
- Comprehensive benchmark suite
  - Per-package detailed benchmarks (176+ benchmarks)
  - Parallel, contention, and throughput tests
  - CPU and memory profiling support
  - Benchmark comparison tools
- CLI tool for testing and benchmarking
- Advanced usage guide (docs/guide.md)
- Performance report (docs/performance-report.md)

### Architecture
- Clean separation between internal implementations and public API
- Composable design allows mixing and matching patterns
- Zero-dependency core (no external dependencies)
- Context-aware throughout

### Performance
| Pattern | Ops/sec | Latency (ns/op) | Allocs |
|---------|---------|-----------------|--------|
| Circuit Breaker | 4.6M | 280 | 0 |
| Adaptive Breaker | 2.5M | 450 | 0 |
| Rate Limiter | 11.5M | 103 | 0 |
| Bulkhead | 8.9M | 146 | 0 |
| Timeout | 515K | 2,701 | 576 B |
| Retry | 165M | 7 | 0 |
| Composite | 325K | 3,656 | 696 B |
