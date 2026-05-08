.PHONY: help build run test test-unit test-integration test-coverage clean docker-build docker-run deps lint swagger

# Variables
BINARY_NAME=api-server
MAIN_PATH=cmd/api/main.go
DOCKER_IMAGE=edi-processing-api

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(MAIN_PATH)

run: ## Run the application
	@echo "Running application..."
	go run $(MAIN_PATH)

test: ## Run all tests
	@echo "Running all tests..."
	go test -v -race ./...

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v -race ./internal/...

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	@echo "Note: API server must be running (make run)"
	go test -v ./tests/...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	go test -v -race -count=1 ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 \
		-e MONGODB_URI=mongodb://host.docker.internal:27017 \
		-e REDIS_URI=host.docker.internal:6379 \
		$(DOCKER_IMAGE):latest

dev-services: ## Start development services (MongoDB and Redis)
	@echo "Starting MongoDB and Redis..."
	docker run -d --name mongodb -p 27017:27017 mongo:latest || docker start mongodb
	docker run -d --name redis -p 6379:6379 redis:latest || docker start redis

stop-services: ## Stop development services
	@echo "Stopping services..."
	docker stop mongodb redis || true

install: build ## Build and install the binary
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_PATH)

swagger: ## Generate/update Swagger documentation
	@echo "Generating Swagger documentation..."
	@if command -v swag > /dev/null; then \
		swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal; \
		echo "Swagger docs generated successfully"; \
	else \
		echo "⚠️  'swag' command not found"; \
		echo "To install swag, run: go install github.com/swaggo/swag/cmd/swag@latest"; \
		echo "Then add it to your PATH: export PATH=\$$PATH:\$$(go env GOPATH)/bin"; \
		echo ""; \
		echo "For now, using manual swagger.yaml in docs/ directory"; \
	fi

ci: deps lint test-coverage ## Run CI pipeline (lint and test with coverage)

all: clean deps fmt vet lint test build ## Run all checks and build
