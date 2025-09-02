# Pi Controller Build System
.PHONY: help build test clean install docker proto deps lint fmt vet

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Go build variables
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

# Directories
BUILD_DIR = build
PROTO_DIR = proto
DOCS_DIR = docs
SCRIPTS_DIR = scripts

# Binaries
CONTROLLER_BINARY = pi-controller
AGENT_BINARY = pi-agent
WEB_BINARY = pi-web

# Docker
DOCKER_REGISTRY ?= localhost:5000
DOCKER_TAG ?= $(VERSION)

# Default target
help: ## Show this help message
	@echo "Pi Controller Build System"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Dependencies
deps: ## Install Go dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Protocol buffers
proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@command -v protoc >/dev/null 2>&1 || { echo "protoc is required but not installed. Aborting." >&2; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

# Code quality
fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is required but not installed. Run 'make install-lint' first." >&2; exit 1; }
	golangci-lint run

install-lint: ## Install golangci-lint
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Testing
test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v -race -short -coverprofile=coverage-unit.out ./internal/services/... ./pkg/gpio/... ./internal/api/handlers/...

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	go test -v -race -run Integration -coverprofile=coverage-integration.out ./test/integration/...

test-security: ## Run security vulnerability tests
	@echo "Running security tests..."
	go test -v -race -run Security -coverprofile=coverage-security.out ./test/security/...

test-gpio: ## Run GPIO hardware simulation tests
	@echo "Running GPIO tests..."
	go test -v -race -run GPIO -coverprofile=coverage-gpio.out ./pkg/gpio/...

test-api: ## Run API endpoint tests
	@echo "Running API tests..."
	go test -v -race -run API -coverprofile=coverage-api.out ./internal/api/handlers/...

test-benchmarks: ## Run performance benchmarks
	@echo "Running performance benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./test/benchmarks/... ./internal/services/... ./pkg/gpio/...

test-security-verbose: ## Run detailed security vulnerability analysis
	@echo "Running comprehensive security analysis..."
	go test -v -race -run TestSecurity -coverprofile=coverage-security.out ./test/security/...
	@echo ""
	@echo "Security test results above identify critical vulnerabilities."
	@echo "See test output for detailed recommendations."

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-threshold: test ## Check if coverage meets minimum threshold
	@echo "Checking coverage threshold..."
	@go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//' > coverage.tmp
	@COVERAGE=$$(cat coverage.tmp); if [ "$$COVERAGE" -lt "80" ]; then echo "Coverage $$COVERAGE% is below 80% threshold"; exit 1; else echo "Coverage $$COVERAGE% meets 80% threshold"; fi
	@rm -f coverage.tmp

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

test-fuzz: ## Run fuzzing tests
	@echo "Running fuzzing tests..."
	go test -fuzz=. -fuzztime=30s ./...

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -race -timeout=30s ./...

test-all: test-unit test-integration test-security test-gpio test-api ## Run all test suites
	@echo "All test suites completed!"

test-comprehensive: test-all test-benchmarks test-security-verbose ## Run comprehensive test suite including benchmarks

# Build targets
build: build-controller build-agent ## Build all binaries

build-controller: ## Build pi-controller binary
	@echo "Building pi-controller for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(CONTROLLER_BINARY)-$(GOOS)-$(GOARCH) \
		./cmd/pi-controller

build-agent: ## Build pi-agent binary
	@echo "Building pi-agent for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(AGENT_BINARY)-$(GOOS)-$(GOARCH) \
		./cmd/pi-agent

# Cross-compilation targets
build-linux-amd64: ## Build for Linux AMD64
	@$(MAKE) build GOOS=linux GOARCH=amd64

build-linux-arm64: ## Build for Linux ARM64 (Raspberry Pi)
	@$(MAKE) build GOOS=linux GOARCH=arm64

build-linux-arm: ## Build for Linux ARM (Raspberry Pi 32-bit)
	@$(MAKE) build GOOS=linux GOARCH=arm GOARM=7

build-darwin-amd64: ## Build for macOS AMD64
	@$(MAKE) build GOOS=darwin GOARCH=amd64

build-darwin-arm64: ## Build for macOS ARM64 (Apple Silicon)
	@$(MAKE) build GOOS=darwin GOARCH=arm64

build-windows-amd64: ## Build for Windows AMD64
	@$(MAKE) build GOOS=windows GOARCH=amd64

build-all: ## Build for all supported platforms
	@echo "Building for all platforms..."
	@$(MAKE) build-linux-amd64
	@$(MAKE) build-linux-arm64
	@$(MAKE) build-linux-arm
	@$(MAKE) build-darwin-amd64
	@$(MAKE) build-darwin-arm64
	@$(MAKE) build-windows-amd64

# Installation
install: build-controller build-agent ## Install binaries to GOPATH/bin
	@echo "Installing binaries..."
	go install -ldflags "$(LDFLAGS)" ./cmd/pi-controller
	go install -ldflags "$(LDFLAGS)" ./cmd/pi-agent

# Docker targets
docker: docker-controller docker-agent ## Build all Docker images

docker-controller: ## Build pi-controller Docker image
	@echo "Building pi-controller Docker image..."
	docker build -t $(DOCKER_REGISTRY)/pi-controller:$(DOCKER_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-f docker/Dockerfile.controller .

docker-agent: ## Build pi-agent Docker image
	@echo "Building pi-agent Docker image..."
	docker build -t $(DOCKER_REGISTRY)/pi-agent:$(DOCKER_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-f docker/Dockerfile.agent .

docker-push: docker ## Push Docker images to registry
	@echo "Pushing Docker images..."
	docker push $(DOCKER_REGISTRY)/pi-controller:$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/pi-agent:$(DOCKER_TAG)

# Development
run-controller: ## Run pi-controller locally
	@echo "Running pi-controller..."
	go run -ldflags "$(LDFLAGS)" ./cmd/pi-controller

run-agent: ## Run pi-agent locally
	@echo "Running pi-agent..."
	go run -ldflags "$(LDFLAGS)" ./cmd/pi-agent

dev: ## Start development environment
	@echo "Starting development environment..."
	@$(MAKE) build-controller
	./$(BUILD_DIR)/$(CONTROLLER_BINARY)-$(GOOS)-$(GOARCH) --log-level debug --log-format text

# Database
db-migrate: ## Run database migrations
	@echo "Running database migrations..."
	go run ./cmd/pi-controller migrate up

db-migrate-up: ## Run pending database migrations
	@echo "Running pending database migrations..."
	go run ./cmd/pi-controller migrate up

db-migrate-down: ## Rollback the last database migration
	@echo "Rolling back last database migration..."
	go run ./cmd/pi-controller migrate down

db-migrate-status: ## Show database migration status
	@echo "Showing database migration status..."
	go run ./cmd/pi-controller migrate status

db-migrate-reset: ## Reset database - drops all tables and reapplies migrations (WARNING: destroys data)
	@echo "Resetting database - this will destroy all data!"
	go run ./cmd/pi-controller migrate reset --confirm

db-reset: ## Reset database by removing file and reapplying migrations (WARNING: destroys data)
	@echo "Resetting database..."
	rm -f data/pi-controller.db
	@$(MAKE) db-migrate-up

db-test-migrations: ## Test migration system with unit tests
	@echo "Testing migration system..."
	go test -v ./internal/migrations/...

# Configuration
config-example: ## Generate example configuration
	@echo "Generating example configuration..."
	@mkdir -p config
	@echo "# Pi Controller Configuration Example" > config/pi-controller.example.yaml
	@echo "" >> config/pi-controller.example.yaml
	@echo "app:" >> config/pi-controller.example.yaml
	@echo "  name: \"pi-controller\"" >> config/pi-controller.example.yaml
	@echo "  environment: \"development\"" >> config/pi-controller.example.yaml
	@echo "  data_dir: \"./data\"" >> config/pi-controller.example.yaml
	@echo "  debug: false" >> config/pi-controller.example.yaml
	@echo "" >> config/pi-controller.example.yaml
	@echo "database:" >> config/pi-controller.example.yaml
	@echo "  path: \"pi-controller.db\"" >> config/pi-controller.example.yaml
	@echo "  max_open_conns: 25" >> config/pi-controller.example.yaml
	@echo "  max_idle_conns: 5" >> config/pi-controller.example.yaml
	@echo "  conn_max_lifetime: \"5m\"" >> config/pi-controller.example.yaml
	@echo "  log_level: \"warn\"" >> config/pi-controller.example.yaml
	@echo "" >> config/pi-controller.example.yaml
	@echo "api:" >> config/pi-controller.example.yaml
	@echo "  host: \"0.0.0.0\"" >> config/pi-controller.example.yaml
	@echo "  port: 8080" >> config/pi-controller.example.yaml
	@echo "  read_timeout: \"30s\"" >> config/pi-controller.example.yaml
	@echo "  write_timeout: \"30s\"" >> config/pi-controller.example.yaml
	@echo "  cors_enabled: true" >> config/pi-controller.example.yaml
	@echo "  auth_enabled: false" >> config/pi-controller.example.yaml
	@echo "" >> config/pi-controller.example.yaml
	@echo "grpc:" >> config/pi-controller.example.yaml
	@echo "  host: \"0.0.0.0\"" >> config/pi-controller.example.yaml
	@echo "  port: 9090" >> config/pi-controller.example.yaml
	@echo "" >> config/pi-controller.example.yaml
	@echo "websocket:" >> config/pi-controller.example.yaml
	@echo "  host: \"0.0.0.0\"" >> config/pi-controller.example.yaml
	@echo "  port: 8081" >> config/pi-controller.example.yaml
	@echo "  path: \"/ws\"" >> config/pi-controller.example.yaml
	@echo "" >> config/pi-controller.example.yaml
	@echo "log:" >> config/pi-controller.example.yaml
	@echo "  level: \"info\"" >> config/pi-controller.example.yaml
	@echo "  format: \"json\"" >> config/pi-controller.example.yaml
	@echo "" >> config/pi-controller.example.yaml
	@echo "gpio:" >> config/pi-controller.example.yaml
	@echo "  enabled: true" >> config/pi-controller.example.yaml
	@echo "  mock_mode: false" >> config/pi-controller.example.yaml
	@echo "" >> config/pi-controller.example.yaml
	@echo "discovery:" >> config/pi-controller.example.yaml
	@echo "  enabled: true" >> config/pi-controller.example.yaml
	@echo "  method: \"mdns\"" >> config/pi-controller.example.yaml
	@echo "  port: 9091" >> config/pi-controller.example.yaml
	@echo "Example configuration written to config/pi-controller.example.yaml"

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

clean-all: clean ## Clean all generated files
	@echo "Cleaning all generated files..."
	go clean -cache
	go clean -modcache
	docker system prune -f

# Release
release: clean fmt vet test build-all ## Build release artifacts
	@echo "Creating release artifacts..."
	@mkdir -p $(BUILD_DIR)/release
	@for binary in $(BUILD_DIR)/*; do \
		if [ -f "$$binary" ]; then \
			cp "$$binary" $(BUILD_DIR)/release/; \
		fi \
	done
	@echo "Release artifacts created in $(BUILD_DIR)/release/"

# CI/CD helpers
ci-deps: ## Install CI dependencies
	@echo "Installing CI dependencies..."
	@$(MAKE) deps
	@$(MAKE) install-lint

ci-test: ## Run CI tests
	@echo "Running CI tests..."
	@$(MAKE) fmt
	@$(MAKE) vet
	@$(MAKE) lint
	@$(MAKE) test

ci-build: ## Run CI build
	@echo "Running CI build..."
	@$(MAKE) build-all

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@mkdir -p $(DOCS_DIR)
	@echo "Documentation generation not yet implemented"

# Version info
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"
	@echo "Go version: $$(go version)"

# Environment info
env: ## Show build environment
	@echo "Build environment:"
	@echo "  GOOS: $(GOOS)"
	@echo "  GOARCH: $(GOARCH)"
	@echo "  CGO_ENABLED: $(CGO_ENABLED)"
	@echo "  VERSION: $(VERSION)"
	@echo "  COMMIT: $(COMMIT)"
	@echo "  DATE: $(DATE)"
	@echo "  DOCKER_REGISTRY: $(DOCKER_REGISTRY)"
	@echo "  DOCKER_TAG: $(DOCKER_TAG)"