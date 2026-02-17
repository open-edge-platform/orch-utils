# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

### configuration ###
# safe shell invocations
SHELL                       := bash -eu -o pipefail

# default goal to show help
.DEFAULT_GOAL               := help

VERSION                     ?= $(shell cat VERSION)
BUILD_DIR                   ?= build
CRD_MODULE_PATH             ?= $(shell go list -m)/${BUILD_DIR}/
DATAMODEL_LOCAL_PATH        ?= $(realpath .)

CONTAINER_ID                ?= ""
BUCKET                      ?= nexus-template-downloads

DATAMODEL_IMAGE_NAME        = tenancy-datamodel
DATAMODEL_DOCKER_REGISTRY   ?= registry-rs.edgeorchestration.intel.com
DATAMODEL_GROUP_NAME        ?= edge-orch/common
DATAMODEL_VERSION           ?= ${VERSION}

ifeq (${DATAMODEL_DOCKER_REGISTRY}, )
DATAMODEL_DOCKER_TAG=${DATAMODEL_GROUP_NAME}/tenancy-datamodel:${DATAMODEL_VERSION}
else
DATAMODEL_DOCKER_TAG=${DATAMODEL_DOCKER_REGISTRY}/${DATAMODEL_GROUP_NAME}/tenancy-datamodel:${DATAMODEL_VERSION}
endif

DATAMODEL_BASE_DOCKER_IMAGE = alpine/kubectl:1.34.1

NAMESPACE                   ?= "default"
HOST_KUBECONFIG             ?= ${HOME}/.kube/config
MOUNTED_KUBECONFIG          ?= /etc/config/kubeconfig

NEXUS_API_GW_K8S_NAMESPACE  ?= default
DOCKER_NETWORK              ?= nexus

# values for nexus datamodel upgrade
FORCE                       ?= true
ARTIFACT_REPO               ?= $(shell cat nexus.yaml | grep artifactRepo | awk '{print $$2}' | tr -d '"')

ifneq (,$(shell echo ${PREV_SPEC_BRANCH}))
# backward compatability check is required.
FORCE=false
endif

TARGET_REPO_NAME     := $(shell echo ${ARTIFACT_REPO} | awk -F'/' '{print $$2}' | tr -d '"'| awk -F'.git' '{print $$1}')
ARTIFACT_REPO_GITCMD := $(shell echo git -C $(shell realpath ${TARGET_REPO_NAME}) pull origin $(shell echo ${PREV_SPEC_BRANCH}))

COMPILER_DOCKER_REGISTRY    ?= $(shell cat nexus.yaml | grep dockerRepo | awk '{print $$2}' | tr -d '"')
COMPILER_DOCKER_REPO        ?= edge-orch/common

GOARCH                      ?= $(shell go env GOARCH)
COMPILER_NAME               ?= nexus/compiler/${GOARCH}
COMPILER_VERSION            ?= $(shell cat ../nexus/VERSION)

OPENAPI_GENERATOR_NAME      ?= nexus/openapi-generator
OPENAPI_GENERATOR_VERSION   ?= $(shell cat ../nexus/VERSION)

ifeq (${COMPILER_DOCKER_REGISTRY}, )
COMPILER_DOCKER_TAG = ${COMPILER_NAME}:${COMPILER_VERSION}
OPENAPI_GENERATOR_DOCKER_TAG = ${OPENAPI_GENERATOR_NAME}:${OPENAPI_GENERATOR_VERSION}
else
COMPILER_DOCKER_TAG = ${COMPILER_DOCKER_REGISTRY}/${COMPILER_DOCKER_REPO}/${COMPILER_NAME}:${COMPILER_VERSION}
OPENAPI_GENERATOR_DOCKER_TAG = ${COMPILER_DOCKER_REGISTRY}/${COMPILER_DOCKER_REPO}/${OPENAPI_GENERATOR_NAME}:${OPENAPI_GENERATOR_VERSION}
endif

