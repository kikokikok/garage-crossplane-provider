# Makefile for provider-garage

# Project
PROJECT_NAME := provider-garage
PROJECT_REPO := github.com/kikokikok/$(PROJECT_NAME)

# Build
PLATFORMS ?= linux_amd64 linux_arm64
GO_BUILD_FLAGS := -v

# Directories
BIN_DIR := bin
BUILD_DIR := build

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build provider binary
	@echo "Building $(PROJECT_NAME)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/provider cmd/provider/main.go

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(BUILD_DIR)

.PHONY: tidy
tidy: ## Run go mod tidy
	@echo "Running go mod tidy..."
	@go mod tidy

.PHONY: fmt
fmt: ## Run go fmt
	@echo "Running go fmt..."
	@go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

.PHONY: generate
generate: ## Generate code (CRDs, deepcopy, etc)
	@echo "Generating code..."
	@go generate ./...

.DEFAULT_GOAL := help
