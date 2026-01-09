// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Test Suite")
}
