// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

//go:build mage

package main

import (
	// mage:import
	. "github.com/open-edge-platform/orch-utils/mage" //nolint: revive
)

// To silence compiler's unused import error.
var _ = Lint{}
