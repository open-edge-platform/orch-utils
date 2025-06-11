// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"github.com/magefile/mage/sh"
)

func (Test) golang() error {
	return sh.RunV(
		"make",
		"ginkgo",
	)
}
