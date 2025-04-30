# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

# Create the virtualenv with python tools installed
VENV_NAME = venv_apimapping

VERSION                       ?= $(shell cat VERSION)
GIT_HASH_SHORT     						:= $(shell git rev-parse --abbrev-ref HEAD)
VERSION_DEV_SUFFIX 						:= ${GIT_HASH_SHORT}

ifneq ($(GIT_HASH_SHORT), main)
	ifeq ($(findstring -dev,$(VERSION)), -dev)
		VERSION := $(VERSION)-$(VERSION_DEV_SUFFIX)
	endif
endif

DOCKER_REGISTRY_OEP	?= registry-rs.edgeorchestration.intel.com
DOCKER_REPOSITORY_OEP	?= edge-orch/common
IMG_NAME_OEP			?= tenancy-api-mapping
IMG_VERSION_OEP		?= ${VERSION}
TENANCY_API_MAPPING_DOCKER_IMAGE_OEP ?= ${DOCKER_REGISTRY_OEP}/${DOCKER_REPOSITORY_OEP}/${IMG_NAME_OEP}:${IMG_VERSION_OEP}
BUILD_DIR               := build

# Go command invocation
GOCMD                   := GOPRIVATE="github.com/open-edge-platform/*" go

# tag for the datamodel openapi spec copy
DM_REPO_TAG_VERSION := main

.PHONY: all
all: lint license build test ## Runs build, lint, test stages


.PHONY: build
build: ## Runs build stage
	go mod vendor
	@echo "---MAKEFILE BUILD---"
	@docker build -t ${TENANCY_API_MAPPING_DOCKER_IMAGE_OEP} . -f Dockerfile
	@echo "---END MAKEFILE Build---"

.PHONY: test
test: run-test clean-submodules   ## Run go tests

.PHONY: run-test
run-test:
	${GOCMD} test -coverprofile ${BUILD_DIR}/coverage.out -coverpkg=$$(go list ./... | grep -v "pkg/git/" | grep -v "pkg/config/" | grep -v "pkg/openapi/" | tr '\n' ,)  -covermode atomic ./...
	gocover-cobertura < ${BUILD_DIR}/coverage.out > ${BUILD_DIR}/coverage.xml
	${GOCMD} tool cover -html=${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/coverage.html
	${GOCMD} tool cover -func ${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/function_coverage.log
	@echo "test run completed"

coverage: ## Run Coverage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

.PHONY: lint
lint: clean-all go-lint hadolint yamllint  ## run all linters run as in CI

.PHONY: lint-ci
lint-ci: clean-all go-lint hadolint yamllint license

.PHONY: hadolint
hadolint: ## lint Dockerfile with hadolint
	hadolint Dockerfile

.PHONY: go-lint
go-lint: ## lint all go code with golangci-lint
	golangci-lint --version
	golangci-lint run --timeout=3m --config .golangci.yml ./...

.PHONY: license
license: $(VENV_NAME) ## lint licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

YAML_FILES := $(shell find . -path './gitsubmodules' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
.PHONY: yamllint
yamllint: ## Lint all yaml files
	yamllint --version
	yamllint -c yamllint_conf.yml -s $(YAML_FILES)

.PHONY: release
release: docker.publish.oep

docker.publish.oep:
	@echo "---MAKEFILE PUBLISH---"
	@docker push ${TENANCY_API_MAPPING_DOCKER_IMAGE_OEP}
	@echo "---END MAKEFILE PUBLISH---"

node_modules:
	npm ci

openapi-lint: node_modules
	npx redocly lint ./openapispecs/generated/*.yaml \
		--skip-rule=no-unused-components \
		--skip-rule=operation-summary \
		--skip-rule=operation-operationId \
		--skip-rule=operation-4xx-response \
		--skip-rule=info-license \
		--skip-rule=tag-description

#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'

# Define a target to generate the openapi spec and clean up
.PHONY: specgen ## Generate all MT OpenAPI Specs
specgen: generate-specs clean-submodules

# Define a target to generate the openapi spec for the apimappingconfigs
.PHONY: generate-specs
generate-specs:
	@echo "\n\n---Running the spec generation..."
	@go run cmd/specgen/specgen.go
	@echo "\n---spec gen complete."

# Define a target to clean git submodules
.PHONY: clean-submodules
clean-submodules:
	@echo "\n\n---Cleaning submodules..."
	@./scripts/remove_submodules.sh
	@echo "---submodules cleanup done."

# Define a target to clean and rebuild dependencies
.PHONY: rebuild-deps
rebuild-deps:
	@echo "\n\n---Cleaning and rebuilding dependencies..."
	@rm -rf node_modules
	@npm install
	@echo "---rebuilding dependencies done."

# Define a target to install dependencies
.PHONY: install-deps
install-deps:
	@echo "\n\n---Installing dependencies..."
	@npm install -g openapi-format @redocly/cli @redocly/openapi-cli
	sudo apt-get install jq
	pip install yq
	@echo "---Installation of dependencies done."

# Define a target to clean the intermediate directories
.PHONY: clean-intermediate
clean-intermediate:
	@echo "\n\n---Cleaning intermediate directories..."
	@rm -rf openapispecs/converted_specs
	@echo "---Cleanup of intermediate directories done."

# Target to copy the datamodel openapi spec
.PHONY: copy-and-preprocess-dm-oas
copy-and-preprocess-dm-oas:
	@echo "\n\n---Copy OAS from DM repo..."
	@DM_REPO_TAG_VERSION=$(DM_REPO_TAG_VERSION) ./scripts/copyDatamodelOpenAPISpec.sh
	@echo "---Copied datamodel OAS."

# Define a target to convert the oas version of all the files
# and combine into one 'openapispecs/combined/combined_spec.yaml'
.PHONY: convert-and-combine
convert-and-combine: copy-and-preprocess-dm-oas
	@echo "\n\n---Running the conversion and combination script..."
	@go run cmd/convertandcombinespec/main.go

# Define a target to clear all
.PHONY: clean-all
clean-all: clean-submodules clean-intermediate

# Define a target to run the full workflow
.PHONY: gen-convert-combine-all
gen-convert-combine-all: install-deps specgen convert-and-combine clean-intermediate
