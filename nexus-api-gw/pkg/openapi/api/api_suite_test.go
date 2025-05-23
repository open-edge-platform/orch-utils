// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

const (
	URI         = "/v1alpha1/project/{projectId}/global-namespaces"
	ResourceURI = "/v1alpha1/project/{projectId}/global-namespaces/{id}"
	ListURI     = "/v1alpha1/global-namespaces/test"
)

func TestApi(t *testing.T) {
	log.StandardLogger().ExitFunc = nil
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Declarative Suite")
}
