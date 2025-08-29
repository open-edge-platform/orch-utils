# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SHELL := bash -eu -o pipefail

# default goal to show help
.DEFAULT_GOAL := help

# Build one service (ONLY_SERVICE=<service>) or all in SERVICES
ONLY_SERVICE ?=

# List of services
SERVICES ?= \
  authService \
  awsSmProxy \
  certSynchronizer \
  secretsConfig \
  squidProxy \
  tokenFS \
  tenancyAPIMapping \
  tenancyManager \
  tenancyDatamodel \
  nexusAPIGateway \
  keycloakTenantController \
  nexusCompiler \
  openAPIGenerator

# Suffix from CI (e.g. "-pr-123"); when set, images will be retagged
VERSION_SUFFIX ?=

.PHONY: docker-build docker-push helm-build helm-push retag-with-suffix

# Build containers for CI (uses ONLY_SERVICE or all)
docker-build: ## Build Docker images (ONLY_SERVICE builds one; otherwise builds all SERVICES)
	@if [ -n "$(ONLY_SERVICE)" ]; then \
		svc="$(ONLY_SERVICE)"; \
		echo "==> Building single service: $$svc"; \
		case $$svc in \
			authService)               $(MAKE) docker-build-auth-service ;; \
			awsSmProxy)                $(MAKE) docker-build-aws-sm-proxy ;; \
			certSynchronizer)          $(MAKE) docker-build-cert-synchronizer ;; \
			secretsConfig)             $(MAKE) docker-build-secrets-config ;; \
			squidProxy)                $(MAKE) docker-build-squid-proxy ;; \
			tokenFS)                   $(MAKE) docker-build-token-fs ;; \
			tenancyAPIMapping)         $(MAKE) docker-build-tenancy-api-mapping ;; \
			tenancyManager)            $(MAKE) docker-build-tenancy-manager ;; \
			tenancyDatamodel)          $(MAKE) docker-build-tenancy-datamodel ;; \
			nexusAPIGateway)           $(MAKE) docker-build-nexus-api-gw ;; \
			keycloakTenantController)  $(MAKE) docker-build-keycloak-tenant-controller ;; \
			nexusCompiler)             $(MAKE) docker-build-nexus/compiler ;; \
			openAPIGenerator)          $(MAKE) docker-build-nexus/openapi-generator ;; \
			*) echo "Unknown service '$$svc'"; exit 2 ;; \
		esac; \
	else \
		echo "==> Building all services: $(SERVICES)"; \
		for svc in $(SERVICES); do \
			ONLY_SERVICE="$$svc" $(MAKE) docker-build; \
		done; \
	fi
	@if [ -n "$(VERSION_SUFFIX)" ]; then \
		$(MAKE) retag-with-suffix; \
	else \
		echo "VERSION_SUFFIX empty; skipping retag."; \
	fi

# Push containers (uses ONLY_SERVICE or all) via mage
docker-push: ## Push Docker images to registry (ONLY_SERVICE or all SERVICES) using mage push:<service>
	@if [ -n "$(ONLY_SERVICE)" ]; then \
		echo "==> Pushing $(ONLY_SERVICE) via mage"; \
		mage push:$(ONLY_SERVICE); \
	else \
		echo "==> Pushing all services via mage"; \
		for svc in $(SERVICES); do mage push:$$svc; done; \
	fi

# Retag built images with VERSION_SUFFIX using mage listContainers
retag-with-suffix: docker-list ## Retag images discovered by 'mage listContainers' with VERSION_SUFFIX
    @set -euo pipefail; \
    echo "==> Retagging with suffix '$(VERSION_SUFFIX)'"; \
    images=$$(mage listContainers); \
    if [ -z "$$images" ]; then echo "No images from mage listContainers"; exit 0; fi; \
    echo "$$images" | while read -r line; do \
      # Expect 'repo:tag' in the first column; skip anything else
      name_tag=$$(echo "$$line" | awk '{print $$1}'); \
      case "$$name_tag" in *:*) ;; *) echo "Skip: $$line"; continue ;; esac; \
      repo=$${name_tag%:*}; \
      tag=$${name_tag##*:}; \
      new_tag="$$tag$(VERSION_SUFFIX)"; \
      echo "Tagging $$repo:$$tag -> $$repo:$$new_tag"; \
      docker tag "$$repo:$$tag" "$$repo:$$new_tag"; \
    done

# ------------------------------
# Helm helpers
# ------------------------------
HELM_DIRS = $(shell ls charts)
helm-list: ## List helm charts, tag format, and versions in YAML format
    @echo "charts:"
    @for d in $(HELM_DIRS); do \
    cname=$$(grep "^name:" "charts/$$d/Chart.yaml" | cut -d " " -f 2) ;\
    echo "  $$cname:" ;\
    echo -n "    "; grep "^version" "charts/$$d/Chart.yaml"  ;\
    echo "    gitTagPrefix: 'chart/$$d/v'" ;\
    echo "    outDir: 'charts/$$d/build'" ;\
  done

helm-build: ## build all helm charts
    mage chartsBuild

helm-push: helm-build ## push helm charts (no-op by default; wire up if needed)
	@echo "helm-push: no-op (implement chart publishing if desired)"

# ------------------------------
# Docker helpers
# ------------------------------
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
    mage build:squidProxy

docker-build-tenancy-api-mapping:
    mage build:tenancyAPIMapping

docker-build-tenancy-datamodel:
    mage build:tenancyDatamodel

docker-build-tenancy-manager:
    mage build:tenancyManager

docker-build-token-fs:
    mage build:tokenFS

# ------------------------------
# Tests
# ------------------------------
ginkgo: ## Run all ginkgo tests in sub-projects
    make -C auth-service        ginkgo
    make -C aws-sm-proxy        ginkgo
    make -C internal            ginkgo
    make -C nexus-api-gw        ginkgo
    make -C nexus               ginkgo
    make -C secrets             ginkgo
    make -C tenancy-manager     ginkgo
    # make -C tenancy-api-mapping ginkgo  # needs to be fixed

#### Help Target ####
help: ## print help for each target
    @echo orch-utils make targets
    @echo "Target               Makefile:Line    Description"
    @echo "-------------------- ---------------- -----------------------------------------"
    @grep -H -n '^[[:alnum:]%_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'