# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

### configuration ###
# safe shell invocations
SHELL           := bash -eu -o pipefail

# default goal to show help
.DEFAULT_GOAL   := help

# default configuration variables
VERSION         ?= $(shell cat VERSION)
ARCH            ?= $(shell go env GOARCH)
DOCKER_REGISTRY ?=

### Docker build targets ###
.PHONY: compiler.builder
compiler.builder:
	cd compiler; DOCKER_REGISTRY=${DOCKER_REGISTRY} BUILDER_VERSION=${VERSION} make docker.builder

.PHONY: compiler.builder.publish
compiler.builder.publish:
	cd compiler; DOCKER_REGISTRY=${DOCKER_REGISTRY} BUILDER_VERSION=${VERSION} make docker.builder.publish

.PHONY: compiler.build
compiler.build:
	cd compiler; DOCKER_REGISTRY=${DOCKER_REGISTRY} BUILDER_VERSION=${VERSION} VERSION=${VERSION} make docker

.PHONY: compiler.build.publish
compiler.build.publish:
	cd compiler; DOCKER_REGISTRY=${DOCKER_REGISTRY} BUILDER_VERSION=${VERSION} VERSION=${VERSION} make docker.publish

OPENAPI_GENERATOR_COMPONENT_NAME ?= openapi-generator
OPENAPI_GENERATOR_DOCKER_IMAGE   ?= ${DOCKER_REGISTRY}nexus/${OPENAPI_GENERATOR_COMPONENT_NAME}:${VERSION}

.PHONY: openapi.generator.docker
openapi.generator.docker:
	docker build --no-cache -t ${OPENAPI_GENERATOR_DOCKER_IMAGE} -f openapi-generator/Dockerfile .

#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'
