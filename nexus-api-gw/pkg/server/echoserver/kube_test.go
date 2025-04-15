// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package echoserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v4"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/client"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/model"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/server/echoserver"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = ginkgo.Describe("Kube tests", ginkgo.Ordered, func() {
	ginkgo.It("should fail the request with invalid body using kubePost handler", func() {
		gnsJSON := `invalid`

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "roots.root.vmware.org",
			GroupName: "root.vmware.org",
			Resource:  "roots",
		}

		err := echoserver.KubePostHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())

		status := nc.Context.Response().Status
		responseBody := rec.Body.String()

		gomega.Expect(status).ToNot(gomega.Equal(http.StatusOK))
		gomega.Expect(responseBody).To(gomega.ContainSubstring(gnsJSON))
	})

	ginkgo.It("should fail creating object without a spec using kubePost handler", func() {
		gnsJSON := `{}`

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "roots.root.vmware.org",
			GroupName: "root.vmware.org",
			Resource:  "roots",
		}

		model.CrdTypeToNodeInfo["roots.root.vmware.org"] = model.NodeInfo{
			Name:            "Root.root",
			ParentHierarchy: []string{},
			Children: map[string]model.NodeHelperChild{
				"globalnamespaces.gns.vmware.org": {
					FieldName:    "gns",
					FieldNameGvk: "gnsGvk",
					IsNamed:      false,
				},
			},
		}

		err := echoserver.KubePostHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).ToNot(gomega.Equal(http.StatusOK))
	})

	ginkgo.It("should create root object without a spec using kubePost handler", func() {
		serverObj, stopCh := setupServer()
		defer teardownServer(serverObj, stopCh)
		gnsJSON := `{
			"apiVersion": "root.vmware.org/v1",
			"kind": "Root",
			"metadata": {
				"name": "root"
			}
		}`

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "roots.root.vmware.org",
			GroupName: "root.vmware.org",
			Resource:  "roots",
		}

		model.CrdTypeToNodeInfo["roots.root.vmware.org"] = model.NodeInfo{
			Name:            "Root.root",
			ParentHierarchy: []string{},
			Children: map[string]model.NodeHelperChild{
				"globalnamespaces.gns.vmware.org": {
					FieldName:    "gns",
					FieldNameGvk: "gnsGvk",
					IsNamed:      false,
				},
			},
		}

		err := echoserver.KubePostHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))

		expectedResponse := `{
			"apiVersion": "root.vmware.org/v1",
			"kind": "Root",
			"metadata": {
				"labels": {
					"nexus/display_name": "root",
					"nexus/is_name_hashed": "true"
				},
				"name": "de3f9fe476b35572145d6b4031712249619efdae"
			},
			"spec": {}
		}`

		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.It("should create gns object using kubePost handler", func() {
		gnsJSON := `{
	"apiVersion": "gns.vmware.org/v1",
	"kind": "GlobalNamespace",
    "metadata": {
        "name": "test",
		"labels": {
			"roots.root.vmware.org": "root"
		}
    },
    "spec": {
        "foo": "bar"
    }
}`

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "globalnamespaces.gns.vmware.org",
			GroupName: "gns.vmware.org",
			Resource:  "globalnamespaces",
		}

		model.CrdTypeToNodeInfo["globalnamespaces.gns.vmware.org"] = model.NodeInfo{
			Name:            "Gns.gns",
			ParentHierarchy: []string{"roots.root.vmware.org"},
		}

		err := echoserver.KubePostHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))
		expectedResponse := `{
			"apiVersion": "gns.vmware.org/v1",
			"kind": "GlobalNamespace",
			"metadata": {
				"labels": {
					"nexus/display_name": "test",
					"nexus/is_name_hashed": "true",
					"roots.root.vmware.org": "root"
				},
				"name": "2587591c2e1023ff9498b1b70ac5cbcb84504352"
			},
			"spec": {
				"foo": "bar"
			}
		}`

		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.It("should get gns object using kubeGet handler", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.QueryParams().Add("limit", "1")
		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "globalnamespaces.gns.vmware.org",
			GroupName: "gns.vmware.org",
			Resource:  "globalnamespaces",
		}

		err := echoserver.KubeGetHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))

		expectedResponse := `{
			"apiVersion": "gns.vmware.org/v1",
			"items": [
				{
					"apiVersion": "gns.vmware.org/v1",
					"kind": "GlobalNamespace",
					"metadata": {
						"labels": {
							"nexus/display_name": "test",
							"nexus/is_name_hashed": "true",
							"roots.root.vmware.org": "root"
						},
						"name": "2587591c2e1023ff9498b1b70ac5cbcb84504352"
					},
					"spec": {
						"foo": "bar"
					}
				}
			],
			"kind": "GlobalNamespaceList",
			"metadata": {
				"continue": "",
				"resourceVersion": ""
			}
		}`
		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.It("should update gns object using kubePost handler", func() {
		gnsJSON := `{
	"apiVersion": "gns.vmware.org/v1",
	"kind": "GlobalNamespace",
    "metadata": {
        "name": "test",
		"labels": {
			"roots.root.vmware.org": "root"
		}
    },
    "spec": {
        "foo": "bar2"
    }
}`

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "globalnamespaces.gns.vmware.org",
			GroupName: "gns.vmware.org",
			Resource:  "globalnamespaces",
		}

		model.CrdTypeToNodeInfo["globalnamespaces.gns.vmware.org"] = model.NodeInfo{
			Name:            "Gns.gns",
			ParentHierarchy: []string{"roots.root.vmware.org"},
		}

		err := echoserver.KubePostHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))
		expectedResponse := `{
			"apiVersion": "gns.vmware.org/v1",
			"kind": "GlobalNamespace",
			"metadata": {
				"labels": {
					"nexus/display_name": "test",
					"nexus/is_name_hashed": "true",
					"roots.root.vmware.org": "root"
				},
				"name": "2587591c2e1023ff9498b1b70ac5cbcb84504352"
			},
			"spec": {
				"foo": "bar2"
			}
		}`
		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.It("should not remove child/link Gvks while update by kubePost handler", func() {
		gvr := schema.GroupVersionResource{
			Group:    "orgchart.vmware.org",
			Version:  "v1",
			Resource: "foos",
		}
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "orgchart.vmware.org/v1",
				"kind":       "Foo",
				"metadata": map[string]interface{}{
					"name": "bb2cbdf1b03e754cea2c9da8e9134c050bc0d547",
				},
				"spec": map[string]interface{}{
					"childGvk": "value_one",
					"linkGvk":  "value_two",
					"name":     "bob",
				},
			},
		}
		_, err := client.Client.Resource(gvr).Create(context.TODO(), obj, metav1.CreateOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		e := echo.New()

		// If the newspec contains new fields, updateResource should add them while retaining the Gvk fields.
		gnsJSON := `{
			"apiVersion": "orgchart.vmware.org/v1",
			"kind": "Foo",
			"metadata": {
				"name": "test",
				"labels": {
					"roots.root.vmware.org": "root"
				}
			},
			"spec": {
				"name": "bob",
				"foo": "bar2"
			}
		}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "foos.orgchart.vmware.org",
			GroupName: "orgchart.vmware.org",
			Resource:  "foos",
		}
		model.CrdTypeToNodeInfo["foos.orgchart.vmware.org"] = model.NodeInfo{
			Name:            "Foo.foo",
			ParentHierarchy: []string{"roots.root.vmware.org"},
			Children: map[string]model.NodeHelperChild{
				"childGVK": {
					FieldNameGvk: "childGvk",
				},
				"linkGVK": {
					FieldNameGvk: "linkGvk",
				},
			},
		}

		err = echoserver.KubePostHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))

		expectedResponse := `{
			"apiVersion": "orgchart.vmware.org/v1",
			"kind": "Foo",
			"metadata": {
				"labels": {
					"nexus/display_name": "test",
					"nexus/is_name_hashed": "true",
					"roots.root.vmware.org": "root"
				},
				"name": "bb2cbdf1b03e754cea2c9da8e9134c050bc0d547"
			},
			"spec": {
				"childGvk": "value_one",
				"foo": "bar2",
				"linkGvk": "value_two",
				"name": "bob"
			}
		}`
		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))

		// If the newspec has empty spec, updateResource should remove the existing spec fields while retaining the Gvk fields.
		gnsJSON = `{
			"apiVersion": "orgchart.vmware.org/v1",
			"kind": "Foo",
			"metadata": {
				"name": "test",
				"labels": {
					"roots.root.vmware.org": "root"
				}
			},
			"spec": {}
		}`
		req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec = httptest.NewRecorder()
		c = e.NewContext(req, rec)
		nc = &echoserver.NexusContext{
			Context:   c,
			CrdType:   "foos.orgchart.vmware.org",
			GroupName: "orgchart.vmware.org",
			Resource:  "foos",
		}

		err = echoserver.KubePostHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))

		expectedResponse = `{
			"apiVersion": "orgchart.vmware.org/v1",
			"kind": "Foo",
			"metadata": {
				"labels": {
					"nexus/display_name": "test",
					"nexus/is_name_hashed": "true",
					"roots.root.vmware.org": "root"
				},
				"name": "bb2cbdf1b03e754cea2c9da8e9134c050bc0d547"
			},
			"spec": {
				"childGvk": "value_one",
				"linkGvk": "value_two"
			}
		}`

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.It("should get gns object using kubeGetByName handler", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:name")
		c.SetParamNames("name")
		c.SetParamValues("2587591c2e1023ff9498b1b70ac5cbcb84504352")

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "globalnamespaces.gns.vmware.org",
			GroupName: "gns.vmware.org",
			Resource:  "globalnamespaces",
		}

		err := echoserver.KubeGetByNameHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))
		expectedResponse := `{
			"apiVersion": "gns.vmware.org/v1",
			"kind": "GlobalNamespace",
			"metadata": {
				"labels": {
					"nexus/display_name": "test",
					"nexus/is_name_hashed": "true",
					"roots.root.vmware.org": "root"
				},
				"name": "2587591c2e1023ff9498b1b70ac5cbcb84504352"
			},
			"spec": {
				"foo": "bar2"
			}
		}`
		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.It("should fail kubeGetByName handler by using object with non-existent name", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:name")
		c.SetParamNames("name")
		c.SetParamValues("non-existent-id")

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "globalnamespaces.gns.vmware.org",
			GroupName: "gns.vmware.org",
			Resource:  "globalnamespaces",
		}

		err := echoserver.KubeGetByNameHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusNotFound))
		expectedResponse := `{
			"metadata": {},
			"status": "Failure",
			"message": "globalnamespaces.gns.vmware.org \"non-existent-id\" not found",
			"reason": "NotFound",
			"details": {
				"name": "non-existent-id",
				"group": "gns.vmware.org",
				"kind": "globalnamespaces"
			},
			"code": 404
		}`
		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.It("should delete gns object using kubeDelete handler with kubectl user agent", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/", http.NoBody)
		req.Header.Set("User-Agent", "kubectl")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:name")
		c.SetParamNames("name")
		c.SetParamValues("test")
		lbl := "nexus/display_name=test, nexus/is_name_hashed=true, roots.root.vmware.org=root"
		c.QueryParams().Add("labelSelector", lbl)

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "globalnamespaces.gns.vmware.org",
			GroupName: "gns.vmware.org",
			Resource:  "globalnamespaces",
		}

		err := echoserver.KubeDeleteHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))

		expectedResponse := `{
			"apiVersion": "v1",
			"details": {
				"group": "gns.vmware.org",
				"kind": "globalnamespaces",
				"name": "test"
			},
			"kind": "Status",
			"metadata": {},
			"status": "Success"
		}`
		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.Context("test delete method with label selector", func() {
		ginkgo.It("should create gns object using kubePost handler", func() {
			gnsJSON := `{
	"apiVersion": "gns.vmware.org/v1",
	"kind": "GlobalNamespace",
    "metadata": {
        "name": "test",
		"labels": {
			"roots.root.vmware.org": "root"
		}
    },
    "spec": {
        "foo": "bar"
    }
}`

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(gnsJSON))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			nc := &echoserver.NexusContext{
				Context:   c,
				CrdType:   "globalnamespaces.gns.vmware.org",
				GroupName: "gns.vmware.org",
				Resource:  "globalnamespaces",
			}

			model.CrdTypeToNodeInfo["globalnamespaces.gns.vmware.org"] = model.NodeInfo{
				Name:            "Gns.gns",
				ParentHierarchy: []string{"roots.root.vmware.org"},
			}

			err := echoserver.KubePostHandler(nc)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))

			expectedResponse := `{
				"apiVersion": "gns.vmware.org/v1",
				"kind": "GlobalNamespace",
				"metadata": {
					"labels": {
						"nexus/display_name": "test",
						"nexus/is_name_hashed": "true",
						"roots.root.vmware.org": "root"
					},
					"name": "2587591c2e1023ff9498b1b70ac5cbcb84504352"
				},
				"spec": {
					"foo": "bar"
				}
			}`
			var actualResponseMap map[string]interface{}
			var expectedResponseMap map[string]interface{}

			err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
			gomega.Expect(err).To(gomega.BeNil())

			err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
		})

		ginkgo.It("should delete gns object using kubeDelete handler", func() {
			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, "/test?labelSelector=roots.root.vmware.org=root", http.NoBody)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/:name?labelSelector=roots.root.vmware.org=root")
			c.SetParamNames("name")
			c.SetParamValues("test")

			nc := &echoserver.NexusContext{
				Context:   c,
				CrdType:   "globalnamespaces.gns.vmware.org",
				GroupName: "gns.vmware.org",
				Resource:  "globalnamespaces",
			}

			err := echoserver.KubeDeleteHandler(nc)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))

			expectedResponse := `{
				"apiVersion": "v1",
				"details": {
					"group": "gns.vmware.org",
					"kind": "globalnamespaces",
					"name": "test"
				},
				"kind": "Status",
				"metadata": {},
				"status": "Success"
			}`
			var actualResponseMap map[string]interface{}
			var expectedResponseMap map[string]interface{}

			err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
			gomega.Expect(err).To(gomega.BeNil())

			err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
		})

		ginkgo.It("should delete root object using kubeDeleteHandler", func() {
			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, "/root", http.NoBody)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/:name")
			c.SetParamNames("name")
			c.SetParamValues("root")

			nc := &echoserver.NexusContext{
				Context:   c,
				CrdType:   "roots.root.vmware.org",
				GroupName: "root.vmware.org",
				Resource:  "roots",
			}

			err := echoserver.KubeDeleteHandler(nc)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusOK))

			expectedResponse := `{
				"apiVersion": "v1",
				"details": {
					"group": "root.vmware.org",
					"kind": "roots",
					"name": "root"
				},
				"kind": "Status",
				"metadata": {},
				"status": "Success"
			}`
			log.Debug(rec.Body.String())
			var actualResponseMap map[string]interface{}
			var expectedResponseMap map[string]interface{}

			err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
			gomega.Expect(err).To(gomega.BeNil())

			err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
		})
	})

	ginkgo.It("should fail kubeDelete handler with kubectl user agent by using object with non-existent name", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/", http.NoBody)
		req.Header.Set("User-Agent", "kubectl")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:name")
		c.SetParamNames("name")
		c.SetParamValues("non-existent-id")

		nc := &echoserver.NexusContext{
			Context:   c,
			CrdType:   "globalnamespaces.gns.vmware.org",
			GroupName: "gns.vmware.org",
			Resource:  "globalnamespaces",
		}

		err := echoserver.KubeDeleteHandler(nc)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(nc.Context.Response().Status).To(gomega.Equal(http.StatusNotFound))
		expectedResponse := `{
			"metadata": {},
			"status": "Failure",
			"message": "globalnamespaces.gns.vmware.org \"6e35a0f511da038c105cba5c6712587a1c023628\" not found",
			"reason": "NotFound",
			"details": {
				"name": "6e35a0f511da038c105cba5c6712587a1c023628",
				"group": "gns.vmware.org",
				"kind": "globalnamespaces"
			},
			"code": 404
		}`
		var actualResponseMap map[string]interface{}
		var expectedResponseMap map[string]interface{}

		err = json.Unmarshal(rec.Body.Bytes(), &actualResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		err = json.Unmarshal([]byte(expectedResponse), &expectedResponseMap)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(actualResponseMap).To(gomega.Equal(expectedResponseMap))
	})

	ginkgo.It("should test updateProxyResponse method when custom not found page cfg is not set", func() {
		if config.Cfg == nil {
			config.Cfg = &config.Config{}
		}
		config.Cfg.CustomNotFoundPage = ""
		err := echoserver.UpdateProxyResponse(&http.Response{})
		gomega.Expect(err).To(gomega.BeNil())
	})

	ginkgo.It("should test updateProxyResponse method when custom not found page cfg is set", func() {
		if config.Cfg == nil {
			config.Cfg = &config.Config{}
		}

		body := []byte(`404 page`)
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
			res.WriteHeader(http.StatusOK)
			res.Write(body)
		}))

		config.Cfg.CustomNotFoundPage = server.URL
		response := &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader(body)), // Properly initialize the response body
		}
		err := echoserver.UpdateProxyResponse(response)
		gomega.Expect(err).To(gomega.BeNil())

		bodyRes, err := io.ReadAll(response.Body)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(bodyRes).To(gomega.Equal(body))
	})
})
