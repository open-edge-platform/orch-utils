// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package model_test

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = ginkgo.Describe("Construct tests", func() {
	ginkgo.It("should construct new datamodel (vmware-test.org)", func() {
		unstructuredObj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"spec": map[string]interface{}{
					"title": "VMWare Datamodel",
				},
			},
		}

		model.ConstructDatamodel(model.Upsert, "vmware-test.org", &unstructuredObj)
		gomega.Expect(model.DatamodelToDatamodelInfo).To(gomega.HaveKey("vmware-test.org"))
	})

	ginkgo.It("should delete datamodel vmware-test.org", func() {
		unstructuredObj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"spec": map[string]interface{}{
					"title": "VMWare Datamodel",
				},
			},
		}

		model.ConstructDatamodel(model.Delete, "vmware-test.org", &unstructuredObj)
		gomega.Expect(model.DatamodelToDatamodelInfo).ToNot(gomega.HaveKey("vmware-test.org"))
	})
})
