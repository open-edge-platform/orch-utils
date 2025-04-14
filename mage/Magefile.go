// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"fmt"
	"regexp"

	"github.com/bitfield/script"
	"github.com/magefile/mage/mg"
)

var PublicCharts = []string{"aws-sm-get-rs-token", "aws-sm-proxy", "oci-secret", "secret-wait", "token-refresh"}
var PublicContainers = []string{"aws-sm-proxy"}

const (
	AWSRegion                         = "us-west-2"
	OpenEdgePlatformRegistryRepoURL   = "080137407410.dkr.ecr.us-west-2.amazonaws.com"
	OpenEdgePlatformRepository        = "edge-orch"
	RegistryRepoSubProj               = "common"
	OpenEdgePlatformContainerRegistry = OpenEdgePlatformRegistryRepoURL + "/" +
		OpenEdgePlatformRepository + "/" +
		RegistryRepoSubProj
	OpenEdgePlatformChartRegistry = OpenEdgePlatformRegistryRepoURL + "/" +
		OpenEdgePlatformRepository + "/" +
		RegistryRepoSubProj + "/charts"
	ECRPublicChartRegistry = OpenEdgePlatformChartRegistry
)

var globalAsdf = []string{
	// there are currently no global tools;
}

// Install ASDF plugins.
func AsdfPlugins() error {
	if _, err := script.File(".tool-versions").Column(1).Reject("catalog-cli").Reject("catalog-schema-tool").
		MatchRegexp(regexp.MustCompile(`^[^\#]`)).ExecForEach("asdf plugin add {{.}}").Stdout(); err != nil {
		return err
	}
	if _, err := script.Exec("asdf install").Stdout(); err != nil {
		return err
	}
	if _, err := script.Exec("asdf current").Stdout(); err != nil {
		return err
	}
	// Set plugins listed in globalAsdf as global
	for _, name := range globalAsdf {
		if _, err := script.File(".tool-versions").MatchRegexp(regexp.MustCompile(name)).Column(2).
			ExecForEach(fmt.Sprintf("asdf global %s {{.}}", name)).Stdout(); err != nil {
			return err
		}
	}
	fmt.Printf("asdf plugins updated 🔌\n")
	return nil
}

type Lint mg.Namespace

// Lint everything.
func (l Lint) All() error {
	if err := l.helm(); err != nil {
		return err
	}
	if err := l.yaml(); err != nil {
		return err
	}
	if err := l.golang(); err != nil {
		return err
	}
	if err := l.dockerfiles(); err != nil {
		return err
	}
	return nil
}

// Lint helm templates.
func (l Lint) Helm() error {
	return l.helm()
}

// Lint helm templates.
func (l Lint) Yaml() error {
	return l.yaml()
}

// Lint golang files.
func (l Lint) Golang() error {
	return l.golang()
}

// Lint golang files.
func (l Lint) Dockerfiles() error {
	return l.dockerfiles()
}

type Gen mg.Namespace

// Re-generate traefik plugin config maps with source code of the middleware.
func (g Gen) TraefikPlugins() error {
	return jwtPluginConfigmap()
}

type Build mg.Namespace

// Builds the secrets-config container image.
func (Build) SecretsConfig() error {
	return secretsConfigBuild()
}

// Builds the aws-sm-proxy container image.
func (Build) AwsSmProxy() error {
	return awsSmProxyBuild()
}

// Builds the token-fs container image.
func (Build) TokenFS() error {
	return tokenFSBuild()
}

// Builds the authService container image.
func (Build) AuthService() error {
	return authServiceBuild()
}

// Builds the CertSynchronizer container image.
func (Build) CertSynchronizer() error {
	return certSynchronizerBuild()
}

// Builds the SquidProxy container image.
func (Build) SquidProxy() error {
	return squidProxyBuild()
}

// Builds the Keycloak Tenant Controller container image.
func (Build) KeycloakTenantController() error {
	return keycloakTenantControllerBuild()
}

// Builds the Nexus compiler builder container image.
func (Build) NexusCompiler() error {
	return datamodelCompilerBuild()
}

