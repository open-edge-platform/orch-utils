# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

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


BUILD_DIR 					?= build
CRD_MODULE_PATH 				?= $(shell go list -m)/${BUILD_DIR}/
CONTAINER_ID 					?= ""
DATAMODEL_LOCAL_PATH 				?= $(realpath .)
BUCKET 						?= nexus-template-downloads
COMPILER_TAG 					?= latest
DATAMODEL_DOCKER_REGISTRY 			?= $(shell cat nexus.yaml | grep dockerRepo | awk '{print $$2}' | tr -d '"')
DATAMODEL_GROUP_NAME 				?= $(shell cat nexus.yaml | grep groupName | awk '{print $$2}' | tr -d '"')
NAMESPACE 					?= "default"
HOST_KUBECONFIG 				?= ${HOME}/.kube/config
MOUNTED_KUBECONFIG 				?= /etc/config/kubeconfig
NEXUS_API_GW_K8S_NAMESPACE 			?= default
DATAMODEL_BASE_DOCKER_IMAGE			=gcr.io/nsx-sm/photon-kubectl:latest
DOCKER_NETWORK 					?= nexus
MOUNTED_KUBECONFIG 				?= /etc/config/kubeconfig
TAG 						?= latest

# values for nexus datamodel upgrade
FORCE 						?= true

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

COMPILER_DOCKER_REGISTRY ?= registry-rs.edgeorchestration.intel.com
COMPILER_DOCKER_REPO ?= edge-orch/common
IMAGE_NAME ?= nexus/compiler
GOARCH ?= $(shell go env GOARCH)
COMPILER_NAME ?= ${IMAGE_NAME}/${GOARCH}
ifeq (${COMPILER_DOCKER_REGISTRY}, )
COMPILER_DOCKER_TAG = ${COMPILER_NAME}:${COMPILER_TAG}
else
COMPILER_DOCKER_TAG = ${COMPILER_DOCKER_REGISTRY}/${COMPILER_DOCKER_REPO}/${COMPILER_NAME}:${COMPILER_TAG}
endif


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
			--volume $(realpath ../..):/go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/ \
			-v /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/datamodel/build/ \
			--volume $(realpath .)/build:/go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/generated/ \
			--volume ~/.ssh:/root/.ssh \
			--volume ${HOME}/.cache/go-build:/home/appuser/.cache/go-build \
			-e GOCACHE=/home/appuser/.cache/go-build \
			-e CRD_MODULE_PATH=${CRD_MODULE_PATH} \
			-e CONFIG_FILE=datamodel/nexus.yaml \
			-e GOPRIVATE=*.eng.vmware.com \
			-e GOINSECURE=*.eng.vmware.com \
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

