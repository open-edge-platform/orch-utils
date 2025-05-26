/*
 * Copyright (C) 2025 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main_test

import (
	"os"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	main "github.com/open-edge-platform/orch-utils/tenancy-api-mapping/cmd/specgen"
	"github.com/open-edge-platform/orch-utils/tenancy-api-mapping/pkg/config"
)

func TestMain(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Main Suite")
}

var _ = ginkgo.Describe("Main", func() {
	var cfg *config.Config
	var err error
	ginkgo.AfterEach(func() {
		if cfg != nil {
			os.RemoveAll(cfg.Global.LocalSubModsDir)
			os.RemoveAll(cfg.Global.SpecOutputDir)
		}
	})
	ginkgo.Context("when running the application", func() {
		ginkgo.It("should not return an error with valid configuration", func() {
			confPath := "../../tests/valid_config.yaml"
			cfg, err = config.LoadConfig(confPath)
			gomega.Expect(err).To(gomega.BeNil())

			err = main.Run(confPath)
			gomega.Expect(err).To(gomega.BeNil())
		})

		ginkgo.It("should return an error with invalid configuration", func() {
			err := main.Run("invalid_config.yaml")
			gomega.Expect(err).ToNot(gomega.BeNil())
		})

		// Additional tests can be added here
	})
})
