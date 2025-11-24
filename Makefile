# Makefile for provider-garage

# Project
PROJECT_NAME := provider-garage
PROJECT_REPO := github.com/kikokikok/$(PROJECT_NAME)

# Image
REGISTRY ?= ghcr.io
IMAGE_NAME ?= kikokikok/provider-garage
IMG ?= $(REGISTRY)/$(IMAGE_NAME):latest

# Build
PLATFORMS ?= linux_amd64 linux_arm64
GO_BUILD_FLAGS := -v

# Directories
BIN_DIR := bin
BUILD_DIR := build
PACKAGE_DIR := package

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: build
build: ## Build provider binary
	@echo "Building $(PROJECT_NAME)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/provider cmd/provider/main.go

.PHONY: run
run: generate fmt vet ## Run the provider locally
	go run cmd/provider/main.go

.PHONY: test
test: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires Kind cluster)
	@echo "Running integration tests..."
	@go test -v -tags=integration ./test/integration/...

##@ Code Quality

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(BUILD_DIR) $(PACKAGE_DIR)/crds coverage.out

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

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m

##@ Code Generation

.PHONY: generate
generate: ## Generate code (CRDs, deepcopy, etc)
	@echo "Generating code..."
	@go generate ./...
	@go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=hack/boilerplate.go.txt paths=./apis/...

.PHONY: generate-crds
generate-crds: ## Generate CRDs
	@echo "Generating CRDs..."
	@mkdir -p $(PACKAGE_DIR)/crds
	@go run sigs.k8s.io/controller-tools/cmd/controller-gen \
		crd:crdVersions=v1 \
		paths="./apis/..." \
		output:crd:artifacts:config=$(PACKAGE_DIR)/crds

##@ Docker

.PHONY: docker-build
docker-build: ## Build docker image
	@echo "Building Docker image $(IMG)..."
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push: ## Push docker image
	@echo "Pushing Docker image $(IMG)..."
	docker push $(IMG)

.PHONY: docker-build-push
docker-build-push: docker-build docker-push ## Build and push docker image

##@ Deployment

.PHONY: install-crds
install-crds: generate-crds ## Install CRDs into the cluster
	@echo "Installing CRDs..."
	kubectl apply -f $(PACKAGE_DIR)/crds/

.PHONY: uninstall-crds
uninstall-crds: ## Uninstall CRDs from the cluster
	@echo "Uninstalling CRDs..."
	kubectl delete -f $(PACKAGE_DIR)/crds/ --ignore-not-found=true

.DEFAULT_GOAL := help