// Builds the Tenancy Datamodel container image.
func (Build) TenancyDatamodel() error {
	return tenancyDatamodelBuild()
}

// Builds the Tenancy API Mapping container image.
func (Build) TenancyAPIMapping() error {
	return tenancyAPIMappingBuild()
}

// Builds the Tenancy Manager container image.
func (Build) TenancyManager() error {
	return tenancyManagerBuild()
}

// Builds the Nexus API Gateway container image.
func (Build) NexusAPIGateway() error {
	return nexusAPIGatewayBuild()
}

type Push mg.Namespace

// Push the secrets-config container image to the AMR registry.
func (Push) SecretsConfig() error {
	return pushImage("secrets-config", "secrets-config")
}

// Push the aws-sm-proxy container image to the AMR registry.
func (Push) AwsSmProxy() error {
	return pushImage("aws-sm-proxy", "aws-sm-proxy")
}

// Push the aws-sm-proxy container image to the AMR registry.
func (Push) TokenFs() error {
	return pushImage("token-fs", "token-file-server")
}

// Push the auth-service container image to the AMR registry.
func (Push) AuthService() error {
	return pushImage("auth-service", "auth-service")
}

// Push the cert-synchronizer container image to the AMR registry.
func (Push) CertSynchronizer() error {
	return pushImage("cert-synchronizer", "cert-synchronizer")
}

// Push the Keycloak Tenant Controller container image to the AMR registry.
func (Push) KeycloakTenantController() error {
	return pushImage("keycloak-tenant-controller", "keycloak-tenant-controller")
}

// Push the aws-sm-proxy container image to the ECR public registry.
func (Push) PublicAwsSmProxy() error {
	return pushImage("aws-sm-proxy", "aws-sm-proxy")
}

// Push the squid-proxy container image to the AMR registry.
func (Push) SquidProxy() error {
	return pushImage("squid-proxy", "squid-proxy")
}

// Push the openapi-generator container image to the registry.
func (Push) OpenAPIGenerator() error {
	return pushOpenAPIGeneratorImage()
}

// Push helm charts to the AMR registry.
func (Push) Charts() error {
	return pushCharts(OpenEdgePlatformChartRegistry)
}

// Push helm charts to the AMR registry.
func (Push) PublicCharts() error {
	return pushSpecificCharts(PublicCharts, ECRPublicChartRegistry)
}

// Push the Nexus compiler container image to the registry.
func (Push) NexusCompiler() error {
	return pushNexusCompilerImage()
}

// Push the Tenancy Datamodel container image to the registry.
func (Push) TenancyDatamodel() error {
	return pushImage("tenancy-datamodel",
		"tenancy-datamodel-init")
}

// Push the Tenancy API Mapping container image to the registry.
func (Push) TenancyAPIMapping() error {
	return pushImage("tenancy-api-mapping",
		"tenancy-api-remapping")
}

// Push the Tenancy Manager container image to the registry.
func (Push) TenancyManager() error {
	return pushImage("tenancy-manager",
		"tenancy-manager")
}

// Push the Nexus API Gateway container image to the registry.
func (Push) NexusAPIGateway() error {
	return pushImage("nexus-api-gw",
		"nexus-api-gw")
}

// Namespace contains test targets.
type Test mg.Namespace

// Test Go source files.
func (t Test) Golang() error {
	return t.golang()
}

// Namespace contains clean targets.
type Clean mg.Namespace

// Cleans the Tenancy API Mapping build environment.
func (Clean) TenancyAPIMapping() error {
	return tenancyAPIMappingClean()
}

// Cleans the Tenancy Manager build environment.
func (Clean) TenancyManager() error {
	return tenancyManagerClean()
}

// Cleans the Nexus API Gateway build environment.
func (Clean) NexusAPIGateway() error {
	return nexusAPIGatewayClean()
}

// Builds the OpenAPI-Generator container image.
func (Build) OpenAPIGenerator() error {
	return openAPIGeneratorBuild()
}

// Builds the OpenAPI-Generator container image.
func ListContainers() error {
	return listContainers()
}
