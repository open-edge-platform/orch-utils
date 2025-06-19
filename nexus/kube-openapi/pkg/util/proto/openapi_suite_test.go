// Copyright 2017 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package proto_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2/config"
	. "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"fmt"
	"testing"
)

func TestOpenapi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Openapi Suite", []Reporter{newlineReporter{}})
}

// Print a newline after the default newlineReporter due to issue
// https://github.com/jstemmer/go-junit-report/issues/31
type newlineReporter struct{}

func (newlineReporter) SpecSuiteWillBegin(config GinkgoConfigType, summary *SuiteSummary) {}

func (newlineReporter) BeforeSuiteDidRun(setupSummary *SetupSummary) {}

func (newlineReporter) AfterSuiteDidRun(setupSummary *SetupSummary) {}

func (newlineReporter) SpecWillRun(specSummary *SpecSummary) {}

func (newlineReporter) SpecDidComplete(specSummary *SpecSummary) {}

// SpecSuiteDidEnd Prints a newline between "35 Passed | 0 Failed | 0 Pending | 0 Skipped" and "--- PASS:"
func (newlineReporter) SpecSuiteDidEnd(summary *SuiteSummary) { fmt.Printf("\n") }
