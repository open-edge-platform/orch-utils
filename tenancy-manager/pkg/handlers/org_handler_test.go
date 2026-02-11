// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	orgv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/org.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/handlers"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

var _ = ginkgo.Describe("OrgHandler", func() {
	var (
		handler     *handlers.OrgHandler
		e           *echo.Echo
		nexusClient *nexus_client.Clientset
	)

	ginkgo.BeforeEach(func() {
		// Create fake Nexus client
		nexusClient = nexus_client.NewFakeClient()
		nexusClient.DynamicClient = fake.NewSimpleDynamicClient(runtime.NewScheme())

		// Create handler
		logger := zerolog.Nop()
		handler = handlers.NewOrgHandler(nexusClient, logger)

		// Create Echo instance
		e = echo.New()
	})

	ginkgo.Describe("CreateOrg", func() {
		ginkgo.Context("when request is valid", func() {
			ginkgo.It("should create organization successfully", func() {
				orgReq := orgv1.Org{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Org",
						APIVersion: "org.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-org",
					},
					Spec: orgv1.OrgSpec{},
				}
				body, _ := json.Marshal(orgReq)
				req := httptest.NewRequest(http.MethodPost, "/v1/orgs", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.CreateOrg(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))

				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(response["message"]).To(gomega.Equal("Organization created successfully"))
			})
		})

		ginkgo.Context("when request body is invalid", func() {
			ginkgo.It("should return bad request error", func() {
				req := httptest.NewRequest(http.MethodPost, "/v1/orgs", bytes.NewReader([]byte("invalid json")))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.CreateOrg(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})

		ginkgo.Context("when organization name is missing", func() {
			ginkgo.It("should return bad request error", func() {
				orgReq := orgv1.Org{
					Spec: orgv1.OrgSpec{},
				}
				body, _ := json.Marshal(orgReq)
				req := httptest.NewRequest(http.MethodPost, "/v1/orgs", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.CreateOrg(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
				gomega.Expect(httpErr.Message).To(gomega.Equal("Organization name is required"))
			})
		})
	})

	ginkgo.Describe("ListOrgs", func() {
		ginkgo.Context("when no organizations exist", func() {
			ginkgo.It("should return empty list", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/orgs", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.ListOrgs(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				items := response["items"].([]interface{})
				gomega.Expect(len(items)).To(gomega.Equal(0))
			})
		})

		ginkgo.Context("when organizations exist", func() {
			ginkgo.It("should return list of organizations", func() {
				// Create test organization
				orgObj := &orgv1.Org{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Org",
						APIVersion: "org.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-org-1",
					},
					Spec: orgv1.OrgSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().AddOrgs(context.Background(), orgObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				req := httptest.NewRequest(http.MethodGet, "/v1/orgs", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err = handler.ListOrgs(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				items := response["items"].([]interface{})
				gomega.Expect(len(items)).To(gomega.BeNumerically(">=", 1))
			})
		})
	})

	ginkgo.Describe("GetOrg", func() {
		ginkgo.Context("when organization exists", func() {
			// Skip: Fake client doesn't properly persist objects for GetByName retrieval
			ginkgo.It("should return organization [Integration]", func() {
				ginkgo.Skip("Requires real Nexus client for GetOrgByName ")
				// Create test organization
				orgObj := &orgv1.Org{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Org",
						APIVersion: "org.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "get-test-org",
					},
					Spec: orgv1.OrgSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().AddOrgs(context.Background(), orgObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				req := httptest.NewRequest(http.MethodGet, "/v1/orgs/get-test-org", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("orgId")
				c.SetParamValues("get-test-org")

				err = handler.GetOrg(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response orgv1.Org
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(response.Name).To(gomega.Equal("get-test-org"))
			})
		})

		ginkgo.Context("when organization does not exist", func() {
			ginkgo.It("should return not found error", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/orgs/nonexistent", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("orgId")
				c.SetParamValues("nonexistent")

				err := handler.GetOrg(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})

		ginkgo.Context("when orgId is missing", func() {
			ginkgo.It("should return bad request error", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/orgs/", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.GetOrg(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})
	})

	ginkgo.Describe("UpdateOrg", func() {
		ginkgo.Context("when organization exists", func() {
			ginkgo.It("should update organization successfully", func() {
				// Create test organization
				orgObj := &orgv1.Org{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Org",
						APIVersion: "org.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "update-test-org",
					},
					Spec: orgv1.OrgSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().AddOrgs(context.Background(), orgObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				updateReq := orgv1.Org{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"env": "test"},
					},
				}
				body, _ := json.Marshal(updateReq)
				req := httptest.NewRequest(http.MethodPut, "/v1/orgs/update-test-org", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("orgId")
				c.SetParamValues("update-test-org")

				err = handler.UpdateOrg(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
			})
		})
		ginkgo.Context("when organization does not exist", func() {
			ginkgo.It("should return not found error", func() {
				updateReq := orgv1.Org{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"env": "test"},
					},
				}
				body, _ := json.Marshal(updateReq)
				req := httptest.NewRequest(http.MethodPut, "/v1/orgs/nonexistent", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("orgId")
				c.SetParamValues("nonexistent")

				err := handler.UpdateOrg(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})
	})

	ginkgo.Describe("DeleteOrg", func() {
		ginkgo.Context("when organization exists", func() {
			ginkgo.It("should delete organization successfully", func() {
				// Create test organization
				orgObj := &orgv1.Org{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Org",
						APIVersion: "org.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "delete-test-org",
					},
					Spec: orgv1.OrgSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().AddOrgs(context.Background(), orgObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/delete-test-org", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("orgId")
				c.SetParamValues("delete-test-org")

				err = handler.DeleteOrg(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response map[string]string
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(response["message"]).To(gomega.Equal("Organization deleted successfully"))
			})
		})

		ginkgo.Context("when organization does not exist", func() {
			ginkgo.It("should return not found error", func() {
				req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/nonexistent", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("orgId")
				c.SetParamValues("nonexistent")

				err := handler.DeleteOrg(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})
	})

	ginkgo.Describe("GetOrgStatus", func() {
		ginkgo.Context("when organization exists", func() {
			ginkgo.It("should return organization status", func() {
				// Create test organization
				orgObj := &orgv1.Org{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Org",
						APIVersion: "org.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "status-test-org",
					},
					Spec: orgv1.OrgSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().AddOrgs(context.Background(), orgObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				req := httptest.NewRequest(http.MethodGet, "/v1/orgs/status-test-org/status", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("orgId")
				c.SetParamValues("status-test-org")

				err = handler.GetOrgStatus(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
			})
		})

		ginkgo.Context("when organization does not exist", func() {
			ginkgo.It("should return not found error", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/orgs/nonexistent/status", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("orgId")
				c.SetParamValues("nonexistent")

				err := handler.GetOrgStatus(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})
	})
})
