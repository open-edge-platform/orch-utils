# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

.PHONY: build test lint clean docker-build ginkgo

# Variables
SERVICE_NAME := component-status
DOCKER_IMAGE := component-status
VERSION := $(shell cat ../VERSION 2>/dev/null || echo "dev")
GIT_REVISION := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go variables
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0

# Build output
BUILD_DIR := build
BINARY := $(BUILD_DIR)/$(SERVICE_NAME)

build:
	@echo "Building $(SERVICE_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build \
		-ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.Revision=$(GIT_REVISION)'" \
		-o $(BINARY) \
		./cmd/$(SERVICE_NAME)
	@echo "Build complete: $(BINARY)"

test: ginkgo ## Run all tests

ginkgo: ## Run ginkgo tests
	ginkgo run -v -r -p --race -randomize-all -randomize-suites

lint:
	@echo "Running linter..."
	golangci-lint run ./...
	@echo "Lint complete"

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
	@echo "Clean complete"

docker-build:
	@echo "Building Docker image..."
	docker build \
		--build-arg org_oci_version=$(VERSION) \
		--build-arg org_oci_revision=$(GIT_REVISION) \
		--build-arg org_oci_created=$(BUILD_DATE) \
		--build-arg org_oci_source=https://github.com/open-edge-platform/orch-utils \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		-f Dockerfile \
		..
	@echo "Docker build complete"
