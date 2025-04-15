# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

SHELL := /bin/bash
#
# App Info
#
VERSION            				?= $(shell cat VERSION)
GIT_HASH_SHORT     				:= $(shell git rev-parse --abbrev-ref HEAD)
VERSION_DEV_SUFFIX 				:= ${GIT_HASH_SHORT}

# Add an identifying suffix for `-dev` builds only.
# Release build versions are verified as unique by the CI build process.
ifneq ($(GIT_HASH_SHORT), main)
	ifeq ($(findstring -dev,$(VERSION)), -dev)
		VERSION := $(VERSION)-$(VERSION_DEV_SUFFIX)
	endif
endif



API_GW_COMPONENT_NAME 				?= api-gw
NAMESPACE 					?= default
CWD 						?= $(shell pwd)
COMPILER_TAG 					?= latest
LOG_LEVEL 					?= ERROR
DOCKER_REGISTRY 				?= registry-rs.edgeorchestration.intel.com
DOCKER_REPO ?= edge-orch/common
OS 						?= $(shell go env GOOS)
ARCH 						?= $(shell go env GOARCH)
DOCKER_BUILDER_PLATFORM 			?= linux/${ARCH}
ADMIN_DATAMODEL_DEFAULT_RUN_TAG 		?= latest
API_GW_DEFAULT_RUN_TAG 				?= latest
TAG 						?= latest
API_GW_DOCKER_IMAGE 				?= ${DOCKER_REGISTRY}/${DOCKER_REPO}/nexus/${API_GW_COMPONENT_NAME}:${TAG}
API_GW_RUN_DOCKER_IMAGE 			?= ${DOCKER_REGISTRY}/${DOCKER_REPO}/nexus/${API_GW_COMPONENT_NAME}:${API_GW_DEFAULT_RUN_TAG}
BUILD_DIR					:= build/bin
GOEXTRAFLAGS					:=-buildvcs=false
GOCMD						:= CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go
GOCMD_TEST                   := GOPRIVATE="github.com/open-edge-platform/*" go

# OEP Enviroment variable
IMG_NAME_OEP					:= nexus-api-gw
IMG_VERSION_OEP				?= ${VERSION}
DOCKER_REGISTRY_OEP         			?= registry-rs.edgeorchestration.intel.com
DOCKER_REPOSITORY_OEP       			?= edge-orch/common
API_GW_DOCKER_IMAGE_OEP 			?= ${DOCKER_REGISTRY_OEP}/${DOCKER_REPOSITORY_OEP}/${IMG_NAME_OEP}:${IMG_VERSION_OEP}

#
# Image Info
#
### adding this to test app init..
CI_COMMIT 					?= $(shell git rev-parse --verify --short=8 HEAD 2> /dev/null || echo "00000000")
IMAGE_TAG 					?= ${CI_COMMIT}
GIT_HEAD  					?= $(shell git rev-parse --verify HEAD 2> /dev/null || echo "0000000000000000")
IMAGE_REGISTRY 					?= gcr.io/nsx-sm/nexus/nexus-api-gateway

FUZZ_SECONDS                   ?= 24


# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

YAML_FILES := $(shell find . -path './venv_charts' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )

# Create the virtualenv with python tools installed
VENV_NAME = venv_apigw

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

.PHONY: all
all: create

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php


.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: check.prereq.git
check.prereq.git:
	$(info Checking access to required git repositories ...)
	@git ls-remote git@github.com:kubernetes/code-generator.git > /dev/null

.PHONY: check.prereq.go
check.prereq.go:
	$(info Checking if you golang requirements are met...)
	@go version > /dev/null

.PHONY: check.prereq.docker
check.prereq.docker:
	$(info Checking if you docker requirements are met...)
	@docker ps > /dev/null

.PHONY: check.prereq
check.prereq: check.prereq.git check.prereq.go check.prereq.docker

##@ Dev

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: go-lint hadolint yamllint ## Scan for all the golang source files, Dockerfile and yaml files

.PHONY: lint-ci
lint-ci: go-lint hadolint license

.PHONY: license
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

.PHONY: hadolint
hadolint: ## Scan Dockerfile with Hadolint
	hadolint Dockerfile

.PHONY: go-lint
go-lint: ## Scan golang source files with golangci-lint
	golangci-lint --version
	golangci-lint run --timeout=3m --config .golangci.yml ./...

.PHONY: yamllint
yamllint: ## Lint yaml files using yamllint
	yamllint --version
	yamllint -c yamllint_conf.yml -s $(YAML_FILES)

go_get:
	GOINSECURE=github.com/open-edge-platform GOPRIVATE=github.com/open-edge-platform go get . ;

.PHONY: go-tidy
go-tidy: ## Updates the Go dependencies and regenerate the go.sum file
	go mod tidy
	go mod download

create:
	mkdir -p ${BUILD_DIR}

.PHONY: go-build
go-build: go-tidy create ## Builds golang source files
	${GOCMD} build ${GOEXTRAFLAGS} -o ${BUILD_DIR}/${API_GW_COMPONENT_NAME}

##@ Test
.PHONY: init-unit-test
init-unit-test:
	if [[ ! -f test/bin/etcd || ! -f test/bin/kube-apiserver || ! -f test/bin/kubectl ]]; then \
		curl -SLo envtest-bins.tar.gz "https://go.kubebuilder.io/test-tools/1.24.2/$$(go env GOOS)/$$(go env GOARCH)"; \
        tar --strip-components=1 -xvf envtest-bins.tar.gz -C test; \
        rm envtest-bins.tar.gz; \
  	fi

