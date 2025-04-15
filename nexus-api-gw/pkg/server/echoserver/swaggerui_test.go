// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package echoserver_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/server/echoserver"
)

var _ = ginkgo.Describe("Swagger handler tests", ginkgo.Ordered, func() {
	ginkgo.It("should ensure default for swagger ui opts", func() {
		opts := echoserver.SwaggerUIOpts{}
		opts.EnsureDefaults()

		gomega.Expect(opts.BasePath).To(gomega.Equal("/"))
		gomega.Expect(opts.Path).To(gomega.Equal("docs"))
		gomega.Expect(opts.SpecURL).To(gomega.Equal("/swagger.json"))
		gomega.Expect(opts.Title).To(gomega.Equal("API documentation"))
	})

	ginkgo.It("should test swagger handler", func() {
		e := echo.New()
		e.GET("/:datamodel/docs", echoserver.SwaggerUI)

		req, err := http.NewRequest(http.MethodGet, "/docs", http.NoBody)
		gomega.Expect(err).To(gomega.BeNil())

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/test/docs")
		c.SetParamNames("datamodel")
		c.SetParamValues("test")

		err = echoserver.SwaggerUI(c)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		gomega.Expect(rec.Header().Get("Content-Type")).To(gomega.Equal("text/html; charset=UTF-8"))
	})
})
