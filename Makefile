# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SHELL	:= bash -eu -o pipefail

# default goal to show help
.DEFAULT_GOAL := help

HELM_DIRS=$(shell ls charts)
helm-list: ## List helm charts, tag format, and versions in YAML format
	@echo "charts:"
	@for d in $(HELM_DIRS); do \
    cname=$$(grep "^name:" "charts/$$d/Chart.yaml" | cut -d " " -f 2) ;\
    echo "  $$cname:" ;\
    echo -n "    "; grep "^version" "charts/$$d/Chart.yaml"  ;\
    echo "    gitTagPrefix: ''" ;\
    echo "    outDir: 'charts/$$d/build'" ;\
  done

helm-build: ## build all helm charts
	mage chartsBuild

docker-list: ## list all docker containers built by this repo
	@mage listContainers

# map container name to the mage build:... invocations
docker-build-auth-service:
	mage build:authService

docker-build-aws-sm-proxy:
	mage build:awsSmProxy

docker-build-cert-synchronizer:
	mage build:certSynchronizer

docker-build-keycloak-tenant-controller:
	mage build:keycloakTenantController

docker-build-nexus-api-gw:
	mage build:nexusAPIGateway

docker-build-nexus/compiler:
	mage build:nexusCompiler

docker-build-nexus/openapi-generator:
	mage build:openAPIGenerator

docker-build-secrets-config:
	mage build:secretsConfig

docker-build-squid-proxy:
	mage build:build:squidProxy

docker-build-tenancy-api-mapping:
	mage build:tenancyAPIMapping

docker-build-tenancy-datamodel:
	mage build:tenancyDatamodel

docker-build-tenancy-manager:
	mage build:tenancyManager

docker-build-token-fs:
	mage build:tokenFS

#### Help Target ####
help: ## print help for each target
	@echo orch-utils make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]%_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'
