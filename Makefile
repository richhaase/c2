# c2 development tasks

.PHONY: help build install test lint vet fmt check clean

# Show available targets
help:
	@echo "Available targets:"
	@echo "  build   - Build the c2 binary with version information"
	@echo "  install - Install c2 to GOPATH/bin"
	@echo "  test    - Run all unit tests"
	@echo "  fmt     - Format Go source code"
	@echo "  lint    - Run golangci-lint v2"
	@echo "  vet     - Run go vet"
	@echo "  check   - Run all quality checks (fmt, vet, lint, tests)"
	@echo "  clean   - Clean build artifacts and test cache"

# Build the c2 binary with version information
build:
	@echo "Building c2 with version information..."
	@mkdir -p bin
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "none"); \
	DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	if ! go build -ldflags "-s -w -X main.version=$$VERSION -X main.commit=$$COMMIT -X main.date=$$DATE" -o bin/c2 ./cmd/c2; then \
		echo "Build failed"; \
		exit 1; \
	fi; \
	echo "Built versioned c2 binary to bin/ (version: $$VERSION)"

# Install c2 to GOPATH/bin
install:
	@echo "Installing c2..."
	@go install ./cmd/c2
	@echo "Installed c2 to $$(go env GOPATH)/bin"

# Run all unit tests
test:
	@echo "Running unit tests..."
	@go test ./...
	@echo "Unit tests passed!"

# Format Go source code
fmt:
	@echo "Formatting Go source code..."
	@goimports -w .
	@echo "Formatting complete!"

# Run golangci-lint v2
lint:
	@echo "Running golangci-lint v2..."
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --timeout=10m ./...
	@echo "Linting passed!"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet passed!"

# Run all quality checks (format, vet, lint, tests)
check: fmt vet lint test
	@echo "All checks passed."

# Clean build artifacts and test cache
clean:
	@echo "Cleaning build artifacts and caches..."
	@rm -rf bin
	@go clean
	@go clean -testcache
	@echo "Build artifacts and test cache cleaned"
