VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE  := github.com/atop0914/goshield

LDFLAGS := -s -w \
    -X $(MODULE)/internal/version.Version=$(VERSION) \
    -X $(MODULE)/internal/version.GitCommit=$(COMMIT) \
    -X $(MODULE)/internal/version.BuildDate=$(DATE)

.PHONY: all test lint build build-clean clean fmt vet bench bench-short bench-all bench-cpu bench-mem bench-compare

all: lint test build

## Run tests
test:
	go test -v -race -count=1 ./...

## Run tests with short flag (skip container tests)
test-short:
	go test -short -v -race -count=1 ./...

## Run benchmarks (default)
bench:
	go test -bench=. -benchmem ./...

## Run benchmarks with short flag (skip container tests)
bench-short:
	go test -short -bench=. -benchmem ./...

## Run all benchmarks (including per-package)
bench-all:
	@echo "=== Running All Benchmarks ==="
	@echo ""
	@echo "--- Top-level Benchmarks ---"
	go test -bench=. -benchmem ./benchmarks/...
	@echo ""
	@echo "--- Breaker Benchmarks ---"
	go test -bench=. -benchmem ./internal/breaker/...
	@echo ""
	@echo "--- Rate Limiter Benchmarks ---"
	go test -bench=. -benchmem ./internal/ratelimit/...
	@echo ""
	@echo "--- Bulkhead Benchmarks ---"
	go test -bench=. -benchmem ./internal/bulkhead/...
	@echo ""
	@echo "--- Backoff Benchmarks ---"
	go test -bench=. -benchmem ./internal/backoff/...
	@echo ""
	@echo "--- Timeout Benchmarks ---"
	go test -bench=. -benchmem ./internal/timeout/...
	@echo ""
	@echo "--- Metrics Benchmarks ---"
	go test -bench=. -benchmem ./internal/metrics/...
	@echo ""
	@echo "=== All Benchmarks Complete ==="

## Run CPU profiling benchmarks
bench-cpu:
	@mkdir -p profiles
	@echo "=== CPU Profiling ==="
	go test -bench=BenchmarkCircuitBreaker_Execute_Closed -benchmem -cpuprofile=profiles/cpu_circuit_breaker.prof ./internal/breaker/...
	go test -bench=BenchmarkRateLimiter_Allow -benchmem -cpuprofile=profiles/cpu_rate_limiter.prof ./internal/ratelimit/...
	go test -bench=BenchmarkBulkhead_Execute -benchmem -cpuprofile=profiles/cpu_bulkhead.prof ./internal/bulkhead/...
	go test -bench=BenchmarkTimeout_Execute -benchmem -cpuprofile=profiles/cpu_timeout.prof ./internal/timeout/...
	go test -bench=BenchmarkRetry_Execute_Success -benchmem -cpuprofile=profiles/cpu_retry.prof ./internal/backoff/...
	go test -bench=BenchmarkAdaptiveBreaker_Execute_Closed -benchmem -cpuprofile=profiles/cpu_adaptive.prof ./internal/breaker/...
	go test -bench=BenchmarkSimpleCounter_Inc -benchmem -cpuprofile=profiles/cpu_metrics.prof ./internal/metrics/...
	@echo "=== CPU Profiles Generated ==="
	@echo "Analyze with: go tool pprof profiles/cpu_*.prof"

## Run memory profiling benchmarks
bench-mem:
	@mkdir -p profiles
	@echo "=== Memory Profiling ==="
	go test -bench=BenchmarkCircuitBreaker_NoAlloc -benchmem -memprofile=profiles/mem_circuit_breaker.prof ./internal/breaker/...
	go test -bench=BenchmarkRateLimiter_NoAlloc -benchmem -memprofile=profiles/mem_rate_limiter.prof ./internal/ratelimit/...
	go test -bench=BenchmarkBulkhead_NoAlloc -benchmem -memprofile=profiles/mem_bulkhead.prof ./internal/bulkhead/...
	go test -bench=BenchmarkTimeout_NoAlloc -benchmem -memprofile=profiles/mem_timeout.prof ./internal/timeout/...
	go test -bench=BenchmarkAdaptiveBreaker_NoAlloc -benchmem -memprofile=profiles/mem_adaptive.prof ./internal/breaker/...
	go test -bench=BenchmarkSimpleCounter_NoAlloc -benchmem -memprofile=profiles/mem_metrics.prof ./internal/metrics/...
	@echo "=== Memory Profiles Generated ==="
	@echo "Analyze with: go tool pprof profiles/mem_*.prof"

