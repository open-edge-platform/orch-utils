// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"fmt"

	"github.com/bitfield/script"
	"github.com/magefile/mage/sh"
)

func tryToCreateECRRepository(repositoryName string) error { //nolint:unparam // explicitly ignoring the error
	cmd := fmt.Sprintf(
		"aws ecr create-repository --region %s --repository-name %s",
		AWSRegion, repositoryName,
	)
	if _, err := script.Exec(cmd).Stdout(); err != nil {
		fmt.Printf("ignoring error creating ECR repository %s: %v\n", repositoryName, err)
	}
	return nil
}

func inspectAndPushImage(registry, imageName, appVersion string) error {
	cmd := fmt.Sprintf(
		"docker manifest inspect %s/%s:%s",
		registry,
		imageName,
		appVersion,
	)
	_, err := script.Exec(cmd).Stdout()
	if err != nil {
		fmt.Printf("attempting push after finding error during docker manifest inspect for %s %s: %v\n",
			imageName, appVersion, err)
		return sh.RunV(
			"docker",
			"push",
			fmt.Sprintf("%s/%s:%s", registry, imageName, appVersion),
		)
	}
	fmt.Printf("docker manifest inspect for %s %s did not return error. Skipping push.\n",
		imageName, appVersion)
	return nil
}

func pushImage(imageName string, chartName string) error {
	appVersion, err := getChartAppVersion(chartName)
	if err != nil {
		return fmt.Errorf("getting chart app version for %s: %w", chartName, err)
	}
	// TODO: Do this better
	if err := tryToCreateECRRepository(fmt.Sprintf("edge-orch/common/%s", imageName)); err != nil {
		return fmt.Errorf("creating ECR repository for %s: %w", imageName, err)
	}
	return inspectAndPushImage(OpenEdgePlatformContainerRegistry, imageName, appVersion)
}

func pushNexusCompilerImage() error {
	appVersion := getNexusVersion()
	imageName := "nexus/compiler/amd64"
	registry := OpenEdgePlatformContainerRegistry

	return inspectAndPushImage(registry, imageName, appVersion)
}

func pushOpenAPIGeneratorImage() error {
	appVersion := getNexusVersion()
	imageName := "nexus/openapi-generator"
	registry := OpenEdgePlatformContainerRegistry

	return inspectAndPushImage(registry, imageName, appVersion)
}
