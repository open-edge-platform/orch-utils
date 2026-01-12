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
    application-orchestration:
      installed: true
    edge-infrastructure-manager:
      installed: true
      inventory:
        installed: true
      device-onboarding:
        installed: false
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

			appOrch, exists := cfg.Orchestrator.Features["application-orchestration"]
			Expect(exists).To(BeTrue())
			Expect(appOrch.Installed).To(BeTrue())

			eim, exists := cfg.Orchestrator.Features["edge-infrastructure-manager"]
			Expect(exists).To(BeTrue())
			Expect(eim.Installed).To(BeTrue())
			Expect(eim.SubFeatures).To(HaveLen(2))

			inventory, exists := eim.SubFeatures["inventory"]
			Expect(exists).To(BeTrue())
			Expect(inventory.Installed).To(BeTrue())

			deviceOnboarding, exists := eim.SubFeatures["device-onboarding"]
			Expect(exists).To(BeTrue())
			Expect(deviceOnboarding.Installed).To(BeFalse())
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
  features: {}
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
  features: {}
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
					Features: map[string]config.Feature{
						"application-orchestration": {
							Installed:   true,
							SubFeatures: map[string]config.Feature{},
						},
						"edge-infrastructure-manager": {
							Installed: true,
							SubFeatures: map[string]config.Feature{
								"inventory": {
									Installed:   true,
									SubFeatures: map[string]config.Feature{},
								},
								"device-onboarding": {
									Installed:   false,
									SubFeatures: map[string]config.Feature{},
								},
							},
						},
						"cluster-orchestration": {
							Installed:   false,
							SubFeatures: map[string]config.Feature{},
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