## Run parallel benchmarks
bench-parallel:
	@echo "=== Parallel Benchmarks ==="
	go test -bench=Parallel -benchmem ./benchmarks/...
	go test -bench=Parallel -benchmem ./internal/breaker/...
	go test -bench=Parallel -benchmem ./internal/ratelimit/...
	go test -bench=Parallel -benchmem ./internal/bulkhead/...
	go test -bench=Parallel -benchmem ./internal/backoff/...
	go test -bench=Parallel -benchmem ./internal/timeout/...
	go test -bench=Parallel -benchmem ./internal/metrics/...
	@echo "=== Parallel Benchmarks Complete ==="

## Run contention benchmarks
bench-contention:
	@echo "=== Contention Benchmarks ==="
	go test -bench=Contention -benchmem ./benchmarks/...
	go test -bench=Contention -benchmem ./internal/breaker/...
	go test -bench=Contention -benchmem ./internal/ratelimit/...
	go test -bench=Contention -benchmem ./internal/bulkhead/...
	@echo "=== Contention Benchmarks Complete ==="

## Run throughput benchmarks
bench-throughput:
	@echo "=== Throughput Benchmarks ==="
	go test -bench=Throughput -benchmem ./benchmarks/...
	go test -bench=Throughput -benchmem ./internal/breaker/...
	go test -bench=Throughput -benchmem ./internal/ratelimit/...
	go test -bench=Throughput -benchmem ./internal/bulkhead/...
	go test -bench=Throughput -benchmem ./internal/timeout/...
	@echo "=== Throughput Benchmarks Complete ==="

## Run comparison benchmarks
bench-compare:
	@echo "=== Comparison Benchmarks ==="
	go test -bench=Comparison -benchmem ./benchmarks/...
	go test -bench=Comparison -benchmem ./internal/breaker/...
	go test -bench=Comparison -benchmem ./internal/ratelimit/...
	go test -bench=Comparison -benchmem ./internal/backoff/...
	go test -bench=Comparison -benchmem ./internal/timeout/...
	go test -bench=Comparison -benchmem ./internal/metrics/...
	@echo "=== Comparison Benchmarks Complete ==="

## Run benchmarks and save results
bench-save:
	@mkdir -p benchmark-results
	@echo "=== Saving Benchmark Results ==="
	@echo "Timestamp: $(shell date -u +%Y-%m-%dT%H:%M:%SZ)" > benchmark-results/latest.txt
	@echo "Commit: $(COMMIT)" >> benchmark-results/latest.txt
	@echo "Version: $(VERSION)" >> benchmark-results/latest.txt
	@echo "" >> benchmark-results/latest.txt
	@echo "--- Top-level Benchmarks ---" >> benchmark-results/latest.txt
	go test -bench=. -benchmem ./benchmarks/... 2>&1 | tee -a benchmark-results/latest.txt
	@echo "" >> benchmark-results/latest.txt
	@echo "--- Breaker Benchmarks ---" >> benchmark-results/latest.txt
	go test -bench=. -benchmem ./internal/breaker/... 2>&1 | tee -a benchmark-results/latest.txt
	@echo "" >> benchmark-results/latest.txt
	@echo "--- Rate Limiter Benchmarks ---" >> benchmark-results/latest.txt
	go test -bench=. -benchmem ./internal/ratelimit/... 2>&1 | tee -a benchmark-results/latest.txt
	@echo "" >> benchmark-results/latest.txt
	@echo "--- Bulkhead Benchmarks ---" >> benchmark-results/latest.txt
	go test -bench=. -benchmem ./internal/bulkhead/... 2>&1 | tee -a benchmark-results/latest.txt
	@echo "" >> benchmark-results/latest.txt
	@echo "--- Backoff Benchmarks ---" >> benchmark-results/latest.txt
	go test -bench=. -benchmem ./internal/backoff/... 2>&1 | tee -a benchmark-results/latest.txt
	@echo "" >> benchmark-results/latest.txt
	@echo "--- Timeout Benchmarks ---" >> benchmark-results/latest.txt
	go test -bench=. -benchmem ./internal/timeout/... 2>&1 | tee -a benchmark-results/latest.txt
	@echo "" >> benchmark-results/latest.txt
	@echo "--- Metrics Benchmarks ---" >> benchmark-results/latest.txt
	go test -bench=. -benchmem ./internal/metrics/... 2>&1 | tee -a benchmark-results/latest.txt
	@echo "" >> benchmark-results/latest.txt
	@echo "=== Results Saved to benchmark-results/latest.txt ==="

