// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"gopkg.in/yaml.v3"
)

// Builds the secrets-config container image.
func secretsConfigBuild() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN must be set")
	}

	appVersion, err := getChartAppVersion("secrets-config")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--secret", "id=GITHUB_TOKEN,env=GITHUB_TOKEN",
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/secrets-config:"+appVersion, // For legacy support
		"--file", filepath.Join("secrets", "Dockerfile"),
		".",
	)
}

// Builds the aws-sm-proxy container image.
func awsSmProxyBuild() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN must be set")
	}

	appVersion, err := getChartAppVersion("aws-sm-proxy")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--secret", "id=GITHUB_TOKEN,env=GITHUB_TOKEN",
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/aws-sm-proxy:"+appVersion, // For legacy support
		"--file", filepath.Join("aws-sm-proxy", "Dockerfile"),
		".",
	)
}

func tokenFSBuild() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN must be set")
	}

	appVersion, err := getChartAppVersion("token-file-server")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--secret", "id=GITHUB_TOKEN,env=GITHUB_TOKEN",
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/token-fs:"+appVersion, // For legacy support
		"--file", filepath.Join("token-fs", "Dockerfile"),
		".",
	)
}

func authServiceBuild() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN must be set")
	}

	appVersion, err := getChartAppVersion("auth-service")
	if err != nil {
		return err
	}

	g0 := sh.OutCmd("git")
	commitID, err := g0("rev-parse", "HEAD")
	if err != nil {
		fmt.Printf("error running git rev-parse HEAD = %s", err)
	}
	gitarg := "GIT_COMMIT=" + commitID
	fmt.Printf("Git Arg = %s", gitarg)

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--secret", "id=GITHUB_TOKEN,env=GITHUB_TOKEN",
		"--build-arg", strings.Trim(gitarg, ""),
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/auth-service:"+appVersion, // For legacy support
		"--file", filepath.Join("auth-service", "Dockerfile"),
		"./auth-service",
	)
}

func getChartAppVersion(chartName string) (string, error) {
	contents, err := os.ReadFile(filepath.Join("charts", chartName, "Chart.yaml"))
	if err != nil {
		return "", fmt.Errorf("read Chart.yaml file: %w", err)
	}

	var chart struct {
		AppVersion string `yaml:"appVersion"`
	}
	if err := yaml.Unmarshal(contents, &chart); err != nil {
		return "", fmt.Errorf("parse Chart.yaml file: %w", err)
	}
	if chart.AppVersion == "" {
		return "", fmt.Errorf("appVersion in Chart.yaml file should not be empty")
	}
	return chart.AppVersion, nil
}

func getChartVersion(chartName string) (string, error) {
	contents, err := os.ReadFile(filepath.Join("charts", chartName, "Chart.yaml"))
	if err != nil {
		return "", fmt.Errorf("read Chart.yaml file: %w", err)
	}

	var chart struct {
		AppVersion string `yaml:"version"`
	}
	if err := yaml.Unmarshal(contents, &chart); err != nil {
		return "", fmt.Errorf("parse Chart.yaml file: %w", err)
	}
	if chart.AppVersion == "" {
		return "", fmt.Errorf("version in Chart.yaml file should not be empty")
	}
	return chart.AppVersion, nil
}

// Builds the cert-synchronizer container image.
func certSynchronizerBuild() error {
	appVersion, err := getChartAppVersion("cert-synchronizer")
	if err != nil {
		return err
	}

	g0 := sh.OutCmd("git")
	commitID, err := g0("rev-parse", "HEAD")
	if err != nil {
		fmt.Printf("error running git rev-parse HEAD = %s", err)
	}
	gitarg := "GIT_COMMIT=" + commitID
	fmt.Printf("Git Arg = %s", gitarg)
	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--build-arg", strings.Trim(gitarg, ""),
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/cert-synchronizer:"+appVersion, // For legacy support
		"--file", filepath.Join("cert-synchronizer", "Dockerfile"),
		"./cert-synchronizer",
	)
}

