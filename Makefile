.PHONY: help build run test lint fmt vet docker-up docker-down docker-restart clean install-tools proto

# Variables
APP_NAME := mocktool
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-s -w"
DOCKER_COMPOSE := docker compose

# Colors for output
COLOR_RESET := \033[0m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "$(COLOR_BLUE)Available commands:$(COLOR_RESET)"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the application binary
build:
	@echo "$(COLOR_GREEN)Building $(APP_NAME)...$(COLOR_RESET)"
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(APP_NAME) .
	@echo "$(COLOR_GREEN)Build complete: bin/$(APP_NAME)$(COLOR_RESET)"

## run: Run the application
run:
	@echo "$(COLOR_GREEN)Starting $(APP_NAME)...$(COLOR_RESET)"
	$(GO) run . service

## test: Run all tests
test:
	@echo "$(COLOR_GREEN)Running tests...$(COLOR_RESET)"
	$(GO) test $(GOFLAGS) ./... -race -coverprofile=coverage.out
	@echo "$(COLOR_GREEN)Tests complete$(COLOR_RESET)"

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "$(COLOR_GREEN)Generating coverage report...$(COLOR_RESET)"
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)Coverage report generated: coverage.html$(COLOR_RESET)"

## lint: Run linters
lint:
	@echo "$(COLOR_GREEN)Running linters...$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(COLOR_YELLOW)golangci-lint not found. Run 'make install-tools' first$(COLOR_RESET)"; \
		exit 1; \
	fi

## fmt: Format code
fmt:
	@echo "$(COLOR_GREEN)Formatting code...$(COLOR_RESET)"
	$(GO) fmt ./...
	gofmt -s -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi
	@echo "$(COLOR_GREEN)Code formatted$(COLOR_RESET)"

## vet: Run go vet
vet:
	@echo "$(COLOR_GREEN)Running go vet...$(COLOR_RESET)"
	$(GO) vet ./...

## tidy: Tidy go modules
tidy:
	@echo "$(COLOR_GREEN)Tidying go modules...$(COLOR_RESET)"
	$(GO) mod tidy
	$(GO) mod verify

## docker-up: Start docker containers
docker-up:
	@echo "$(COLOR_GREEN)Starting Docker containers...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(COLOR_GREEN)Docker containers started$(COLOR_RESET)"

## docker-down: Stop docker containers
docker-down:
	@echo "$(COLOR_YELLOW)Stopping Docker containers...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) down

## docker-restart: Restart docker containers
docker-restart: docker-down docker-up

## docker-logs: View docker logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## clean: Clean build artifacts and cache
clean:
	@echo "$(COLOR_YELLOW)Cleaning...$(COLOR_RESET)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GO) clean -cache -testcache -modcache
	@echo "$(COLOR_GREEN)Clean complete$(COLOR_RESET)"

## install-tools: Install development tools
install-tools:
	@echo "$(COLOR_GREEN)Installing development tools...$(COLOR_RESET)"
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	$(GO) install go.uber.org/mock/mockgen@latest
	@echo "$(COLOR_GREEN)Tools installed$(COLOR_RESET)"

## proto: Generate protobuf files
proto:
	@echo "$(COLOR_GREEN)Generating protobuf files...$(COLOR_RESET)"
	protoc --go_out=pkg/generated --go-grpc_out=pkg/generated pkg/errorcustome/error_detail.proto
	@echo "$(COLOR_GREEN)Protobuf generation complete$(COLOR_RESET)"

## mocks: Generate mock files using go generate
mocks:
	@echo "$(COLOR_GREEN)Generating mocks...$(COLOR_RESET)"
# 	@mkdir -p mocks
	$(GO) generate ./internal/...
	@echo "$(COLOR_GREEN)Mock generation complete$(COLOR_RESET)"

## dev: Start development environment (docker + app)
dev: docker-up
	@echo "$(COLOR_GREEN)Starting development environment...$(COLOR_RESET)"
	@sleep 2
	$(MAKE) run

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "$(COLOR_GREEN)All checks passed!$(COLOR_RESET)"

## deps: Download dependencies
deps:
	@echo "$(COLOR_GREEN)Downloading dependencies...$(COLOR_RESET)"
	$(GO) mod download
	@echo "$(COLOR_GREEN)Dependencies downloaded$(COLOR_RESET)"

## web: Open web interface
web:
	@echo "$(COLOR_GREEN)Opening web interface...$(COLOR_RESET)"
	@open web/index.html || xdg-open web/index.html || start web/index.html

## example-http: Run HTTP example
example-http:
	@echo "$(COLOR_GREEN)Starting HTTP example...$(COLOR_RESET)"
	$(GO) run ./example/http/main.go

## example-grpc: Run gRPC example
example-grpc:
	@echo "$(COLOR_GREEN)Starting gRPC example...$(COLOR_RESET)"
	$(GO) run ./example/grpc/main.go

## docker-build: Build Docker image
docker-build:
	@echo "$(COLOR_GREEN)Building Docker image...$(COLOR_RESET)"
	docker build -t $(APP_NAME):latest .
	@echo "$(COLOR_GREEN)Docker image built: $(APP_NAME):latest$(COLOR_RESET)"
