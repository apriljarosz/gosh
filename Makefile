# Makefile for gosh shell

# Variables
BINARY_NAME=gosh
BUILD_DIR=build
TEST_BINARY=$(BUILD_DIR)/gosh_test

# Default target
.PHONY: all
all: build

# Build the shell
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

# Run the shell
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run unit tests
.PHONY: test
test:
	@echo "Running unit tests..."
	go test -v ./internal/...

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test -v ./test/...

# Run all tests
.PHONY: test-all
test-all: test test-integration

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (if golangci-lint is available)
.PHONY: lint
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f gosh_test

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Development workflow: format, lint, test
.PHONY: dev
dev: fmt lint test-all

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build the shell binary"
	@echo "  run             - Build and run the shell"
	@echo "  test            - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-all        - Run all tests"
	@echo "  fmt             - Format code"
	@echo "  lint            - Lint code (requires golangci-lint)"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Install and tidy dependencies"
	@echo "  dev             - Run development workflow (fmt, lint, test-all)"
	@echo "  help            - Show this help message"