# c2 development tasks

.PHONY: help build install test test-coverage fmt fmt-check lint vet tidy clean staticcheck check

help:
	@echo "Available targets:"
	@echo "  build         - Build the c2 binary with version information"
	@echo "  install       - Install c2 to GOBIN/GOPATH"
	@echo "  test          - Run all unit tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  fmt           - Format Go source code"
	@echo "  fmt-check     - Check Go source formatting"
	@echo "  lint          - Run golangci-lint v2"
	@echo "  vet           - Run go vet"
	@echo "  tidy          - Tidy go modules"
	@echo "  clean         - Clean build artifacts and test cache"
	@echo "  staticcheck   - Run staticcheck"
	@echo "  check         - Run all quality checks"

build:
	@echo "Building c2 with version information..."
	@mkdir -p bin
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "none"); \
	DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	go build -ldflags "-X main.version=$$VERSION -X main.commit=$$COMMIT -X main.date=$$DATE" -o bin/c2 ./cmd/c2
	@echo "Built c2 binary to bin/"

install:
	@echo "Installing c2..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "none"); \
	DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	go install -ldflags "-X main.version=$$VERSION -X main.commit=$$COMMIT -X main.date=$$DATE" ./cmd/c2
	@GOBIN=$$(go env GOBIN); \
	if [ -n "$$GOBIN" ]; then \
		echo "Installed c2 to $$GOBIN/c2"; \
	else \
		echo "Installed c2 to $$(go env GOPATH)/bin/c2"; \
	fi

test:
	@go test ./...

test-coverage:
	@go clean -testcache
	@go test -covermode=atomic -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out > coverage.txt
	@go tool cover -html=coverage.out -o coverage.html

fmt:
	@go fmt ./...

fmt-check:
	@unformatted=$$(gofmt -l $$(git ls-files '*.go')); \
	if [ -n "$$unformatted" ]; then \
		echo "Go files need formatting:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint:
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0 run --timeout=10m ./...

vet:
	@go vet ./...

tidy:
	@go mod tidy

clean:
	@rm -rf bin
	@rm -f coverage.out coverage.html coverage.txt
	@go clean
	@go clean -testcache

staticcheck:
	@go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...

check: fmt-check lint vet staticcheck test
