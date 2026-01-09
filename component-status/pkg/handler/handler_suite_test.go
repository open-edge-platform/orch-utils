// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handler Test Suite")
}
