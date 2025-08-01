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
	docker build --load -t ${OPENAPI_GENERATOR_DOCKER_IMAGE} -f openapi-generator/Dockerfile .

.PHONY: ginkgo
## Note: should fix/enable these tests
ginkgo: ## run ginkgo tests
	ginkgo run -v -r -p --race -randomize-all -randomize-suites \
  --skip-package="compiler/example/tests,compiler/pkg/generator,compiler/pkg/openapi_generator,compiler/pkg/parser,compiler/pkg/parser/rest,compiler/pkg/preparser,gqlgen/api,gqlgen/codegen/templates,gqlgen/codegen/testserver/followschema,gqlgen/codegen/testserver/singlefile,gqlgen/internal/code,gqlgen/internal/rewrite,gqlgen/graphql/handler/transport,gqlgen/plugin/federation,gqlgen/plugin/modelgen,gqlgen/plugin/resolvergen,kube-openapi/pkg/idl,kube-openapi/pkg/util/proto,kube-openapi/pkg/util/proto/validation,kube-openapi/test/integration,install-validator/pkg/dir" --timeout=10m

compiler-docker-list:
	@make -C compiler docker-list

docker-list: compiler-docker-list ## Print names of docker container images
	@echo "  nexus/$(OPENAPI_GENERATOR_COMPONENT_NAME):"
	@echo "    name: '$(OPENAPI_GENERATOR_DOCKER_IMAGE)'"
	@echo "    version: '$(VERSION)'"
	@echo "    gitTagPrefix: 'nexus/$(VERSION)'"
	@echo "    buildTarget: 'docker-build-nexus/openapi-generator'"

#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'
