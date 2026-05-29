// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// runUnitTests runs the Go unit tests for cmd/ and internal/ without a database.
func runUnitTests() error {
	cmd := exec.Command("go", "test", "-race", "-count=1", "./cmd/...", "./internal/...")
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPRIVATE=github.com/open-edge-platform/*")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runIntegrationTests executes tests/integration/run.sh with --skip-build
// (Binary:Build is declared as a dependency in the Test:Integration target).
func runIntegrationTests() error {
	script := filepath.Join("tests", "integration", "run.sh")
	cmd := exec.Command("bash", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runFuzzTests runs the fuzzing engine against every Fuzz* target in
// internal/api for the given number of minutes per target.
func runFuzzTests(minutes int) error {
	fuzzTime := fmt.Sprintf("%dm", minutes)

	targets := []string{
		"FuzzDecodeBody",
		"FuzzEventQueryParams",
		"FuzzOrgRouting",
		"FuzzProjectRouting",
	}

	for _, target := range targets {
		fmt.Printf("fuzzing %s for %s...\n", target, fuzzTime)
		cmd := exec.Command("go", "test",
			"-fuzz="+target,
			"-fuzztime="+fuzzTime,
			"./internal/api/...",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", target, err)
		}
	}
	return nil
}
