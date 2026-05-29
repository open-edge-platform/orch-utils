//go:build mage

// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	// mage:import
	. "github.com/open-edge-platform/orch-utils/tenancy-manager/mage" //nolint:revive
)

// Silence the compiler's unused-import error for the dot import.
var _ = Build{}
