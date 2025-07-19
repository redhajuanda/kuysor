.PHONY: help test test-verbose test-cover format lint clean bench mod-tidy

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Testing
test: ## Run tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

test-cover: ## Run tests with coverage report
	go test -cover ./...

test-race: ## Run tests with race detection
	go test -race ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

# Code quality
format: ## Format code using gofmt
	gofmt -s -w .

lint: ## Run basic linting with go vet
	go vet ./...

# Dependencies
mod-tidy: ## Tidy and verify module dependencies
	go mod tidy
	go mod verify

# Build
build: ## Build the project (mainly for verification)
	go build ./...

# Clean
clean: ## Clean build artifacts and test cache
	go clean ./...
	go clean -testcache

# Development workflow
check: format lint test ## Run format, lint, and test (pre-commit checks)

# Coverage report with HTML output
test-cover-html: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# All checks for CI
ci: mod-tidy format lint test-race test-cover ## Run all CI checks 