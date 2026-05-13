# GoShield Benchmarks

Run benchmarks:

```bash
# All benchmarks
go test -bench=. -benchmem ./benchmarks/

# Specific benchmark
go test -bench=BenchmarkCircuitBreaker -benchmem ./benchmarks/

# With CPU profiling
go test -bench=BenchmarkCompositeExecutor -benchmem -cpuprofile=cpu.prof ./benchmarks/
go tool pprof cpu.prof
```

## Expected Performance

| Pattern | Ops/sec | Latency (ns/op) | Allocs/op |
|---------|---------|------------------|-----------|
| Circuit Breaker (closed) | 10M+ | ~100 | 0 |
| Rate Limiter (allow) | 50M+ | ~25 | 0 |
| Bulkhead | 10M+ | ~100 | 0 |
| Timeout | 5M+ | ~200 | 0 |
| Composite | 2M+ | ~500 | 0 |
