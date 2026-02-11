// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestHTTPConfig(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "HTTP Config Suite")
}

var _ = ginkgo.Describe("HTTPServerConfig", func() {
	ginkgo.Context("GetHTTPServerConfig", func() {
		ginkgo.BeforeEach(func() {
			// Clear environment variables before each test
			os.Unsetenv("HTTP_SERVER_ENABLED")
			os.Unsetenv("HTTP_SERVER_HOST")
			os.Unsetenv("HTTP_SERVER_PORT")
			os.Unsetenv("HTTP_TLS_ENABLED")
		})

		ginkgo.When("all environment variables are set", func() {
			ginkgo.It("should return config with all values", func() {
				os.Setenv("HTTP_SERVER_ENABLED", "true")
				os.Setenv("HTTP_SERVER_HOST", "0.0.0.0")
				os.Setenv("HTTP_SERVER_PORT", "8080")
				os.Setenv("HTTP_TLS_ENABLED", "true")

				config := GetHTTPServerConfig()

				gomega.Expect(config.Enabled).To(gomega.BeTrue())
				gomega.Expect(config.Host).To(gomega.Equal("0.0.0.0"))
				gomega.Expect(config.Port).To(gomega.Equal("8080"))
				gomega.Expect(config.TLSEnabled).To(gomega.BeTrue())
			})
		})

		ginkgo.When("enabled is set to false", func() {
			ginkgo.It("should return config with enabled false", func() {
				os.Setenv("HTTP_SERVER_ENABLED", "false")
				os.Setenv("HTTP_SERVER_HOST", "localhost")
				os.Setenv("HTTP_SERVER_PORT", "9090")

				config := GetHTTPServerConfig()

				gomega.Expect(config.Enabled).To(gomega.BeFalse())
				gomega.Expect(config.Host).To(gomega.Equal("localhost"))
				gomega.Expect(config.Port).To(gomega.Equal("9090"))
			})
		})

		ginkgo.When("no environment variables are set", func() {
			ginkgo.It("should return config with default values", func() {
				config := GetHTTPServerConfig()

				gomega.Expect(config.Enabled).To(gomega.BeTrue()) // Default is true
				gomega.Expect(config.Host).To(gomega.Equal("0.0.0.0")) // Default is 0.0.0.0
				gomega.Expect(config.Port).To(gomega.Equal("8080")) // Default is 8080
				gomega.Expect(config.TLSEnabled).To(gomega.BeFalse()) // Default is false
			})
		})

		ginkgo.When("only some environment variables are set", func() {
			ginkgo.It("should return config with mixed values", func() {
				os.Setenv("HTTP_SERVER_ENABLED", "true")
				os.Setenv("HTTP_SERVER_PORT", "3000")
				// HTTP_SERVER_HOST and HTTP_TLS_ENABLED not set

				config := GetHTTPServerConfig()

				gomega.Expect(config.Enabled).To(gomega.BeTrue())
				gomega.Expect(config.Host).To(gomega.Equal("0.0.0.0")) // Should use default
				gomega.Expect(config.Port).To(gomega.Equal("3000"))
				gomega.Expect(config.TLSEnabled).To(gomega.BeFalse()) // Should use default
			})
		})

		ginkgo.When("enabled has invalid value", func() {
			ginkgo.It("should treat as false", func() {
				os.Setenv("HTTP_SERVER_ENABLED", "invalid")
				os.Setenv("HTTP_SERVER_HOST", "127.0.0.1")
				os.Setenv("HTTP_SERVER_PORT", "8080")

				config := GetHTTPServerConfig()

				gomega.Expect(config.Enabled).To(gomega.BeFalse())
				gomega.Expect(config.Host).To(gomega.Equal("127.0.0.1"))
				gomega.Expect(config.Port).To(gomega.Equal("8080"))
			})
		})

		ginkgo.When("TLS enabled has invalid value", func() {
			ginkgo.It("should treat as false", func() {
				os.Setenv("HTTP_SERVER_ENABLED", "true")
				os.Setenv("HTTP_TLS_ENABLED", "not-a-boolean")

				config := GetHTTPServerConfig()

				gomega.Expect(config.Enabled).To(gomega.BeTrue())
				gomega.Expect(config.TLSEnabled).To(gomega.BeFalse())
			})
		})

		ginkgo.When("using different port numbers", func() {
			ginkgo.It("should accept various valid ports", func() {
				testCases := []struct {
					port     string
					expected string
				}{
					{"80", "80"},
					{"443", "443"},
					{"8080", "8080"},
					{"8443", "8443"},
					{"3000", "3000"},
				}

				for _, tc := range testCases {
					os.Setenv("HTTP_SERVER_PORT", tc.port)
					config := GetHTTPServerConfig()
					gomega.Expect(config.Port).To(gomega.Equal(tc.expected))
				}
			})
		})

		ginkgo.When("using different host addresses", func() {
			ginkgo.It("should accept various valid hosts", func() {
				testCases := []struct {
					host     string
					expected string
				}{
					{"0.0.0.0", "0.0.0.0"},
					{"127.0.0.1", "127.0.0.1"},
					{"localhost", "localhost"},
					{"192.168.1.1", "192.168.1.1"},
				}

				for _, tc := range testCases {
					os.Setenv("HTTP_SERVER_HOST", tc.host)
					config := GetHTTPServerConfig()
					gomega.Expect(config.Host).To(gomega.Equal(tc.expected))
				}
			})
		})
	})
})
