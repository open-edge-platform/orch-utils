# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

### configuration ###
# safe shell invocations
SHELL                     := bash -eu -o pipefail

# default goal to show help
.DEFAULT_GOAL             := help

VERSION                     ?= $(shell cat VERSION)
BUILD_DIR                   ?= build
CRD_MODULE_PATH             ?= $(shell go list -m)/${BUILD_DIR}/
DATAMODEL_LOCAL_PATH        ?= $(realpath .)

CONTAINER_ID                ?= ""
BUCKET                      ?= nexus-template-downloads


DATAMODEL_BASE_DOCKER_IMAGE	= bitnami/kubectl:latest
DATAMODEL_DOCKER_REGISTRY   ?= $(shell cat nexus.yaml | grep dockerRepo | awk '{print $$2}' | tr -d '"')
DATAMODEL_GROUP_NAME        ?= $(shell cat nexus.yaml | grep groupName | awk '{print $$2}' | tr -d '"')

NAMESPACE                   ?= "default"
HOST_KUBECONFIG             ?= ${HOME}/.kube/config
MOUNTED_KUBECONFIG          ?= /etc/config/kubeconfig

NEXUS_API_GW_K8S_NAMESPACE  ?= default
DOCKER_NETWORK              ?= nexus

# Environment variable
IMG_NAME_OEP               := tenancy-datamodel
IMG_VERSION_OEP            ?= ${VERSION}
DOCKER_REGISTRY_OEP        ?= registry-rs.edgeorchestration.intel.com
DOCKER_REPOSITORY_OEP      ?= edge-orch/common
TENANCY_DATAMODEL_DOCKER_IMAGE_OEP  ?= ${DOCKER_REGISTRY_OEP}/${DOCKER_REPOSITORY_OEP}/${IMG_NAME_OEP}:${IMG_VERSION_OEP}

# values for nexus datamodel upgrade
FORCE ?= true

ifeq (,$(shell echo ${ARTIFACT_REPO}))
ARTIFACT_REPO = $(shell cat nexus.yaml | grep artifactRepo | awk '{print $$2}' | tr -d '"')
endif

ifneq (,$(shell echo ${PREV_SPEC_BRANCH}))
# backward compatability check is required.
FORCE=false

# check if ARTIFACT_REPO is configured
ifeq (,$(shell echo ${ARTIFACT_REPO}))
# Error and exit because user provided the branch and artifact repo details are not available
$(error Unable to determine ARTIFACT_REPO to validate backward compatibility with ${PREV_SPEC_BRANCH} branch)
endif

TARGET_REPO_NAME = $(shell echo ${ARTIFACT_REPO} | awk -F'/' '{print $$2}' | tr -d '"'| awk -F'.git' '{print $$1}')

ifeq (,$(shell echo $(shell realpath ${TARGET_REPO_NAME}/crds)))
ARTIFACT_REPO_GITCMD=$(shell echo git clone $(shell echo ${ARTIFACT_REPO}) --branch $(shell echo ${PREV_SPEC_BRANCH}) --single-branch $(shell echo ${TARGET_REPO_NAME}))
else
ARTIFACT_REPO_GITCMD=$(shell echo git -C $(shell realpath ${TARGET_REPO_NAME}) pull origin $(shell echo ${PREV_SPEC_BRANCH}))
endif

endif

ifeq (${DATAMODEL_DOCKER_REGISTRY}, )
DATAMODEL_DOCKER_TAG=tenancy-datamodel/${DATAMODEL_GROUP_NAME}:${TAG}
else
DATAMODEL_DOCKER_TAG=${DATAMODEL_DOCKER_REGISTRY}/tenancy-datamodel/${DATAMODEL_GROUP_NAME}:${TAG}
endif

COMPILER_DOCKER_REGISTRY  ?=
COMPILER_DOCKER_REPO      ?= edge-orch/common

GOARCH            ?= $(shell go env GOARCH)

COMPILER_NAME     ?= nexus/compiler/${GOARCH}
COMPILER_VERSION  ?= 0.1.0-dev

OPENAPI_GENERATOR_NAME      ?= nexus/openapi-generator
OPENAPI_GENERATOR_VERSION   ?= 0.1.0-dev

ifeq (${COMPILER_DOCKER_REGISTRY}, )
COMPILER_DOCKER_TAG = ${COMPILER_NAME}:${COMPILER_VERSION}
OPENAPI_GENERATOR_DOCKER_TAG = ${OPENAPI_GENERATOR_NAME}:${OPENAPI_GENERATOR_VERSION}
else
COMPILER_DOCKER_TAG = ${COMPILER_DOCKER_REGISTRY}/${COMPILER_DOCKER_REPO}/${COMPILER_NAME}:${COMPILER_VERSIOn}
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
	hadolint Dockerfile

.PHONY: go-lint
go-lint: ## lint checks using the make targets
	golangci-lint --version
	golangci-lint run --timeout 5m --config .golangci.yml ./...

.PHONY: yamllint
yamllint:
	yamllint --version
	yamllint -c yamllint_conf.yml -s $(YAML_FILES)

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

# Usage: DATAMODEL_DOCKER_REGISTRY=<registry> TAG=<tag-value> make docker.build
# Tag and build datamodel docker image.
#
# Example: DATAMODEL_DOCKER_REGISTRY=822995803632.dkr.ecr.us-west-2.amazonaws.com TAG=letstest make docker.build
#
.PHONY: build
build: docker.build-oep

