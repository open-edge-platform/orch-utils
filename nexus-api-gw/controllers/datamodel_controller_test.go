// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = ginkgo.Describe("Datamodel controller", func() {
	ginkgo.It("should create datamodel crd", func() {
		gvr := schema.GroupVersionResource{
			Group:    "nexus.com",
			Version:  "v1",
			Resource: "datamodels",
		}

		unstructuredObject := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "nexus.com/v1",
				"kind":       "Datamodel",
				"metadata": map[string]interface{}{
					"name": "nexus.com",
				},
				"spec": map[string]interface{}{
					"name":  "nexus.com",
					"title": "Example title",
				},
			},
		}
		_, err := dynamicClient.Resource(gvr).Create(context.TODO(), &unstructuredObject, metav1.CreateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		gomega.Eventually(func() bool {
			if _, ok := model.GetDatamodel("nexus.com"); ok {
				return true
			}
			return false
		}).Should(gomega.BeTrue())
	})
})
