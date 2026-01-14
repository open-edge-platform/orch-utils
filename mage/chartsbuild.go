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

// ChartsBuild package all helm charts in the charts directory.
func ChartsBuild() error {
	chartsDir := "charts"
	err := filepath.Walk(chartsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && strings.Count(path, string(os.PathSeparator)) ==
			strings.Count(chartsDir, string(os.PathSeparator))+1 {
			fmt.Println("Packaging chart:", path)
			buildDir := filepath.Join("chartsBuildDir", filepath.Base(path))
			if err := os.MkdirAll(buildDir, os.ModePerm); err != nil {
				return fmt.Errorf("error creating build directory for chart %s: %w", path, err)
			}
			if strings.Contains(path, "umbrella") {
				return nil
			}
			if err := sh.RunV("helm", "package", path, "--destination", buildDir); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
