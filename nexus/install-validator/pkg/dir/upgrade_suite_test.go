// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dir

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade Suite")
}
