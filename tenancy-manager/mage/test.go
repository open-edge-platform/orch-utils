// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"os"
	"os/exec"
	"path/filepath"
)

// runIntegrationTests executes tests/integration/run.sh with --skip-build
// (Binary:Build is declared as a dependency in the Test:Integration target).
func runIntegrationTests() error {
	script := filepath.Join("tests", "integration", "run.sh")
	cmd := exec.Command("bash", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
