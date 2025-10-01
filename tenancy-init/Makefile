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
IMG_NAME_OEP                  ?= tenancy-api-mapping
IMG_VERSION_OEP               ?= ${VERSION}
TENANCY_API_MAPPING_DOCKER_IMAGE_OEP ?= ${DOCKER_REGISTRY_OEP}/${DOCKER_REPOSITORY_OEP}/${IMG_NAME_OEP}:${IMG_VERSION_OEP}
OUT_DIR                       := out
REPO_DIR                      := repos
DOWNLOAD_DIR                  := downloads
API_MAPPING_CONFIG_DIR        := apimappingconfigcrs

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
all: lint build test ## Runs build, lint, test stages

# virtualenv for python tools
VENV_NAME = venv_apimapping
$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

### Lint targets ###
.PHONY: lint
lint: license go-lint hadolint yamllint  ## run all linters run as in CI

.PHONY: license
license: $(VENV_NAME) ## lint licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

.PHONY: go-lint
go-lint: ## lint all go code with golangci-lint
	golangci-lint --version
	golangci-lint run --timeout=3m --config .golangci.yml ./...

.PHONY: go-fix
go-fix:
	golangci-lint run --fix --config .golangci.yml

.PHONY: hadolint
hadolint: ## lint Dockerfile with hadolint
	hadolint Dockerfile

YAML_FILES := $(shell find . -path './repos' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
.PHONY: yamllint
yamllint: $(VENV_NAME) ## Lint all yaml files
	. ./$</bin/activate; set -u;\
  yamllint --version ;\
  yamllint -c yamllint_conf.yml -s $(YAML_FILES)

.PHONY: checkov
checkov: $(VENV_NAME) ## Perform checkov scans on generated APIs
	. ./$</bin/activate; set -u;\
  checkov -v ;\
  checkov -d openapispecs/generated/

### Docker targets ###
.PHONY: build
build: ## Runs build stage
	go mod vendor
	docker build -t ${TENANCY_API_MAPPING_DOCKER_IMAGE_OEP} . -f Dockerfile

.PHONY: release
release: docker.publish.oep

docker.publish.oep:
	@docker push ${TENANCY_API_MAPPING_DOCKER_IMAGE_OEP}

### npm/openapi lint targets ###

# Define a target to clean and rebuild dependencies
.PHONY: rebuild-deps
rebuild-deps:
	rm -rf node_modules
	npm install

# Define a target to install dependencies
.PHONY: install-deps
install-deps:
	npm install openapi-format @redocly/cli @redocly/openapi-cli

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

### targets to obtain APIs being mapped###
$(REPO_DIR):
	@mkdir -p $@

# Get list of directories to checkout repos, named per api mapping
REPO_LIST := $(notdir $(shell grep -l repoConf $(API_MAPPING_CONFIG_DIR)/* ))
REPOS := $(addprefix $(REPO_DIR)/, $(basename $(REPO_LIST)))

checkout-repos: $(REPOS) ## checkout all git repos with OpenAPI specs
	@echo "All repos checked out"
	@for repo in $(REPO_LIST); do \
    GIT_REF=$$(yq eval ".spec.repoConf.tag" "$(API_MAPPING_CONFIG_DIR)/$$repo" );\
    pushd "$(REPO_DIR)/$${repo%.*}" ;\
      git fetch -q ;\
      git switch --detach "$$GIT_REF" ;\
    popd ;\
	done

# individual repo clones, based on presence
$(REPO_DIR)/%:
	@REPO_URL=$$(yq eval ".spec.repoConf.url" "$(API_MAPPING_CONFIG_DIR)/$*.yaml" );\
  git clone -q "$$REPO_URL" $@

# Get list of downloaded API sources, named per api mapping
DOWNLOAD_APIS := $(shell grep -l downloadConf $(API_MAPPING_CONFIG_DIR)/* )

.PHONY: refresh-downloads
refresh-downloads: ## re-download OpenAPI specs
	@for dlapi in $(DOWNLOAD_APIS); do \
    DL_URL=$$(yq eval ".spec.downloadConf.url" "$$dlapi" ) ;\
    DL_VERSION=$$(yq eval ".spec.downloadConf.version" "$$dlapi" ) ;\
    DL_FILE="$$(yq eval ".metadata.name" $$dlapi)_$${DL_VERSION}.yaml";\
    curl -H "Accept: application/yaml" -o "$(DOWNLOAD_DIR)/$${DL_FILE}" "$${DL_URL}" ;\
    yq -i 'del(.servers)' "$(DOWNLOAD_DIR)/$${DL_FILE}" ;\
  done

### Test targets ###
.PHONY: test
test: checkout-repos setup-go-cache | $(OUT_DIR) ## Run go tests
	$(GOCMD) test -coverprofile $(OUT_DIR)/coverage.out -coverpkg=$$(go list ./... | grep -v "pkg/git/" | grep -v "pkg/config/" | grep -v "pkg/openapi/" | tr '\n' ,)  -covermode atomic ./...
	gocover-cobertura < $(OUT_DIR)/coverage.out > $(OUT_DIR)/coverage.xml
	$(GOCMD) tool cover -html=$(OUT_DIR)/coverage.out -o $(OUT_DIR)/coverage.html
	$(GOCMD) tool cover -func $(OUT_DIR)/coverage.out -o $(OUT_DIR)/function_coverage.log
	@echo "test run completed"

.PHONY: ginkgo
ginkgo: ## run ginkgo tests
	ginkgo run -v -r -p --race -randomize-all -randomize-suites

coverage: ## Run coverage
	echo  "See output of test"

### Build targets ###

# Define a target to generate the openapi spec and clean up
.PHONY: specgen ## Generate all MT OpenAPI Specs
specgen: generate-specs

.PHONY: generate-specs
generate-specs: setup-go-cache ## Generate openapi specs from apimapping
	@echo "---Running the spec generation..."
	@GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go run cmd/specgen/specgen.go
	@echo "---spec gen complete."
	./scripts/copyDatamodelOpenAPISpec.sh

# Define a target to convert the oas version of all the files
# and combine into one 'openapispecs/combined/combined_spec.yaml'
.PHONY: convert-and-combine
convert-and-combine: setup-go-cache
	@echo "---Running the conversion and combination script..."
	@GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go run cmd/convertandcombinespec/main.go

# Define a target to run the full workflow
.PHONY: gen-convert-combine-all
gen-convert-combine-all: install-deps checkout-repos specgen convert-and-combine

### Cleanup ###

# Define a target to clean the intermediate directories
.PHONY: clean-intermediate
clean-intermediate:
	rm -rf openapispecs/converted_specs

.PHONY: clean-submodules
clean-submodules: # clean up checked out repos
	rm -rf $(REPO_DIR)

.PHONY: clean-cache
clean-cache: ## Clean Go cache directories
	rm -rf /tmp/gocache /tmp/gomodcache

.PHONY: clean-all
clean-all: clean-submodules clean-intermediate clean-cache ## Cleanup all build files
	rm -rf $(VENV_NAME) $(OUT_DIR) vendor/

#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'

