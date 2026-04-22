// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import "github.com/magefile/mage/mg"

// Binary targets build the Go binary locally.
type Binary mg.Namespace

// Build compiles the tenancy-manager binary to ./bin/tenancy-manager.
func (Binary) Build() error {
	return goBuild()
}

// Build targets build container images.
type Build mg.Namespace

// Docker builds the tenancy-manager Docker image, versioned from the Helm chart appVersion.
func (Build) Docker() error {
	return dockerBuild()
}

// Test targets run test suites.
type Test mg.Namespace

// Unit runs the Go unit tests (no database required).
func (Test) Unit() error {
	return runUnitTests()
}

// Integration runs the functional integration tests (requires docker).
func (Test) Integration() error {
	mg.Deps(Binary.Build)
	return runIntegrationTests()
}