.PHONY: docker.build
docker.build:
ifeq ($(DATAMODEL_DOCKER_REGISTRY),)
$(error ERROR: DATAMODEL_DOCKER_REGISTRY is a mandatory argument. Configure it to point to container registry.)
endif
ifeq ($(DATAMODEL_DOCKER_TAG),)
$(error ERROR: TAG is a mandatory argument. It should contain datamodel container image tag.)
endif
	@echo "${DATAMODEL_DOCKER_TAG}" > IMAGE
	@echo "${DATAMODEL_GROUP_NAME}" > NAME
	docker build --build-arg DOCKER_BASE_IMAGE=${DATAMODEL_BASE_DOCKER_IMAGE} --build-arg IMAGE_NAME=${DATAMODEL_DOCKER_TAG} --build-arg NAME=${DATAMODEL_GROUP_NAME} -t ${DATAMODEL_DOCKER_TAG} . -f Dockerfile


.PHONY: docker.build-oep
docker.build-oep:
ifeq ($(DATAMODEL_DOCKER_REGISTRY),)
$(error ERROR: DATAMODEL_DOCKER_REGISTRY is a mandatory argument. Configure it to point to container registry.)
endif
ifeq ($(DATAMODEL_DOCKER_TAG),)
$(error ERROR: TAG is a mandatory argument. It should contain datamodel container image tag.)
endif
	@echo "${DATAMODEL_DOCKER_TAG}" > IMAGE
	@echo "${DATAMODEL_GROUP_NAME}" > NAME
	docker build --build-arg DOCKER_BASE_IMAGE=${DATAMODEL_BASE_DOCKER_IMAGE} --build-arg IMAGE_NAME=${DATAMODEL_DOCKER_TAG} --build-arg NAME=${DATAMODEL_GROUP_NAME} -t ${TENANCY_DATAMODEL_DOCKER_IMAGE_OEP} . -f Dockerfile
#
# Usage: DATAMODEL_DOCKER_REGISTRY=<registry> TAG=<tag-value> make docker.publish
# Tag and publish datamodel docker image.
#
# Example: DATAMODEL_DOCKER_REGISTRY=822995803632.dkr.ecr.us-west-2.amazonaws.com TAG=letstest make docker.publish
#
# .PHONY: docker.publish
# docker.publish:
# 	docker push $(TENANCY_DATAMODEL_DOCKER_IMAGE_OEP)
.PHONY: release
release: docker.publish.oep

.PHONY: docker.publish.oep
docker.publish.oep:
	@# Help: Publish datamodel docker image in oep registry
	docker push $(TENANCY_DATAMODEL_DOCKER_IMAGE_OEP)

.PHONY: openapispec.generate
openapispec.generate:
	docker run \
		--pull=missing \
		--mount type=bind,source=$(realpath .),target=/datamodel \
		--entrypoint /bin/openapi-generator \
		${OPENAPI_GENERATOR_DOCKER_TAG} \
		--yamls-path /datamodel/build/crds --datamodel-path /datamodel/nexus.yaml --datamodel-name ${DATAMODEL_GROUP_NAME} --output-file-path /datamodel/build/openapi/edge-orchestrator.intel.com.json

#
# Usage: DATAMODEL_DOCKER_REGISTRY=<registry> TAG=<tag-value> make docker.build
# Tag and build datamodel docker image.
#
# Example: DATAMODEL_DOCKER_REGISTRY=822995803632.dkr.ecr.us-west-2.amazonaws.com TAG=letstest make docker.build
#
.PHONY: build.all
build.all: datamodel_build
ifeq ($(DATAMODEL_DOCKER_REGISTRY),)
$(error ERROR: DATAMODEL_DOCKER_REGISTRY is a mandatory argument. Configure it to point to container registry.)
endif
ifeq ($(DATAMODEL_DOCKER_TAG),)
$(error ERROR: TAG is a mandatory argument. It should contain datamodel container image tag.)
endif
	docker build --build-arg DOCKER_BASE_IMAGE=${DATAMODEL_BASE_DOCKER_IMAGE} --build-arg IMAGE_NAME=${DATAMODEL_DOCKER_TAG} --build-arg NAME=${DATAMODEL_GROUP_NAME} -t ${DATAMODEL_DOCKER_TAG} . -f Dockerfile

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

#
# Usage: DATAMODEL_DOCKER_REGISTRY=<registry> TAG=<tag-value> make dm.install.helm
# Install datamodel using helm
#
# Example: DATAMODEL_DOCKER_REGISTRY=822995803632.dkr.ecr.us-west-2.amazonaws.com TAG=letstest make dm.install.helm
#
.PHONY: dm.install.helm
dm.install.helm:
	@# Help: Run datamodel helm install
	ifeq ($(DATAMODEL_DOCKER_REGISTRY),)
	$(info ERROR: DATAMODEL_DOCKER_REGISTRY is a mandatory argument. Configure it to point to container registry.)
	endif
	ifeq ($(DATAMODEL_DOCKER_TAG),)
	$(info ERROR: TAG is a mandatory argument. It should contain datamodel container image tag.)
	endif
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
