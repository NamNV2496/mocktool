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

tidy:
	@echo "$(COLOR_GREEN)Tidying go modules...$(COLOR_RESET)"
	$(GO) mod tidy
	$(GO) mod verify

install-tools:
	@echo "$(COLOR_GREEN)Installing development tools...$(COLOR_RESET)"
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	$(GO) install go.uber.org/mock/mockgen@latest
	$(GO) install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	$(GO) install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	@echo "$(COLOR_GREEN)Tools installed$(COLOR_RESET)"

proto:
	@echo "$(COLOR_GREEN)Generating protobuf files...$(COLOR_RESET)"
	protoc --go_out=pkg/generated --go-grpc_out=pkg/generated pkg/errorcustome/error_detail.proto
	@echo "$(COLOR_GREEN)Protobuf generation complete$(COLOR_RESET)"

mocks:
	@echo "$(COLOR_GREEN)Generating mocks...$(COLOR_RESET)"
	$(GO) generate ./internal/...
	@echo "$(COLOR_GREEN)Mock generation complete$(COLOR_RESET)"


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