// Builds the squid-proxy container image.
func squidProxyBuild() error {
	appVersion, err := getChartAppVersion("squid-proxy")
	if err != nil {
		return err
	}

	g0 := sh.OutCmd("git")
	commitID, err := g0("rev-parse", "HEAD")
	if err != nil {
		fmt.Printf("error running git rev-parse HEAD = %s", err)
	}
	gitarg := "GIT_COMMIT=" + commitID
	fmt.Printf("Git Arg = %s", gitarg)
	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--progress=plain",
		"--build-arg", strings.Trim(gitarg, ""),
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/squid-proxy:"+appVersion, // For legacy support
		"--file", filepath.Join("squid-proxy", "Dockerfile"),
		"./squid-proxy",
	)
}

// Builds the Keycloak Tenant Controller container image.
func keycloakTenantControllerBuild() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN must be set")
	}

	appVersion, err := getChartAppVersion("keycloak-tenant-controller")
	if err != nil {
		return err
	}

	g0 := sh.OutCmd("git")
	commitID, err := g0("rev-parse", "HEAD")
	if err != nil {
		fmt.Printf("error running git rev-parse HEAD = %s", err)
	}
	gitarg := "KTC_GIT_COMMIT=" + commitID
	fmt.Printf("Git Arg = %s", gitarg)
	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--secret", "id=GITHUB_TOKEN,env=GITHUB_TOKEN",
		"--build-arg", strings.Trim(gitarg, ""),
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/keycloak-tenant-controller:"+appVersion,
		"--file", filepath.Join("keycloak-tenant-controller", "images", "Dockerfile"),
		"./keycloak-tenant-controller",
	)
}

// Builds the Nexus compiler container image.
func datamodelCompilerBuild() error {
	TAG := getNexusCompilerTag()
	// build compiler builder
	cmdBuilderBuild := fmt.Sprintf("cd nexus/compiler; DOCKER_REGISTRY=%s BUILDER_TAG=%s make docker.builder",
		OpenEdgePlatformContainerRegistry, TAG)

	if err := runCommand(cmdBuilderBuild); err != nil {
		return err
	}
	// build compiler
	cmdCompilerBuild := fmt.Sprintf(
		"cd nexus/compiler; DOCKER_REGISTRY=%s BUILDER_TAG=%s TAG=%s make docker",
		OpenEdgePlatformContainerRegistry, TAG, TAG)
	return runCommand(cmdCompilerBuild)
}

// Builds the Tenancy Datamodel container image.
func tenancyDatamodelBuild() error {
	projectDir := "tenancy-datamodel"
	nexusFile := "nexus.yaml"
	baseImage := "bitnami/kubectl:latest"

	nexusConf := struct {
		GroupName string `yaml:"groupName"`
	}{}

	yamlFile, err := os.ReadFile(filepath.Join(projectDir, nexusFile))
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(yamlFile, &nexusConf); err != nil {
		return err
	}

	appVersion, err := getChartAppVersion("tenancy-datamodel-init")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--build-arg", "DOCKER_BASE_IMAGE="+baseImage,
		"--build-arg", "IMAGE_NAME="+OpenEdgePlatformContainerRegistry+"/tenancy-datamodel:"+appVersion,
		"--build-arg", "NAME="+nexusConf.GroupName,
		"--tag", OpenEdgePlatformContainerRegistry+"/tenancy-datamodel:"+appVersion,
		"--file", filepath.Join(projectDir, "Dockerfile"),
		projectDir,
	)
}

// Builds the Tenancy API Mapping container image.
func tenancyAPIMappingBuild() error {
	// some errors below are deliberately ignored to suppress “file already/doesn’t” exist errors
	// Mage uses %v when formatting errors, so they cannot be unwrapped and handled on a case by case

	projectDir := "tenancy-api-mapping"
	homeDir := os.Getenv("HOME")

	linkFiles := []string{".gitconfig", ".netrc"}
	linkDirs := []string{".ssh"}

	mg.Deps(tenancyAPIMappingClean)

	for _, file := range linkFiles {
		// deliberately ignored errors
		_ = sh.RunV("touch", filepath.Join(projectDir, file))
		_ = sh.Copy(filepath.Join(projectDir, file), filepath.Join(homeDir, file))
	}

	for _, dir := range linkDirs {
		// deliberately ignored errors
		_ = sh.RunV("mkdir", "-p", filepath.Join(projectDir, dir))
		_ = sh.RunV("cp", "-r", filepath.Join(homeDir, dir), filepath.Join(projectDir, dir))
	}

	appVersion, err := getChartAppVersion("tenancy-api-remapping")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--tag", OpenEdgePlatformContainerRegistry+"/tenancy-api-mapping:"+appVersion,
		"--file", filepath.Join(projectDir, "Dockerfile"),
		projectDir,
	)
}

