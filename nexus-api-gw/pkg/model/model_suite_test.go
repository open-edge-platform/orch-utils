// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package model_test

import (
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func TestModel(t *testing.T) {
	log.StandardLogger().ExitFunc = nil
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Model Test Suite")
}