## Compare benchmarks with previous results
bench-diff:
	@if [ ! -f benchmark-results/baseline.txt ]; then \
		echo "No baseline found. Run 'make bench-save' first to create baseline."; \
		exit 1; \
	fi
	@echo "=== Benchmark Comparison ==="
	@echo "Baseline: $(shell head -3 benchmark-results/baseline.txt)"
	@echo "Current:  $(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
	@echo ""
	@echo "Run 'benchstat benchmark-results/baseline.txt benchmark-results/latest.txt' for detailed comparison"

## Save current results as baseline
bench-baseline:
	@if [ ! -f benchmark-results/latest.txt ]; then \
		echo "No results found. Run 'make bench-save' first."; \
		exit 1; \
	fi
	cp benchmark-results/latest.txt benchmark-results/baseline.txt
	@echo "Baseline saved to benchmark-results/baseline.txt"

## Run linter
lint:
	golangci-lint run ./...

## Build binary
build:
	go build -ldflags="$(LDFLAGS)" -o bin/goshield ./cmd/goshield

## Build for all platforms
build-all:
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/goshield-linux-amd64 ./cmd/goshield
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/goshield-linux-arm64 ./cmd/goshield
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/goshield-darwin-amd64 ./cmd/goshield
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/goshield-darwin-arm64 ./cmd/goshield
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/goshield-windows-amd64.exe ./cmd/goshield

## Clean build artifacts
clean:
	rm -rf bin/ dist/ profiles/ benchmark-results/

## Format code
fmt:
	gofmt -w .
	goimports -w .

## Run vet
vet:
	go vet ./...

## Generate benchmark report
bench-report:
	@mkdir -p benchmark-results
	@echo "=== Generating Benchmark Report ==="
	@echo "# GoShield Performance Report" > benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "Generated: $(shell date -u +%Y-%m-%dT%H:%M:%SZ)" >> benchmark-results/report.md
	@echo "Commit: $(COMMIT)" >> benchmark-results/report.md
	@echo "Version: $(VERSION)" >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "## Summary" >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "| Pattern | Single-threaded (ns/op) | Parallel (ns/op) | Allocs (B/op) |" >> benchmark-results/report.md
	@echo "|---------|------------------------|------------------|---------------|" >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "## Detailed Results" >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "### Circuit Breaker" >> benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	go test -bench=BenchmarkCircuitBreaker -benchmem ./internal/breaker/... 2>&1 | tee -a benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "### Rate Limiter" >> benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	go test -bench=BenchmarkRateLimiter -benchmem ./internal/ratelimit/... 2>&1 | tee -a benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "### Bulkhead" >> benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	go test -bench=BenchmarkBulkhead -benchmem ./internal/bulkhead/... 2>&1 | tee -a benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "### Backoff" >> benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	go test -bench=BenchmarkBackoff -benchmem ./internal/backoff/... 2>&1 | tee -a benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "### Timeout" >> benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	go test -bench=BenchmarkTimeout -benchmem ./internal/timeout/... 2>&1 | tee -a benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "### Adaptive Breaker" >> benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	go test -bench=BenchmarkAdaptiveBreaker -benchmem ./internal/breaker/... 2>&1 | tee -a benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "### Metrics" >> benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	go test -bench=BenchmarkSimple -benchmem ./internal/metrics/... 2>&1 | tee -a benchmark-results/report.md
	@echo '```' >> benchmark-results/report.md
	@echo "" >> benchmark-results/report.md
	@echo "=== Report Generated: benchmark-results/report.md ==="

## Show help
help:
	@echo "GoShield Makefile Targets:"
	@echo ""
	@echo "  make test           - Run all tests"
	@echo "  make test-short     - Run tests (skip container tests)"
	@echo "  make bench          - Run benchmarks (default)"
	@echo "  make bench-short    - Run benchmarks (skip container tests)"
	@echo "  make bench-all      - Run all benchmarks (per-package)"
	@echo "  make bench-cpu      - Run CPU profiling benchmarks"
	@echo "  make bench-mem      - Run memory profiling benchmarks"
	@echo "  make bench-parallel - Run parallel benchmarks"
	@echo "  make bench-contention - Run contention benchmarks"
	@echo "  make bench-throughput - Run throughput benchmarks"
	@echo "  make bench-compare  - Run comparison benchmarks"
	@echo "  make bench-save     - Run benchmarks and save results"
	@echo "  make bench-diff     - Compare with baseline results"
	@echo "  make bench-baseline - Save current results as baseline"
	@echo "  make bench-report   - Generate markdown benchmark report"
	@echo "  make lint           - Run linter"
	@echo "  make build          - Build binary"
	@echo "  make build-all      - Build for all platforms"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run vet"
	@echo "  make help           - Show this help"
