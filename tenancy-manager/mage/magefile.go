// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import "github.com/magefile/mage/mg"

// Build targets build artefacts — the local binary and the container image.
type Build mg.Namespace

// Binary compiles the tenancy-manager binary to ./bin/tenancy-manager.
func (Build) Binary() error {
	return goBuild()
}

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

// Fuzz runs the fuzzing engine against every Fuzz* target in internal/api
// for the given number of minutes per target. Example: mage test:fuzz 5
func (Test) Fuzz(minutes int) error {
	return runFuzzTests(minutes)
}

// Integration runs the functional integration tests (requires docker).
func (Test) Integration() error {
	mg.Deps(Build.Binary)
	return runIntegrationTests()
}
