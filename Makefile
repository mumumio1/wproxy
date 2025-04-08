.PHONY: build test bench clean run help all

BINARY=proxy
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build binary
	go build ${LDFLAGS} -o $(BINARY) ./cmd/proxy

test: ## Run tests
	go test -v -race -coverprofile=coverage.txt ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

clean: ## Clean artifacts
	rm -f $(BINARY) coverage.txt coverage.html

run: build ## Build and run with example config
	./$(BINARY) -config config.example.yaml

all: clean test build ## Clean, test and build

