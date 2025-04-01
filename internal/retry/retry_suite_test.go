// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package retry_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRetry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Retry Suite")
}
