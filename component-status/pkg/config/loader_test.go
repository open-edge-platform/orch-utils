// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-edge-platform/orch-utils/component-status/pkg/config"
)

var _ = Describe("Config Loader", func() {
	Describe("Load", func() {
		It("should load a valid config file", func() {
			content := `schema-version: "1.0"
orchestrator:
  version: "2026.0"
  features:
    - name: application-orchestration
      status: enabled
    - name: edge-infrastructure-manager
      status: enabled
      features:
        - name: inventory
          status: enabled
        - name: device-onboarding
          status: disabled
`

			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.Write([]byte(content))
			Expect(err).ToNot(HaveOccurred())
			err = tmpfile.Close()
			Expect(err).ToNot(HaveOccurred())

			cfg, err := config.Load(tmpfile.Name())
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.SchemaVersion).To(Equal("1.0"))
			Expect(cfg.Orchestrator.Version).To(Equal("2026.0"))
			Expect(cfg.Orchestrator.Features).To(HaveLen(2))

			appOrch := cfg.Orchestrator.Features[0]
			Expect(appOrch.Name).To(Equal("application-orchestration"))
			Expect(appOrch.Status).To(Equal("enabled"))

			eim := cfg.Orchestrator.Features[1]
			Expect(eim.Name).To(Equal("edge-infrastructure-manager"))
			Expect(eim.Status).To(Equal("enabled"))
			Expect(eim.Features).To(HaveLen(2))

			inventory := eim.Features[0]
			Expect(inventory.Name).To(Equal("inventory"))
			Expect(inventory.Status).To(Equal("enabled"))

			deviceOnboarding := eim.Features[1]
			Expect(deviceOnboarding.Name).To(Equal("device-onboarding"))
			Expect(deviceOnboarding.Status).To(Equal("disabled"))
		})

		It("should return error for non-existent file", func() {
			_, err := config.Load("/nonexistent/file.yaml")
			Expect(err).To(HaveOccurred())
		})

		It("should return error for invalid YAML", func() {
			content := `invalid: yaml: content:`

			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.Write([]byte(content))
			Expect(err).ToNot(HaveOccurred())
			err = tmpfile.Close()
			Expect(err).ToNot(HaveOccurred())

			_, err = config.Load(tmpfile.Name())
			Expect(err).To(HaveOccurred())
		})

		Context("missing required fields", func() {
			It("should return error for missing schema-version", func() {
				content := `orchestrator:
  version: "2026.0"
  features: []
`
				tmpfile, err := os.CreateTemp("", "config-*.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer os.Remove(tmpfile.Name())

				_, err = tmpfile.Write([]byte(content))
				Expect(err).ToNot(HaveOccurred())
				err = tmpfile.Close()
				Expect(err).ToNot(HaveOccurred())

				_, err = config.Load(tmpfile.Name())
				Expect(err).To(HaveOccurred())
			})

			It("should return error for missing orchestrator.version", func() {
				content := `schema-version: "1.0"
orchestrator:
  features: []
`
				tmpfile, err := os.CreateTemp("", "config-*.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer os.Remove(tmpfile.Name())

				_, err = tmpfile.Write([]byte(content))
				Expect(err).ToNot(HaveOccurred())
				err = tmpfile.Close()
				Expect(err).ToNot(HaveOccurred())

				_, err = config.Load(tmpfile.Name())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("IsFeatureInstalled", func() {
		var cfg *config.Config

		BeforeEach(func() {
			cfg = &config.Config{
				SchemaVersion: "1.0",
				Orchestrator: config.Orchestrator{
					Version: "2026.0",
					Features: []config.Feature{
						{
							Name:   "application-orchestration",
							Status: "enabled",
						},
						{
							Name:   "edge-infrastructure-manager",
							Status: "enabled",
							Features: []config.Feature{
								{
									Name:   "inventory",
									Status: "enabled",
								},
								{
									Name:   "device-onboarding",
									Status: "disabled",
								},
							},
						},
						{
							Name:   "cluster-orchestration",
							Status: "disabled",
						},
					},
				},
			}
		})

		It("should return true for installed top-level feature", func() {
			Expect(cfg.IsFeatureInstalled("application-orchestration")).To(BeTrue())
		})

		It("should return false for disabled top-level feature", func() {
			Expect(cfg.IsFeatureInstalled("cluster-orchestration")).To(BeFalse())
		})

		It("should return true for installed nested feature", func() {
			Expect(cfg.IsFeatureInstalled("edge-infrastructure-manager", "inventory")).To(BeTrue())
		})

		It("should return false for disabled nested feature", func() {
			Expect(cfg.IsFeatureInstalled("edge-infrastructure-manager", "device-onboarding")).To(BeFalse())
		})

		It("should return false for non-existent feature", func() {
			Expect(cfg.IsFeatureInstalled("non-existent")).To(BeFalse())
		})

		It("should return false for non-existent nested feature", func() {
			Expect(cfg.IsFeatureInstalled("edge-infrastructure-manager", "non-existent")).To(BeFalse())
		})

		It("should return false for empty path", func() {
			Expect(cfg.IsFeatureInstalled()).To(BeFalse())
		})
	})
})

