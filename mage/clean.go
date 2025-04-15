// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"github.com/magefile/mage/sh"
)

// removes a list of files or directories.
func deleteFiles(files []string) error {
	for _, file := range files {
		if err := sh.Rm(file); err != nil {
			return err
		}
	}

	return nil
}

func tenancyAPIMappingClean() error {
	filesToDelete := []string{
		"tenancy-api-mapping/.gitconfig",
		"tenancy-api-mapping/.netrc",
		"tenancy-api-mapping/.ssh/",
	}

	return deleteFiles(filesToDelete)
}

func tenancyManagerClean() error {
	filesToDelete := []string{
		"tenancy-manager/.gitconfig",
		"tenancy-manager/.netrc",
		"tenancy-manager/.ssh/",
	}

	return deleteFiles(filesToDelete)
}

func nexusAPIGatewayClean() error {
	filesToDelete := []string{
		"nexus-api-gw/.gitconfig",
		"nexus-api-gw/.netrc",
		"nexus-api-gw/.ssh/",
	}

	return deleteFiles(filesToDelete)
}
