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

	"github.com/magefile/mage/sh"
)

// Pushes all helm chart .tgz files as OCI artifacts to the specified OCI registry.
func pushCharts(ociRegistry string) error { //nolint: cyclop
	chartsDir := "charts"
	chartsBuildDir := "chartsBuildDir"

	// Get all chart directories in charts directory
	entries, err := os.ReadDir(chartsDir)
	if err != nil {
		return fmt.Errorf("reading charts directory: %w", err)
	}

	// Create the parent ECR repository
	if err := tryToCreateECRRepository("edge-orch/common/charts"); err != nil {
		return fmt.Errorf("creating ECR repository for %s: %w", ociRegistry, err)
	}

	// Create ECR repositories for each chart
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		fmt.Println("Creating ECR repository for", entry.Name())

		if err := tryToCreateECRRepository("edge-orch/common/charts/" + entry.Name()); err != nil {
			return fmt.Errorf("creating ECR repository for %s: %w", entry.Name(), err)
		}
	}

	// Recursively find and push all .tgz files in charts directory
	if err := filepath.Walk(chartsBuildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tgz") {
			chartTgzPath := path

			chartName := strings.Split(chartTgzPath, "/")[1]

			chartVersion, err := getChartVersion(chartName)
			if err != nil {
				return fmt.Errorf("getting chart version for %s: %w", chartName, err)
			}

			registryPath := fmt.Sprintf("oci://%s/", ociRegistry)

			// Skip if the chart already exists in the registry
			if _, err := sh.Output(
				"helm",
				"show",
				"chart",
				"--version", chartVersion,
				registryPath+chartName,
			); err == nil {
				fmt.Printf("Chart %s already exists in OCI registry %s, skipping\n", chartTgzPath, registryPath)
				return nil
			}

			fmt.Printf("Pushing chart %s to OCI registry %s\n", chartTgzPath, registryPath)
			if err := sh.RunV("helm", "push", chartTgzPath, registryPath); err != nil {
				return fmt.Errorf("error pushing chart %s: %w", chartTgzPath, err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walking charts directory: %w", err)
	}

	return nil
}

func pushSpecificCharts(charts []string, ociRegistry string) error {
	for _, chart := range charts {
		chartBuildDir := fmt.Sprintf("charts/%s/build/", chart)
		err := filepath.Walk(chartBuildDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".tgz") {
				chartTgzPath := path
				repoName := strings.TrimSuffix(info.Name(), ".tgz")
				if err := tryToCreateECRRepository(repoName); err != nil {
					return fmt.Errorf("creating ECR repository for %s: %w", repoName, err)
				}
				fmt.Printf("Pushing chart %s to OCI registry %s\n", chartTgzPath, ociRegistry)
				if err := sh.RunV("helm", "push", chartTgzPath, ociRegistry); err != nil {
					return fmt.Errorf("error pushing chart %s: %w", chartTgzPath, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
