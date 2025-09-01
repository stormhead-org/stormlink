# Stormlink Backend Test Suite Makefile
# Provides convenient commands for running various test suites

.PHONY: help test test-unit test-integration test-performance test-e2e test-all
.PHONY: test-coverage test-benchmark test-race test-short
.PHONY: setup-test-env cleanup-test-env
.PHONY: lint fmt vet check
.PHONY: build run
.PHONY: docker-test docker-build docker-clean
.PHONY: deps deps-update deps-tidy

# Default target
.DEFAULT_GOAL := help

# Variables
GO_VERSION := 1.21
PROJECT_NAME := stormlink
COVERAGE_DIR := ./coverage
TEST_TIMEOUT := 30m
PARALLEL_TESTS := 4

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
WHITE := \033[0;37m
NC := \033[0m # No Color

##@ Help

help: ## Display this help
	@echo "$(CYAN)Stormlink Backend Test Suite$(NC)"
	@echo "$(YELLOW)Available commands:$(NC)\n"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Testing

test: ## Run all tests (unit + integration)
	@echo "$(BLUE)🚀 Running all tests...$(NC)"
	@go test ./tests/unit/... ./tests/integration/... -v -timeout $(TEST_TIMEOUT) -parallel $(PARALLEL_TESTS)

test-unit: ## Run unit tests only
	@echo "$(GREEN)📋 Running unit tests...$(NC)"
	@go test ./tests/unit/... -v -timeout 10m -parallel $(PARALLEL_TESTS)

test-integration: ## Run integration tests only
	@echo "$(PURPLE)🔗 Running integration tests...$(NC)"
	@go test ./tests/integration/... -v -timeout 20m -parallel 2

test-performance: ## Run performance tests
	@echo "$(YELLOW)⚡ Running performance tests...$(NC)"
	@go test ./tests/performance/... -v -timeout 45m -parallel 1

test-e2e: ## Run end-to-end tests
	@echo "$(CYAN)🌐 Running E2E tests...$(NC)"
	@go test ./tests/integration/e2e_test.go -v -timeout 30m

test-all: ## Run all test suites including performance
	@echo "$(BLUE)🎯 Running complete test suite...$(NC)"
	@$(MAKE) test-unit
	@$(MAKE) test-integration
	@$(MAKE) test-performance
	@$(MAKE) test-e2e

test-short: ## Run tests in short mode (skip slow tests)
	@echo "$(GREEN)⚡ Running tests in short mode...$(NC)"
	@go test ./tests/unit/... ./tests/integration/... -short -v -timeout 10m

test-race: ## Run tests with race detection
	@echo "$(RED)🏃 Running tests with race detection...$(NC)"
	@go test ./tests/unit/... ./tests/integration/... -race -v -timeout 15m

##@ Coverage

test-coverage: ## Run tests with coverage report
	@echo "$(CYAN)📊 Running tests with coverage...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@go test ./tests/unit/... -coverprofile=$(COVERAGE_DIR)/unit.out -covermode=atomic
	@go test ./tests/integration/... -coverprofile=$(COVERAGE_DIR)/integration.out -covermode=atomic
	@go run ./tools/merge-coverage.go $(COVERAGE_DIR)/unit.out $(COVERAGE_DIR)/integration.out > $(COVERAGE_DIR)/coverage.out
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1
	@echo "$(GREEN)✅ Coverage report generated: $(COVERAGE_DIR)/coverage.html$(NC)"

coverage-html: test-coverage ## Generate HTML coverage report
	@echo "$(BLUE)🌐 Opening coverage report in browser...$(NC)"
	@which open >/dev/null && open $(COVERAGE_DIR)/coverage.html || echo "Please open $(COVERAGE_DIR)/coverage.html manually"

coverage-summary: ## Show coverage summary
	@echo "$(CYAN)📈 Coverage Summary:$(NC)"
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out

##@ Benchmarks

test-benchmark: ## Run benchmark tests
	@echo "$(YELLOW)📊 Running benchmark tests...$(NC)"
	@go test ./tests/performance/... -bench=. -benchmem -run=^$

benchmark-compare: ## Run benchmarks and compare with previous results
	@echo "$(PURPLE)📊 Running benchmark comparison...$(NC)"
	@go test ./tests/performance/... -bench=. -benchmem -run=^$ > benchmarks.new
	@test -f benchmarks.old && benchstat benchmarks.old benchmarks.new || echo "No previous benchmarks found"
	@cp benchmarks.new benchmarks.old

##@ Code Quality

lint: ## Run linter
	@echo "$(BLUE)🔍 Running linter...$(NC)"
	@which golangci-lint >/dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run ./...

fmt: ## Format code
	@echo "$(GREEN)✨ Formatting code...$(NC)"
	@go fmt ./...
	@goimports -w .

vet: ## Run go vet
	@echo "$(YELLOW)🔍 Running go vet...$(NC)"
	@go vet ./...

check: fmt vet lint ## Run all code quality checks
	@echo "$(GREEN)✅ All code quality checks passed$(NC)"

##@ Dependencies

deps: ## Download dependencies
	@echo "$(BLUE)📦 Downloading dependencies...$(NC)"
	@go mod download

deps-update: ## Update dependencies
	@echo "$(YELLOW)🔄 Updating dependencies...$(NC)"
	@go get -u ./...
	@go mod tidy

deps-tidy: ## Clean up dependencies
	@echo "$(GREEN)🧹 Tidying dependencies...$(NC)"
	@go mod tidy

##@ Build & Run

build: ## Build the application
	@echo "$(BLUE)🔨 Building application...$(NC)"
	@go build -o bin/stormlink ./server.go

build-race: ## Build with race detection
	@echo "$(RED)🔨 Building with race detection...$(NC)"
	@go build -race -o bin/stormlink-race ./server.go

run: build ## Build and run the application
	@echo "$(GREEN)🚀 Starting application...$(NC)"
	@./bin/stormlink

##@ Database

migrate-up: ## Run database migrations
	@echo "$(BLUE)🗄️ Running database migrations...$(NC)"
	@go run ./server/cmd/migrate/main.go up

migrate-down: ## Rollback database migrations
	@echo "$(YELLOW)🗄️ Rolling back database migrations...$(NC)"
	@go run ./server/cmd/migrate/main.go down

migrate-reset: ## Reset database migrations
	@echo "$(RED)🗄️ Resetting database migrations...$(NC)"
	@go run ./server/cmd/migrate/main.go reset

##@ Docker

docker-build: ## Build Docker image
	@echo "$(BLUE)🐳 Building Docker image...$(NC)"
	@docker build -t $(PROJECT_NAME):latest .

docker-test: ## Run tests in Docker container
	@echo "$(PURPLE)🐳 Running tests in Docker...$(NC)"
	@docker run --rm -v $(PWD):/app -w /app golang:$(GO_VERSION) make test

docker-clean: ## Clean Docker images and containers
	@echo "$(RED)🧹 Cleaning Docker resources...$(NC)"
	@docker system prune -f

##@ Test Environment

setup-test-env: ## Setup test environment
	@echo "$(CYAN)🛠️ Setting up test environment...$(NC)"
	@docker-compose -f docker-compose.test.yml up -d
	@echo "$(GREEN)✅ Test environment ready$(NC)"

cleanup-test-env: ## Cleanup test environment
	@echo "$(YELLOW)🧹 Cleaning up test environment...$(NC)"
	@docker-compose -f docker-compose.test.yml down -v
	@echo "$(GREEN)✅ Test environment cleaned$(NC)"

test-with-env: setup-test-env test cleanup-test-env ## Run tests with fresh environment

##@ CI/CD

ci-test: ## Run tests in CI environment
	@echo "$(BLUE)🤖 Running CI tests...$(NC)"
	@go test ./tests/unit/... ./tests/integration/... -v -timeout $(TEST_TIMEOUT) -coverprofile=coverage.out
	@go tool cover -func=coverage.out

ci-build: ## Build for CI
	@echo "$(BLUE)🤖 Building for CI...$(NC)"
	@go build -v ./...

ci-lint: ## Run linting for CI
	@echo "$(BLUE)🤖 Running CI linting...$(NC)"
	@golangci-lint run --timeout=5m ./...

ci: ci-lint ci-build ci-test ## Run all CI checks

##@ Utilities

clean: ## Clean build artifacts
	@echo "$(RED)🧹 Cleaning build artifacts...$(NC)"
	@rm -rf bin/
	@rm -rf $(COVERAGE_DIR)/
	@rm -f benchmarks.old benchmarks.new
	@go clean -cache -testcache -modcache

install-tools: ## Install development tools
	@echo "$(BLUE)🔧 Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install golang.org/x/perf/cmd/benchstat@latest

stats: ## Show project statistics
	@echo "$(CYAN)📊 Project Statistics:$(NC)"
	@echo "Go files: $$(find . -name '*.go' | grep -v vendor | wc -l)"
	@echo "Test files: $$(find . -name '*_test.go' | wc -l)"
	@echo "Lines of code: $$(find . -name '*.go' | grep -v vendor | xargs wc -l | tail -1)"
	@echo "Dependencies: $$(go list -m all | wc -l)"

##@ Advanced Testing

test-stress: ## Run stress tests
	@echo "$(RED)💪 Running stress tests...$(NC)"
	@go test ./tests/performance/... -run=TestStress -timeout=10m -v

test-fuzz: ## Run fuzz tests
	@echo "$(PURPLE)🎲 Running fuzz tests...$(NC)"
	@go test ./tests/unit/... -fuzz=. -fuzztime=30s

test-memory: ## Run tests with memory profiling
	@echo "$(YELLOW)🧠 Running tests with memory profiling...$(NC)"
	@go test ./tests/performance/... -memprofile=mem.prof -bench=. -benchmem

test-cpu: ## Run tests with CPU profiling
	@echo "$(CYAN)⚡ Running tests with CPU profiling...$(NC)"
	@go test ./tests/performance/... -cpuprofile=cpu.prof -bench=.

profile-analyze: ## Analyze performance profiles
	@echo "$(BLUE)📊 Analyzing performance profiles...$(NC)"
	@test -f cpu.prof && go tool pprof cpu.prof
	@test -f mem.prof && go tool pprof mem.prof

##@ Documentation

docs: ## Generate documentation
	@echo "$(BLUE)📚 Generating documentation...$(NC)"
	@godoc -http=:6060
	@echo "$(GREEN)Documentation available at http://localhost:6060$(NC)"

##@ Quick Commands

quick-test: test-unit ## Quick test (unit tests only)
	@echo "$(GREEN)✅ Quick test completed$(NC)"

full-test: test-coverage benchmark-compare ## Full test suite with coverage and benchmarks
	@echo "$(GREEN)🏆 Full test suite completed$(NC)"

dev-check: fmt vet test-unit ## Development checks (format, vet, unit tests)
	@echo "$(GREEN)✅ Development checks completed$(NC)"

pre-commit: clean dev-check lint ## Pre-commit checks
	@echo "$(GREEN)🚀 Ready to commit$(NC)"

##@ Examples

example-unit: ## Run example unit test
	@echo "$(CYAN)📋 Running example unit test...$(NC)"
	@go test ./tests/unit/user_usecase_test.go -v -run=TestUserUsecase_GetUserByID

example-integration: ## Run example integration test
	@echo "$(PURPLE)🔗 Running example integration test...$(NC)"
	@go test ./tests/integration/user_integration_test.go -v -run=TestUserWorkflow

example-benchmark: ## Run example benchmark
	@echo "$(YELLOW)📊 Running example benchmark...$(NC)"
	@go test ./tests/performance/system_performance_test.go -bench=BenchmarkUserRetrieval -benchmem

##@ Environment Info

env-info: ## Show environment information
	@echo "$(CYAN)🔍 Environment Information:$(NC)"
	@echo "Go version: $$(go version)"
	@echo "GOPATH: $$(go env GOPATH)"
	@echo "GOROOT: $$(go env GOROOT)"
	@echo "GOOS: $$(go env GOOS)"
	@echo "GOARCH: $$(go env GOARCH)"
	@echo "CGO_ENABLED: $$(go env CGO_ENABLED)"
	@echo "Current directory: $$(pwd)"

##@ Debugging

debug-test: ## Run tests with debugging info
	@echo "$(RED)🐛 Running tests with debugging...$(NC)"
	@go test ./tests/unit/... -v -x

test-verbose: ## Run tests with maximum verbosity
	@echo "$(BLUE)🔊 Running tests with maximum verbosity...$(NC)"
	@go test ./tests/unit/... -v -x -count=1

test-failfast: ## Run tests and stop on first failure
	@echo "$(RED)🛑 Running tests with fail-fast...$(NC)"
	@go test ./tests/unit/... -v -failfast
