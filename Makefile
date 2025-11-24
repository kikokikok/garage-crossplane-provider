# ====================================================================================
# Setup Project

PROJECT_NAME := provider-garage
PROJECT_REPO := github.com/kikokikok/$(PROJECT_NAME)

# Terraform provider details
TERRAFORM_PROVIDER_SOURCE := deuxfleurs-org/garage
TERRAFORM_PROVIDER_REPO := https://github.com/deuxfleurs-org/garage-terraform-provider
TERRAFORM_PROVIDER_VERSION := 0.9.0
TERRAFORM_PROVIDER_DOWNLOAD_NAME := terraform-provider-garage
TERRAFORM_NATIVE_PROVIDER_BINARY := terraform-provider-garage_v0.9.0
TERRAFORM_DOCS_PATH := docs/resources

# ====================================================================================
# Common

# Set the shell to bash always
SHELL := /bin/bash

# Set the build platform
PLATFORMS ?= linux_amd64 linux_arm64

# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile.
-include build/makelib/common.mk

# ====================================================================================
# Setup Go

GO_REQUIRED_VERSION = 1.24
GOLANGCILINT_VERSION = 1.54.2

# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile.
-include build/makelib/golang.mk

# ====================================================================================
# Targets

# run `make help` to see the targets and options

# We want submodules to be set up the first time `make` is run.
# We manage the build/ folder and its Makefiles as a submodule.
# The first time `make` is run, the includes of build/*.mk files will
# all fail, and this target will be run. The next time, the default as defined
# by the includes will be run instead.
fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# Generate deepcopy, CRD, and controller code
.PHONY: generate
generate: go.generate

# Build the provider binary
.PHONY: build
build:
	@echo "Building provider..."
	@CGO_ENABLED=0 go build -o bin/provider cmd/provider/main.go

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/

# Run go mod tidy
.PHONY: tidy
tidy:
	@echo "Running go mod tidy..."
	@go mod tidy

# Run all linters
.PHONY: lint
lint:
	@echo "Running linters..."
	@golangci-lint run --timeout 5m

# Run unit tests
.PHONY: test
test:
	@echo "Running unit tests..."
	@go test -v ./...

# Setup development environment
.PHONY: dev
dev: build
	@echo "Development setup complete"

# Display help information
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  build      - Build the provider binary"
	@echo "  generate   - Generate deepcopy, CRD, and controller code"
	@echo "  clean      - Clean build artifacts"
	@echo "  tidy       - Run go mod tidy"
	@echo "  lint       - Run all linters"
	@echo "  test       - Run unit tests"
	@echo "  dev        - Setup development environment"
	@echo "  help       - Display this help message"

.PHONY: submodules
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# ====================================================================================
# Special Targets

.DEFAULT_GOAL := help

