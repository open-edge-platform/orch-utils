// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-edge-platform/orch-utils/component-status/pkg/config"
	"github.com/open-edge-platform/orch-utils/component-status/pkg/handler"
)

var _ = Describe("Status Handler", func() {
	var (
		cfg            *config.Config
		statusHandler  *handler.StatusHandler
		responseWriter *httptest.ResponseRecorder
	)

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
				},
			},
		}
		statusHandler = handler.NewStatusHandler(cfg)
		responseWriter = httptest.NewRecorder()
	})

	Describe("GetStatus", func() {
		It("should return component status successfully", func() {
			req, err := http.NewRequest("GET", "/v1/orchestrator", nil)
			Expect(err).ToNot(HaveOccurred())

			statusHandler.GetStatus(responseWriter, req)

			Expect(responseWriter.Code).To(Equal(http.StatusOK))
			Expect(responseWriter.Header().Get("Content-Type")).To(Equal("application/json"))

			var response config.Config
			err = json.Unmarshal(responseWriter.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response.SchemaVersion).To(Equal("1.0"))
			Expect(response.Orchestrator.Version).To(Equal("2026.0"))
		})

		It("should return method not allowed for non-GET requests", func() {
			req, err := http.NewRequest("POST", "/v1/orchestrator", nil)
			Expect(err).ToNot(HaveOccurred())

			statusHandler.GetStatus(responseWriter, req)

			Expect(responseWriter.Code).To(Equal(http.StatusMethodNotAllowed))
		})
	})

	Describe("HealthCheck", func() {
		It("should return healthy status", func() {
			req, err := http.NewRequest("GET", "/healthz", nil)
			Expect(err).ToNot(HaveOccurred())

			http.HandlerFunc(handler.HealthCheck).ServeHTTP(responseWriter, req)

			Expect(responseWriter.Code).To(Equal(http.StatusOK))

			var response map[string]string
			err = json.Unmarshal(responseWriter.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response["status"]).To(Equal("healthy"))
		})
	})

	Describe("ReadyCheck", func() {
		It("should return ready status", func() {
			req, err := http.NewRequest("GET", "/readyz", nil)
			Expect(err).ToNot(HaveOccurred())

			http.HandlerFunc(handler.ReadyCheck).ServeHTTP(responseWriter, req)

			Expect(responseWriter.Code).To(Equal(http.StatusOK))

			var response map[string]string
			err = json.Unmarshal(responseWriter.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response["status"]).To(Equal("ready"))
		})
	})
})
