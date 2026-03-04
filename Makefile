VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test lint vet fmt check clean

build:
	go build $(LDFLAGS) -o c2 ./cmd/c2

test:
	go test ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run

vet:
	go vet ./...

fmt:
	goimports -w .

check: fmt vet lint test
	@echo "All checks passed."

clean:
	rm -f c2