YAML_FILES := $(shell find . -path './venv_charts' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
# Create the virtualenv with python tools installed
VENV_NAME = venv_datamodel

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

.PHONY: lint
lint: go-lint hadolint license shellcheck yamllint

.PHONY: lint-ci
lint-ci: go-lint hadolint license shellcheck

.PHONY: license
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

.PHONY: hadolint
hadolint: ## Check Dockerfile with Hadolint
	hadolint Dockerfile

.PHONY: go-lint
go-lint: ## lint checks using the make targets
	golangci-lint --version
	golangci-lint run --timeout 5m --config .golangci.yml ./...

.PHONY: yamllint
yamllint:
	yamllint --version
	yamllint -c yamllint_conf.yml -s $(YAML_FILES)

.PHONY: shellcheck
# https://github.com/koalaman/shellcheck
SH_FILES := $(shell find . -type f \( -name '*.sh' \) -print )
shellcheck: ## lint shell scripts with shellcheck
	shellcheck --version
	shellcheck -x -S style $(SH_FILES)

.PHONY: go-tidy
go-tidy:
	go mod tidy

.PHONY: datamodel_build
datamodel_build:
	@# Help: Build datamodel source code and docker image
	mkdir -p ${BUILD_DIR}
	@echo "FORCE: ${FORCE}"
	@echo "PREV_SPEC_BRANCH: ${PREV_SPEC_BRANCH}"
	@echo "ARTIFACT_REPO: ${ARTIFACT_REPO}"
	@echo "TARGET_REPO_NAME: ${TARGET_REPO_NAME}"
	@echo "ARTIFACT_REPO_GITCMD: ${ARTIFACT_REPO_GITCMD}"

	if [ ${FORCE} = false ] ; then \
            		bash -c "${ARTIFACT_REPO_GITCMD}";\
              		cp -rf ${TARGET_REPO_NAME}/crds ${BUILD_DIR} ;\
                	echo "Copying of ${TARGET_REPO_NAME}/crds to the destination directory ${BUILD_DIR}/crds"; \
    fi ;\

	if [ -z $(CONTAINER_ID) ]; then \
		docker run --pull=missing\
			--volume $(realpath .):/go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/datamodel/ \
			-v /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/datamodel/build/ \
			--volume $(realpath .)/build:/go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/generated/ \
			--volume ~/.ssh:/root/.ssh \
			--volume ${HOME}/.cache/go-build:/home/appuser/.cache/go-build \
			-e GOCACHE=/home/appuser/.cache/go-build \
			-e CRD_MODULE_PATH=${CRD_MODULE_PATH} \
			-e CONFIG_FILE=datamodel/nexus.yaml \
			-e GOPRIVATE=*.eng.vmware.com \
			-e GOINSECURE=*.eng.vmware.com \
			--user root:root \
			-e FORCE=$(FORCE) \
			${COMPILER_DOCKER_TAG} /bin/bash -c 'make generate_code && chown -R $(shell id -u):$(shell id -g) ./datamodel/build;'; \
	else \
		docker run --pull=missing\
			--volumes-from=$(CONTAINER_ID) \
			-e DATAMODEL_PATH=$(DATAMODEL_LOCAL_PATH) \
			-e GENERATED_OUTPUT_DIRECTORY=$(DATAMODEL_LOCAL_PATH)/build \
			-e CONFIG_FILE=${DATAMODEL_LOCAL_PATH}/nexus.yaml \
			-e CRD_MODULE_PATH=${CRD_MODULE_PATH} \
			-e GOPRIVATE=*.eng.vmware.com \
			-e GOINSECURE=*.eng.vmware.com \
			-e CICD_TOKEN=${CICD_TOKEN} \
			-e FORCE=${FORCE} \
			--user root:root \
			-e FORCE=$(FORCE) \
			${COMPILER_DOCKER_TAG}; \
	fi
	make openapispec.generate

.PHONY: openapispec.generate
openapispec.generate:
	docker run \
		--pull=missing \
		--mount type=bind,source=$(realpath .),target=/datamodel \
		--entrypoint /bin/openapi-generator \
		${OPENAPI_GENERATOR_DOCKER_TAG} \
		--yamls-path /datamodel/build/crds --datamodel-path /datamodel/nexus.yaml --datamodel-name ${DATAMODEL_GROUP_NAME} --output-file-path /datamodel/build/openapi/edge-orchestrator.intel.com.json

# Usage: DATAMODEL_DOCKER_REGISTRY=<registry> make docker.build
# Tag and build datamodel docker image.
#
# Example: DATAMODEL_DOCKER_REGISTRY=822995803632.dkr.ecr.us-west-2.amazonaws.com make docker.build

.PHONY: build
build: docker.build

.PHONY: build.all
build.all: docker.build

.PHONY: docker.build
docker.build:
	@echo "${DATAMODEL_DOCKER_TAG}" > IMAGE
	@echo "${DATAMODEL_GROUP_NAME}" > NAME
	docker build -t ${DATAMODEL_DOCKER_TAG} . -f Dockerfile

# Usage: DATAMODEL_DOCKER_REGISTRY=<registry> TAG=<tag-value> make docker.publish
# Tag and publish datamodel docker image.
#
# Example: DATAMODEL_DOCKER_REGISTRY=822995803632.dkr.ecr.us-west-2.amazonaws.com make docker.publish

.PHONY: release
release: docker.publish

.PHONY: docker.publish
docker.publish:
	@# Help: Publish datamodel docker image in oep registry
	docker push ${DATAMODEL_DOCKER_TAG}

.PHONY: dm.install
dm.install:
	@# Help: Run datamodel install
	docker run \
		--net ${DOCKER_NETWORK} \
		--pull=missing \
		--mount type=bind,source=${HOST_KUBECONFIG},target=${MOUNTED_KUBECONFIG},readonly \
		-e KUBECONFIG=${MOUNTED_KUBECONFIG} \
		-e NAME=${DATAMODEL_GROUP_NAME} \
		-e IMAGE=${DATAMODEL_DOCKER_TAG} \
                -e https_proxy= \
                -e http_proxy= \
                -e HTTPS_PROXY= \
                -e HTTP_PROXY= \
		${DATAMODEL_DOCKER_TAG}

.PHONY: dm.install.helm
dm.install.helm:
	cd helm-chart; helm install iam --set global.datamodelName=iam --set namespace=${NEXUS_API_GW_K8S_NAMESPACE} --set image=${DATAMODEL_DOCKER_TAG} --set serviceAccountName=datamodel-installer .

.PHONY: dm.uninstall.helm
dm.uninstall.helm:
		@# Help: Run datamodel helm uninstall
		cd helm-chart; helm uninstall iam

test:
	@# Help: Runs test stage
	@echo "---MAKEFILE TEST---"
	echo $@
	@echo "---END MAKEFILE TEST---"

coverage:
	@# Help: Runs coverage stage
	@echo "---MAKEFILE COVERAGE---"
	echo $@
	@echo "---END MAKEFILE COVERAGE---"

#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'
