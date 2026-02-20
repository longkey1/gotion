.PHONY: build clean test lint fmt tidy release

# Variables
BINARY_NAME := gotion
BUILD_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X github.com/longkey1/gotion/internal/version.Version=$(VERSION) -X github.com/longkey1/gotion/internal/version.Commit=$(COMMIT) -X github.com/longkey1/gotion/internal/version.Date=$(DATE)"

# Default target
all: build

# Build the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf dist

# Run tests
test:
	go test -v -race ./...

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Tidy dependencies
tidy:
	go mod tidy

# Release (requires goreleaser)
# Usage: make release type=patch dryrun=true
release:
	@if [ "$(dryrun)" = "true" ]; then \
		goreleaser release --snapshot --clean; \
	else \
		goreleaser release --clean; \
	fi

# Install locally
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOBIN)/$(BINARY_NAME)

# Help
help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  clean    - Clean build artifacts"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linter"
	@echo "  fmt      - Format code"
	@echo "  tidy     - Tidy dependencies"
	@echo "  release  - Create a release (use dryrun=true for dry run)"
	@echo "  install  - Install binary to GOBIN"
	@echo "  help     - Show this help"
