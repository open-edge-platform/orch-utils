# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# App Info
#
GIT_HASH_SHORT     						:= $(shell git rev-parse --abbrev-ref HEAD)
VERSION_DEV_SUFFIX 						:= ${GIT_HASH_SHORT}
TENANCY_MANAGER_COMPONENT_NAME          ?= tenancy-manager
BUILD_DIR      := build/bin

# Project variables
PROJECT_NAME := ${TENANCY_MANAGER_COMPONENT_NAME}
BINARY_NAME  := ${TENANCY_MANAGER_COMPONENT_NAME}

# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
# Hardcode 0 here, to avoid changing API version right before. We should get the major from the VERSION file via  $(shell cut -c 1 VERSION)
VERSION_MAJOR := 0
IMG_NAME      := ${PROJECT_NAME}
IMG_VERSION   ?= ${VERSION}
GIT_COMMIT    ?= $(shell git rev-parse HEAD)


# Docker variables
DOCKER_ENV              := DOCKER_BUILDKIT=1
DOCKER_REGISTRY         ?= registry-rs.edgeorchestration.intel.com
DOCKER_REPOSITORY       ?= edge-orch/common
DOCKER_TAG              := ${DOCKER_REGISTRY}/${DOCKER_REPOSITORY}/${IMG_NAME}
DOCKER_LABEL_REPO_URL   ?= $(shell git remote get-url $(shell git remote | head -n 1))
DOCKER_LABEL_VERSION    ?= ${IMG_VERSION}
DOCKER_LABEL_REVISION   ?= ${GIT_COMMIT}
DOCKER_LABEL_BUILD_DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

# Go command invocation
GOCMD                      := GOPRIVATE="github.com/open-edge-platform/*" go

# Misc variables
# REUSE-IgnoreStart
COPYRIGHT_BANNER := '1i---\n\# SPDX-FileCopyrightText: (C) 2022 Intel Corporation\n\# SPDX-License-Identifier: LicenseRef-Intel\n\# Generated file do not edit !!!'
# REUSE-IgnoreEnd


#-trimpath: Remove all file system paths from the resulting executable.
#-mod=readonly: Ignore vendor directory and report an error if go.mod needs to be updated
#-gcflags="all=-m": Print optimizations applied by the compiler for review and verification against security requirements.
#-gcflags="all=-spectre=all" Enable all available Spectre mitigations
#-ldflags="all=-s -w" remove the symbol and debug info
#-ldflags="all=-X ..." Embedd binary build stamping information.
# Note that arm64 (MacBooks) does not support any spectre mititations.
GOARCH:=$(shell go env GOARCH)
ifeq ($(GOARCH),arm64)
  GOEXTRAFLAGS:=-trimpath -gcflags="all=-spectre= -N -l" -asmflags="all=-spectre=" -ldflags="all=-s -w -X 'main.RepoURL=${DOCKER_LABEL_REPO_URL}' -X 'main.Version=${DOCKER_LABEL_VERSION}' -X 'main.Revision=${DOCKER_LABEL_REVISION}' -X 'main.BuildDate=${DOCKER_LABEL_BUILD_DATE}'"
else
  GOEXTRAFLAGS:=-trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -X 'main.RepoURL=${DOCKER_LABEL_REPO_URL}' -X 'main.Version=${DOCKER_LABEL_VERSION}' -X 'main.Revision=${DOCKER_LABEL_REVISION}' -X 'main.BuildDate=${DOCKER_LABEL_BUILD_DATE}'"
endif


ifneq ($(GIT_HASH_SHORT), main)
	ifeq ($(findstring -dev,$(VERSION)), -dev)
		VERSION := $(VERSION)-$(VERSION_DEV_SUFFIX)
	endif
endif

# Enviroment variable
IMG_NAME_OEP                           := tenancy-manager
IMG_VERSION_OEP                        ?= ${VERSION}
DOCKER_REGISTRY_OEP                    ?= registry-rs.edgeorchestration.intel.com
DOCKER_REPOSITORY_OEP                  ?= edge-orch/common
TENANCY_MANAGER_DOCKER_IMAGE_OEP       ?= ${DOCKER_REGISTRY_OEP}/${DOCKER_REPOSITORY_OEP}/${IMG_NAME_OEP}:${IMG_VERSION_OEP}

# Fuzz timeout
FUZZ_SECONDS                   ?= 30

YAML_FILES := $(shell find . -path './venv_charts' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
# Create the virtualenv with python tools installed
VENV_NAME = venv_tenancy_manager

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: vendor
vendor:
	go mod vendor

build.amd64: tidy
	go mod download
	mkdir -p bin/amd64
	GOARCH=amd64 GOOS=linux go build -buildvcs=false -o bin/amd64 ./...

build.darwin: tidy
	go mod download
	mkdir -p bin/darwin
	GOARCH=arm64 GOOS=darwin go build -buildvcs=false -o bin/darwin ./...

create:
	mkdir -p ${BUILD_DIR}

PHONY: go-build
go-build: tidy vendor create ## go build
	${GOCMD} build $(GOEXTRAFLAGS) -o ${BUILD_DIR}/ ./...

.PHONY: build
build: go-build docker.build.oep

.PHONY: docker.build.oep
docker.build.oep:
	go mod vendor
	docker build --build-arg TENANCY_MANAGER_COMPONENT_NAME=${TENANCY_MANAGER_COMPONENT_NAME} --build-arg HTTP_PROXY=${http_proxy} --build-arg HTTPS_PROXY=${https_proxy} --build-arg FTP_PROXY=${ftp_proxy} --build-arg SOCKS_PROXY=${socks_proxy} --build-arg NO_PROXY=${no_proxy} -t ${TENANCY_MANAGER_DOCKER_IMAGE_OEP} -f Dockerfile .

.PHONY: release
release: docker.publish.oep

.PHONY: docker.publish.oep
docker.publish.oep:
	docker push ${TENANCY_MANAGER_DOCKER_IMAGE_OEP}


.PHONY: test
test: go-build
	${GOCMD} test --race -coverprofile ${BUILD_DIR}/coverage.out -coverpkg=$$(go list ./... | grep -v "pkg/tenancy" | grep -v "tenancy-manager" | grep -v "cluster-orchestrator" | tr '\n' ,)  -covermode atomic ./...
	gocover-cobertura < ${BUILD_DIR}/coverage.out > ${BUILD_DIR}/coverage.xml
	${GOCMD} tool cover -html=${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/coverage.html
	${GOCMD} tool cover -func ${BUILD_DIR}/coverage.out -o ${BUILD_DIR}/function_coverage.log

coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

.PHONY: lint
lint: go-lint hadolint yamllint

.PHONY: lint-ci
lint-ci: go-lint hadolint license

.PHONY: license
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

.PHONY: hadolint
hadolint: ## Check Dockerfile with Hadolint
	hadolint --ignore DL3008 Dockerfile

.PHONY: go-lint
go-lint: ## lint code
	golangci-lint --version
	golangci-lint run --config .golangci.yml

.PHONY: yamllint
yamllint:
	yamllint --version
	yamllint -c yamllint_conf.yml -s $(YAML_FILES)

clean-testdata:
	rm -rf ./fuzztest/testdata

.PHONY: test-fuzz
test-fuzz: clean-testdata
	$(GOCMD) test ./fuzztest -fuzz FuzzTenancyOrgProjectCreate -fuzztime=${FUZZ_SECONDS}s

help:
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
        | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
        | sort \
        | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
        | xargs -I _ sh -c 'printf "%-20s " _; make _ -nB | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'
