// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package declarative_test

import (
	"net/http"
	"os"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/declarative"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/server/echoserver"
	nexusClient "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"k8s.io/client-go/kubernetes"
)

var _ = ginkgo.Describe("OpenAPI tests", ginkgo.Ordered, func() {
	ginkgo.It("should setup and load openapi file", func() {
		openAPISpecFile := "testFile"
		f, err := os.Create(openAPISpecFile)
		defer os.RemoveAll(openAPISpecFile)
		gomega.Expect(err).To(gomega.BeNil())
		err = f.Sync()
		gomega.Expect(err).To(gomega.BeNil())
		defer f.Close()
		bytesWritten, err := f.Write(spec)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(bytesWritten).ToNot(gomega.Equal(0))
		err = f.Sync()
		gomega.Expect(err).To(gomega.BeNil())
		err = declarative.Setup(openAPISpecFile)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(declarative.Paths).To(gomega.HaveKey(URI))
		gomega.Expect(declarative.Paths).To(gomega.HaveKey(ResourceURI))
	})

	ginkgo.It("should add resource get operation uri to apis list", func() {
		ec := declarative.SetupContext(URI, http.MethodGet, declarative.Paths[URI].Get)
		declarative.AddApisEndpoint(ec)

		gomega.Expect(declarative.ApisList).To(gomega.HaveKey(ec.URI))
		gomega.Expect(declarative.ApisList[ec.URI]).To(gomega.HaveKey(http.MethodGet))
		gomega.Expect(declarative.ApisList[ec.URI]).ToNot(gomega.HaveKey(http.MethodPost))
		gomega.Expect(declarative.ApisList[ec.URI][http.MethodGet]).To(gomega.BeEquivalentTo(map[string]interface{}{
			"group":  ec.GroupName,
			"kind":   ec.KindName,
			"params": []string{"projectId"},
			"uri":    ec.SpecURI,
		}))
	})

	ginkgo.It("should register declarative router", func() {
		config.Cfg = &config.Config{
			Server:             config.ServerConfig{},
			EnableNexusRuntime: true,
			BackendService:     "",
		}
		e := echoserver.NewEchoServer(config.Cfg, &kubernetes.Clientset{},
			&nexusClient.Clientset{},
			&nexusClient.Clientset{},
		)
		e.RegisterDeclarativeRouter()

		c := e.Echo.NewContext(nil, nil)
		e.Echo.Router().Find(http.MethodGet, "/apis/gns.vmware.org/v1/globalnamespaces", c)
		gomega.Expect(c.Path()).To(gomega.Equal("/apis/gns.vmware.org/v1/globalnamespaces"))

		// short name
		c = e.Echo.NewContext(nil, nil)
		e.Echo.Router().Find(http.MethodGet, "/apis/v1/gns", c)
		gomega.Expect(c.Path()).To(gomega.Equal("/apis/v1/gns"))

		c = e.Echo.NewContext(nil, nil)
		e.Echo.Router().Find(http.MethodGet, "/apis/gns.vmware.org/v1/globalnamespaces/:name", c)
		gomega.Expect(c.Path()).To(gomega.Equal("/apis/gns.vmware.org/v1/globalnamespaces/:name"))

		// short name
		c = e.Echo.NewContext(nil, nil)
		e.Echo.Router().Find(http.MethodGet, "/apis/v1/gns/:name", c)
		gomega.Expect(c.Path()).To(gomega.Equal("/apis/v1/gns/:name"))

		c = e.Echo.NewContext(nil, nil)
		e.Echo.Router().Find(http.MethodPut, "/apis/gns.vmware.org/v1/globalnamespaces", c)
		gomega.Expect(c.Path()).To(gomega.Equal("/apis/gns.vmware.org/v1/globalnamespaces"))

		// short name
		c = e.Echo.NewContext(nil, nil)
		e.Echo.Router().Find(http.MethodGet, "/apis/v1/gns", c)
		gomega.Expect(c.Path()).To(gomega.Equal("/apis/v1/gns"))

		c = e.Echo.NewContext(nil, nil)
		e.Echo.Router().Find(http.MethodDelete, "/apis/gns.vmware.org/v1/globalnamespaces/:name", c)
		gomega.Expect(c.Path()).To(gomega.Equal("/apis/gns.vmware.org/v1/globalnamespaces/:name"))

		// short name
		c = e.Echo.NewContext(nil, nil)
		e.Echo.Router().Find(http.MethodGet, "/apis/v1/gns/:name", c)
		gomega.Expect(c.Path()).To(gomega.Equal("/apis/v1/gns/:name"))
	})

	ginkgo.It("should parse schema for GlobalNamespace", func() {
		ec := declarative.SetupContext(URI, http.MethodGet, declarative.Paths[URI].Get)
		declarative.AddApisEndpoint(ec)

		gomega.Expect(declarative.ApisList).To(gomega.HaveKey("/apis/gns.vmware.org/v1/globalnamespaces"))
		gomega.Expect(declarative.ApisList["/apis/gns.vmware.org/v1/globalnamespaces"]).To(gomega.HaveKey("yaml"))

		expectedYaml := `apiVersion: gns.vmware.org/v1
kind: GlobalNamespace
metadata:
  labels:
    projectId: string
  name: string
spec:
  api_discovery_enabled: true
  ca: string
  ca_type: PreExistingCA
  color: string
  description: string
  display_name: string
  domain_name: string
  match_conditions:
  - cluster:
      match: string
      type: string
    namespace:
      match: string
      type: string
    service: object
  mtls_enforced: true
  name: string
  use_shared_gateway: true
  version: string
`
		gomega.Expect(declarative.ApisList["/apis/gns.vmware.org/v1/globalnamespaces"]["yaml"]).To(gomega.Equal(expectedYaml))
	})
})
