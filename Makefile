# Makefile for unboundCLI

# Variables
BINARY_NAME=unboundCLI
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X github.com/jeeftor/unboundCLI/cmd.Version=$(VERSION) -X github.com/jeeftor/unboundCLI/cmd.Commit=$(COMMIT) -X github.com/jeeftor/unboundCLI/cmd.Date=$(BUILD_DATE)"

.PHONY: all build clean test vet fmt check install release-dry-run

# Default target
all: check build

# Build the application
build:
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	go build $(LDFLAGS) -o $(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run all checks
check: fmt vet test

# Install the application
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS)

# Cross-compile for multiple platforms
cross-build:
	@echo "Cross-compiling for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_amd64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_arm64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_amd64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_arm64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_windows_amd64.exe

# Run GoReleaser in dry-run mode
release-dry-run:
	@echo "Running GoReleaser in dry-run mode..."
	goreleaser release --snapshot --clean --skip=publish

# Help target
help:
	@echo "Available targets:"
	@echo "  all            : Run checks and build the application (default)"
	@echo "  build          : Build the application"
	@echo "  clean          : Clean build artifacts"
	@echo "  test           : Run tests"
	@echo "  vet            : Run go vet"
	@echo "  fmt            : Format code"
	@echo "  check          : Run all checks (fmt, vet, test)"
	@echo "  install        : Install the application"
	@echo "  cross-build    : Cross-compile for multiple platforms"
	@echo "  release-dry-run: Run GoReleaser in dry-run mode"
	@echo "  help           : Show this help message"
