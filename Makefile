# Makefile for caddy-sync

# Variables
BINARY_NAME=caddy-sync
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
REMOTE_HOST?=caddy
REMOTE_PATH?=/usr/local/bin/caddy-dns-sync
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X github.com/jeeftor/caddy-dns-sync/cmd.Version=$(VERSION) -X github.com/jeeftor/caddy-dns-sync/cmd.Commit=$(COMMIT) -X github.com/jeeftor/caddy-dns-sync/cmd.Date=$(BUILD_DATE)"

# Default target
help:
	@echo "Available targets:"
	@echo "  all            : Show this help message"
	@echo "  build          : Build the application"
	@echo "  clean          : Clean build artifacts"
	@echo "  test           : Run tests"
	@echo "  vet            : Run go vet"
	@echo "  fmt            : Format code"
	@echo "  web-install    : Install web UI dependencies"
	@echo "  web-build      : Build the React web UI assets"
	@echo "  web-dev        : Run the React web UI dev server"
	@echo "  web-lint       : Lint web UI (Zustand selectors, hooks, etc.)"
	@echo "  check          : Run all checks (fmt, vet, test, web-lint)"
	@echo "  install        : Install the application"
	@echo "  cross-build    : Cross-compile for multiple platforms"
	@echo "  install-remote  : Build linux/amd64 and deploy to REMOTE_HOST (default: caddy)"
	@echo "  install-service : Deploy and install as a systemd service on REMOTE_HOST"
	@echo "  uninstall-service: Remove the systemd service from REMOTE_HOST"
	@echo "  release-dry-run : Run GoReleaser in dry-run mode"
	@echo "  help           : Show this help message"

.PHONY: all help build clean test vet fmt web-install web-build web-dev web-lint check install cross-build install-remote release-dry-run

all: help

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

# Install web UI dependencies
web-install:
	@echo "Installing web UI dependencies..."
	cd web && npm install

# Build React web UI assets into internal/web/static
web-build:
	@echo "Building web UI assets..."
	cd web && npm run build

# Run React web UI dev server
web-dev:
	@echo "Starting web UI dev server..."
	cd web && npm run dev

# Lint web UI (catches unstable Zustand selectors, hook violations, etc.)
web-lint:
	@echo "Linting web UI..."
	cd web && npm run lint

# Run all checks
check: fmt vet test web-lint

# Install the application
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS)

# Cross-compile for multiple platforms
cross-build:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_arm64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_windows_amd64.exe

# Build and deploy linux/amd64 binary to a remote host over SSH
# Override defaults: make install-remote REMOTE_HOST=myserver REMOTE_PATH=/usr/local/bin/caddy-sync
build-linux:
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_amd64

# Keep file target for incremental builds (cross-build etc.)
dist/$(BINARY_NAME)_linux_amd64: build-linux

install-remote: build-linux
	@printf '\033[36m  →  \033[0mCopying to \033[97m$(REMOTE_HOST):$(REMOTE_PATH)\033[0m...\n'
	-ssh $(REMOTE_HOST) systemctl stop caddy-sync 2>/dev/null || true
	scp dist/$(BINARY_NAME)_linux_amd64 $(REMOTE_HOST):$(REMOTE_PATH)
	ssh $(REMOTE_HOST) chmod +x $(REMOTE_PATH)
	@printf '\033[33m  ⚠   Service stopped. Restart it:\033[0m\n'
	@printf '       \033[33mssh $(REMOTE_HOST) systemctl restart caddy-sync\033[0m\n'
	@printf '     or run \033[33mmake install-service\033[0m to also update the unit file.\n\n'
	@printf '\033[1;32m  ✓  Binary deployed to $(REMOTE_HOST):$(REMOTE_PATH)\033[0m\n\n'
	@printf '\033[1;36m── Next steps ──────────────────────────────────────────\033[0m\n\n'
	@printf '  \033[1m1.\033[0m Update config (if needed):\n'
	@printf '       \033[33mssh $(REMOTE_HOST) $(REMOTE_PATH) web --host 127.0.0.1\033[0m  (one-off)\n\n'
	@printf '  \033[1m2.\033[0m Restart the service:\n'
	@printf '       \033[33mssh $(REMOTE_HOST) systemctl restart caddy-sync\033[0m\n\n'
	@printf '  \033[1m3.\033[0m Verify everything is wired up:\n'
	@printf '       \033[33mssh $(REMOTE_HOST) $(REMOTE_PATH) doctor\033[0m\n\n'

# Build, deploy, update unit file, and restart the service on the remote host
install-service: build-linux
	@printf '\033[36m  →  \033[0mCopying to \033[97m$(REMOTE_HOST):$(REMOTE_PATH)\033[0m...\n'
	-ssh $(REMOTE_HOST) systemctl stop caddy-sync 2>/dev/null || true
	scp dist/$(BINARY_NAME)_linux_amd64 $(REMOTE_HOST):$(REMOTE_PATH)
	ssh $(REMOTE_HOST) chmod +x $(REMOTE_PATH)
	ssh $(REMOTE_HOST) $(REMOTE_PATH) install-service --start
	@printf '\n\033[1;32m  ✓  Service updated and restarted on $(REMOTE_HOST)\033[0m\n\n'

# Remove the systemd service from the remote host
uninstall-service:
	ssh $(REMOTE_HOST) $(REMOTE_PATH) uninstall-service

# Run GoReleaser in dry-run mode
release-dry-run:
	@echo "Running GoReleaser in dry-run mode..."
	goreleaser release --snapshot --clean --skip=publish
