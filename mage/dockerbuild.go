// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
	"gopkg.in/yaml.v3"
)

// Builds the secrets-config container image.
func secretsConfigBuild() error {
	appVersion, err := getChartAppVersion("secrets-config")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
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
	appVersion, err := getChartAppVersion("aws-sm-proxy")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
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
	appVersion, err := getChartAppVersion("token-fs")
	if err != nil {
		return err
	}

	return sh.RunV(
		"docker",
		"build",
		"--load",
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

func componentStatusBuild() error {
	appVersion, err := getChartAppVersion("component-status")
	if err != nil {
		return err
	}

	g0 := sh.OutCmd("git")
	commitID, err := g0("rev-parse", "HEAD")
	if err != nil {
		fmt.Printf("error running git rev-parse HEAD = %s", err)
	}
	commitID = strings.TrimSpace(commitID)

	return sh.RunV(
		"docker",
		"build",
		"--load",
		"--build-arg", "org_oci_version="+appVersion,
		"--build-arg", "org_oci_revision="+commitID,
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", OpenEdgePlatformContainerRegistry+"/component-status:"+appVersion,
		"--file", filepath.Join("component-status", "Dockerfile"),
		"./component-status",
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

// Builds the Tenancy Manager container image.
func tenancyManagerBuild() error {
	// some errors below are deliberately ignored to suppress “file already/doesn’t” exist errors
	// Mage uses %v when formatting errors, so they cannot be unwrapped and handled on a case by case

	projectDir := "tenancy-manager"
	componentName := "tenancy-manager"

	appVersion, err := getChartAppVersion(projectDir)
	if err != nil {
		return err
	}

	// run go mod vendor in project directory
	if err := sh.RunV("sh", "-c", fmt.Sprintf("cd %s && go mod vendor", projectDir)); err != nil {
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

func listContainers() error {
	return listTaggedContainers()
}

func listTaggedContainers() error {
	fmt.Print("images:\n")

	images := []string{
		"auth-service",
		"aws-sm-proxy",
		"cert-synchronizer",
		"keycloak-tenant-controller",
		"secrets-config",
		"squid-proxy",
		"tenancy-manager",
		"token-fs",
	}

	for _, image := range images {
		imagever, err := getChartAppVersion(image)
		if err != nil {
			return err
		}

		fmt.Printf("  %s:\n", image)
		fmt.Printf("    tag: '%s'\n", OpenEdgePlatformContainerRegistry+"/"+image+":"+imagever)
		fmt.Printf("    version: '%s'\n", imagever)
		fmt.Printf("    gitTagPrefix: '%s/v'\n", image)
		fmt.Printf("    buildTarget: 'docker-build-%s'\n", image)
	}

	return nil
}
