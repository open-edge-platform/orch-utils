// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/utils"
)

var _ = ginkgo.Describe("Common tests", func() {
	ginkgo.It("should get correct datamodel name from crd", func() {
		datamodelName := utils.GetDatamodelName("route.route.admin.nexus.com")
		gomega.Expect(datamodelName).To(gomega.Equal("admin.nexus.com"))
	})

	ginkgo.It("should check if file exist", func() {
		file, err := os.Create("test-file.txt")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		check := utils.IsFileExists(file.Name())
		gomega.Expect(check).To(gomega.BeTrue())

		err = os.Remove("test-file.txt")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.It("should check if file not exist", func() {
		check := utils.IsFileExists("non-existent-file")
		gomega.Expect(check).To(gomega.BeFalse())
	})

	ginkgo.It("should check if server config is valid", func() {
		isValid := utils.IsServerConfigValid(&config.Config{
			Server: config.ServerConfig{
				Address:  "address",
				CertPath: "cert_path",
				KeyPath:  "key_path",
			},
		})
		gomega.Expect(isValid).To(gomega.BeTrue())
	})

	ginkgo.It("should check if server config is not valid", func() {
		isValid := utils.IsServerConfigValid(&config.Config{})
		gomega.Expect(isValid).To(gomega.BeFalse())
	})

	ginkgo.It("should get crd type", func() {
		crdType := utils.GetCrdType("Test", "vmware.org")
		gomega.Expect(crdType).To(gomega.Equal("tests.vmware.org"))
	})

	ginkgo.It("should get resource name", func() {
		resource := utils.GetGroupResourceName("Test")
		gomega.Expect(resource).To(gomega.Equal("tests"))
	})
})
