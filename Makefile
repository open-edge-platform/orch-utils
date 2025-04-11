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

DOCKER_DIRS=auth-service aws-sm-proxy cert-synchronizer keycloak-tenant-controller/images nexus/openapi-generator nexus/compiler nexus/compiler/builder nexus-api-gateway secrets squid-proxy tenancy-api-mapping tenancy-datamodel tenancy-manager token-fs

docker-list: ## list all docker containers built by this repo
	@echo "images:"
	@for d in $(DOCKER_DIRS); do \
		echo "  $$d:" ;\
	  echo "    name: '$$d'" ;\
	  echo "    version: ''" ;\
	  echo "    gitTagPrefix: ''" ;\
	  echo "    buildTarget: 'docker-build'" ;\
	done

#### Help Target ####
help: ## print help for each target
	@echo orch-utils make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]%_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'
