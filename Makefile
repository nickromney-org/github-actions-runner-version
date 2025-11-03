.PHONY: build clean install test lint fmt help

# Binary name
BINARY_NAME=github-actions-runner-version
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOINSTALL=$(GOCMD) install

# Build targets
all: test build

build: ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	$(GOBUILD) ${LDFLAGS} -o bin/${BINARY_NAME} .
	@echo "✅ Build complete: bin/${BINARY_NAME}"

build-all: ## Build for all platforms
	@echo "Building for multiple platforms..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-arm64 .
	GOOS=linux GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) ${LDFLAGS} -o bin/${BINARY_NAME}-linux-arm64 .
	GOOS=windows GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o bin/${BINARY_NAME}-windows-amd64.exe .
	@echo "✅ Multi-platform build complete"

install: ## Install the binary to GOPATH/bin
	$(GOINSTALL) ${LDFLAGS} .

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf bin/

test: ## Run tests
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	$(GOCMD) tool cover -html=coverage.out

lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt: ## Format code
	$(GOCMD) fmt ./...
	$(GOCMD) mod tidy

deps: ## Download dependencies
	$(GOGET) -v ./...
	$(GOCMD) mod download

run: ## Run the application (example)
	$(GOBUILD) -o bin/${BINARY_NAME} .
	./bin/${BINARY_NAME} -c 2.327.1 -v

docker-build: ## Build Docker image
	docker build -t ${BINARY_NAME}:${VERSION} .

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