.PHONY: race-unit-test
race-unit-test: init-unit-test
	${GOCMD_TEST} test -coverprofile ${BUILD_DIR}/coverage.out -coverpkg=$$(go list ./... | tr '\n' ,)  -covermode atomic ./...

.PHONY: test-fmt
test-fmt:
	test -z $$(goimports -w -l cmd pkg)

.PHONY: test
test: go-build race-unit-test  ## Runs unit test

clean-testdata:
	rm -rf ./pkg/apiremap/testdata
	rm -rf ./pkg/auth/authn/testdata
	rm -rf ./pkg/auth/authz/testdata

test-fuzz: clean-testdata
	$(GOCMD) test ./pkg/apiremap -fuzz FuzzTenancyAPIRemapping -fuzztime=${FUZZ_SECONDS}s
	$(GOCMD) test ./pkg/auth/authn -fuzz FuzzVerifyJWT -fuzztime=${FUZZ_SECONDS}s
	$(GOCMD) test ./pkg/auth/authz -fuzz FuzzVerifyAuthorization -fuzztime=${FUZZ_SECONDS}s
	$(GOCMD) test ./pkg/auth/authz -fuzz FuzzVerifyAuthzForProjects -fuzztime=${FUZZ_SECONDS}s
	$(GOCMD) test ./pkg/auth/authz -fuzz FuzzAuthzForProjectsMemberRole -fuzztime=${FUZZ_SECONDS}s

.PHONY: integration_test
integration_test:
	echo "Add your integration_tests for your app here!!!!..." ;
	echo "Sample Workflow can be.";
	echo "1. Create Environment";
	echo "2. Start built application in the Environment";
	echo "3. Start integration_tests with go test / gingko framework";

.PHONY: integration_test_cleanup
integration_test_cleanup:
	echo "Add your cleanup steps here!!!!....";
	echo "Possible steps you can do.";
	echo "1. Get logs of integration test as artifacts"
	echo "2. Get logs of components in clusters as artifacts for debugging"

.PHONY: teardown_environment
teardown_environment:
	echo "Add cluster cleanup step after integration_tests pass/fail here..";
	echo "Clear clusters created";

##@ Coverage
.PHONY: coverage
coverage: test ## Runs unit test coverage
	gocover-cobertura < ${BUILD_DIR}/coverage.out > ${BUILD_DIR}/coverage.xml
	${GOCMD_TEST} tool cover -html=${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/coverage.html
	${GOCMD_TEST} tool cover -func ${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/function_coverage.log

.PHONY: cleanbuildflag 
cleanbuildflag:
	@rm -f .gitconfig
	@rm -f .netrc
	@rm -rf .ssh

.PHONY: linkbuildflag 
linkbuildflag:
	touch .gitconfig .netrc
	mkdir -p .ssh
	cp ${HOME}/.gitconfig . || true
	cp ${HOME}/.netrc . || true
	cp -r ${HOME}/.ssh .ssh || true

##@ Build
#
# Usage: DOCKER_REGISTRY=<registry> TAG=<tag-value> make api-gw.build
# Tag and publish nexus api gw.
#
# Example: DOCKER_REGISTRY=822995803632.dkr.ecr.us-west-2.amazonaws.com TAG=letstest make api-gw.build
#
.PHONY: api-gw.build
api-gw.build: cleanbuildflag linkbuildflag
	docker build --build-arg API_GW_COMPONENT_NAME=${API_GW_COMPONENT_NAME} --build-arg HTTP_PROXY=${http_proxy} --build-arg HTTPS_PROXY=${https_proxy} --build-arg FTP_PROXY=${ftp_proxy} --build-arg SOCKS_PROXY=${socks_proxy} --build-arg NO_PROXY=${no_proxy} -t ${API_GW_DOCKER_IMAGE} -f Dockerfile .

.PHONY: buildall
buildall: api-gw.build

.PHONY: build
build: go-build api-gw.build-oep  ## Creates the docker container

.PHONY: api-gw.build-oep
api-gw.build-oep: cleanbuildflag linkbuildflag
	docker build --build-arg API_GW_COMPONENT_NAME=${API_GW_COMPONENT_NAME} --build-arg HTTP_PROXY=${http_proxy} --build-arg HTTPS_PROXY=${https_proxy} --build-arg FTP_PROXY=${ftp_proxy} --build-arg SOCKS_PROXY=${socks_proxy} --build-arg NO_PROXY=${no_proxy} -t ${API_GW_DOCKER_IMAGE_OEP} -f Dockerfile .

##@ Publish
.PHONY: api-gw.publish
api-gw.publish:
	docker push ${API_GW_DOCKER_IMAGE}

.PHONY: publishall
publishall: api-gw.publish

.PHONY: release
release: docker.publish.oep ## Publishes the built docker container to pre-defined docker container registry. This registry is set in a env variable (API_GW_DOCKER_IMAGE_OEP) in nexus-api-gateway/Makefile

.PHONY: docker.publish.oep
docker.publish.oep:## tag and push Docker image
	docker push ${API_GW_DOCKER_IMAGE_OEP}

.PHONY: image_scan
image_scan:
	flash docker scan image ${API_GW_DOCKER_IMAGE}
