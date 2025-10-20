# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# safe shell invocations
SHELL   := bash -eu -o pipefail

# default goal to show help
.DEFAULT_GOAL := help

VERSION                       ?= $(shell cat VERSION)
GIT_HASH_SHORT                := $(shell git rev-parse --abbrev-ref HEAD)
VERSION_DEV_SUFFIX            := ${GIT_HASH_SHORT}

ifneq ($(GIT_HASH_SHORT), main)
	ifeq ($(findstring -dev,$(VERSION)), -dev)
		VERSION := $(VERSION)-$(VERSION_DEV_SUFFIX)
	endif
endif

DOCKER_REGISTRY_OEP           ?= registry-rs.edgeorchestration.intel.com
DOCKER_REPOSITORY_OEP         ?= edge-orch/common
IMG_NAME_OEP                  ?= tenancy-init
IMG_VERSION_OEP               ?= ${VERSION}
TENANCY_INIT_DOCKER_IMAGE_OEP ?= ${DOCKER_REGISTRY_OEP}/${DOCKER_REPOSITORY_OEP}/${IMG_NAME_OEP}:${IMG_VERSION_OEP}
OUT_DIR                       := out

# Go command invocation with proper cache directories
GOCMD                         := GOPRIVATE="github.com/open-edge-platform/*" GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go

# directory creation
$(OUT_DIR):
	@mkdir -p $@

# Ensure Go cache directories exist with proper permissions
.PHONY: setup-go-cache
setup-go-cache:
	@mkdir -p /tmp/gocache /tmp/gomodcache
	@chmod 755 /tmp/gocache /tmp/gomodcache

.PHONY: all
all: lint build ## Runs build, lint, test stages

### Lint targets ###
.PHONY: lint
lint: go-lint

.PHONY: go-lint
go-lint: ## lint all go code with golangci-lint
	golangci-lint --version
	golangci-lint run --timeout=3m --config .golangci.yml ./...

.PHONY: go-fix
go-fix:
	golangci-lint run --fix --config .golangci.yml

### Docker targets ###
.PHONY: build
build: ## Runs build stage
	go mod vendor
	docker build -t ${TENANCY_INIT_DOCKER_IMAGE_OEP} . -f Dockerfile

.PHONY: release
release: docker.publish.oep

docker.publish.oep:
	@docker push ${TENANCY_INIT_DOCKER_IMAGE_OEP}


#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'

