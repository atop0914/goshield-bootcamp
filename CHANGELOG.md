# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-05-14

### Added
- Circuit Breaker pattern with sliding window (count-based and time-based)
- Retry with multiple backoff strategies (fixed, exponential, exponential random, fibonacci)
- Rate Limiter with token bucket and sliding window implementations
- Timeout enforcement for operations
- Bulkhead pattern for limiting concurrent executions
- Fallback pattern for providing alternative responses
- HTTP middleware for all resilience patterns (net/http compatible)
- Composable decorator chain for combining patterns
- Composite executor for easy multi-pattern usage
- Prometheus metrics integration for all patterns
- CLI tool for testing and benchmarking
- Comprehensive test suite
- Documentation and examples

### Architecture
- Clean separation between internal implementations and public API
- Composable design allows mixing and matching patterns
- Zero-dependency core (only prometheus for metrics)
- Context-aware throughout
