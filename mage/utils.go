// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func getNexusVersion() string {
	data, err := os.ReadFile("./nexus/VERSION")
	if err != nil {
		fmt.Printf("Error reading nexus/VERSION file: %v\n", err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(data))
}

func runCommand(cmd string) error {
	fmt.Println("Running command:", cmd)
	c := exec.Command("sh", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
