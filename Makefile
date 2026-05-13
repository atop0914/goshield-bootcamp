VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE  := github.com/atop0914/goshield

LDFLAGS := -s -w \
    -X $(MODULE)/internal/version.Version=$(VERSION) \
    -X $(MODULE)/internal/version.GitCommit=$(COMMIT) \
    -X $(MODULE)/internal/version.BuildDate=$(DATE)

.PHONY: all test lint build build-all clean

all: lint test build

## Run tests
test:
	go test -v -race -count=1 ./...

## Run tests with short flag (skip container tests)
test-short:
	go test -short -v -race -count=1 ./...

## Run benchmarks
bench:
	go test -bench=. -benchmem ./...

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
	rm -rf bin/ dist/

## Format code
fmt:
	gofmt -w .
	goimports -w .

## Run vet
vet:
	go vet ./...