// Builds the Tenancy Manager container image.
func tenancyManagerBuild() error {
	// some errors below are deliberately ignored to suppress “file already/doesn’t” exist errors
	// Mage uses %v when formatting errors, so they cannot be unwrapped and handled on a case by case

	projectDir := "tenancy-manager"
	componentName := "tenancy-manager"
	homeDir := os.Getenv("HOME")

	linkFiles := []string{".gitconfig", ".netrc"}
	linkDirs := []string{".ssh"}

	mg.Deps(tenancyManagerClean)

	for _, file := range linkFiles {
		// deliberately ignored errors
		_ = sh.RunV("touch", filepath.Join(projectDir, file))
		_ = sh.Copy(filepath.Join(projectDir, file), filepath.Join(homeDir, file))
	}

	for _, dir := range linkDirs {
		// deliberately ignored errors
		_ = sh.RunV("mkdir", "-p", filepath.Join(projectDir, dir))
		_ = sh.RunV("cp", "-r", filepath.Join(homeDir, dir), filepath.Join(projectDir, dir))
	}

	appVersion, err := getChartAppVersion("tenancy-manager")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--build-arg", "TENANCY_MANAGER_COMPONENT_NAME="+componentName,
		"--tag", OpenEdgePlatformContainerRegistry+"/tenancy-manager:"+appVersion,
		"--file", filepath.Join(projectDir, "Dockerfile"),
		projectDir,
	)
}

// Builds the Nexus API Gateway container image.
func nexusAPIGatewayBuild() error {
	// some errors below are deliberately ignored to suppress “file already/doesn’t” exist errors
	// Mage uses %v when formatting errors, so they cannot be unwrapped and handled on a case by case

	projectDir := "nexus-api-gateway"
	componentName := "api-gw"
	homeDir := os.Getenv("HOME")

	linkFiles := []string{".gitconfig", ".netrc"}
	linkDirs := []string{".ssh"}

	mg.Deps(nexusAPIGatewayClean)

	appVersion, err := getChartAppVersion("nexus-api-gw")
	if err != nil {
		return err
	}

	for _, file := range linkFiles {
		// deliberately ignored errors
		_ = sh.RunV("touch", filepath.Join(projectDir, file))
		_ = sh.Copy(filepath.Join(projectDir, file), filepath.Join(homeDir, file))
	}

	for _, dir := range linkDirs {
		// deliberately ignored errors
		_ = sh.RunV("mkdir", "-p", filepath.Join(projectDir, dir))
		_ = sh.RunV("cp", "-r", filepath.Join(homeDir, dir), filepath.Join(projectDir, dir))
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--build-arg", "API_GW_COMPONENT_NAME="+componentName,
		"--tag", OpenEdgePlatformContainerRegistry+"/nexus-api-gw:"+appVersion,
		"--file", filepath.Join(projectDir, "Dockerfile"),
		projectDir,
	)
}

// Builds the openapi-generator container image.
func openAPIGeneratorBuild() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN must be set")
	}

	TAG := getNexusCompilerTag()

	g0 := sh.OutCmd("git")
	commitID, err := g0("rev-parse", "HEAD")
	if err != nil {
		fmt.Printf("error running git rev-parse HEAD = %s", err)
	}
	gitarg := "OPENAPI_GEN_GIT_COMMIT=" + commitID
	fmt.Printf("Git Arg = %s", gitarg)
	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--secret", "id=GITHUB_TOKEN,env=GITHUB_TOKEN",
		"--build-arg", strings.Trim(gitarg, ""),
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/nexus/openapi-generator:"+TAG,
		"--file", filepath.Join("nexus", "openapi-generator", "Dockerfile"),
		"./nexus",
	)
}
