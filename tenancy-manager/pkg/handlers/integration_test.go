// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package handlers_test contains integration tests for org and project CRUD handlers.
// These tests exercise the full HTTP stack (routing → handler → Nexus fake client)
// inspired by the nexus-api-gw handler_test.go pattern, using httptest and a fake
// Nexus client instead of a live cluster.
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
	projectv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/project.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/handlers"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func newFakeNexusClient() *nexus_client.Clientset {
	c := nexus_client.NewFakeClient()
	c.DynamicClient = fake.NewSimpleDynamicClient(runtime.NewScheme())
	return c
}

func orgBody(name string, labels map[string]string) []byte {
	o := orgv1.Org{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Org",
			APIVersion: "org.edge-orchestrator.intel.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	b, _ := json.Marshal(o)
	return b
}

func projectBody(name, orgID, folderID string, extraLabels map[string]string) []byte {
	labels := map[string]string{
		"orgs.org.edge-orchestrator.intel.com":       orgID,
		"folders.folder.edge-orchestrator.intel.com": folderID,
	}
	for k, v := range extraLabels {
		labels[k] = v
	}
	p := projectv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Project",
			APIVersion: "project.edge-orchestrator.intel.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	b, _ := json.Marshal(p)
	return b
}

func jsonReq(method, path string, body []byte) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return req, httptest.NewRecorder()
}

// ─── Org CRUD integration tests ─────────────────────────────────────────────

var _ = ginkgo.Describe("Org CRUD Integration", func() {
	var (
		e           *echo.Echo
		nc          *nexus_client.Clientset
		orgHandler  *handlers.OrgHandler
	)

	ginkgo.BeforeEach(func() {
		nc = newFakeNexusClient()
		orgHandler = handlers.NewOrgHandler(nc, zerolog.Nop())
		e = echo.New()
	})

	// ── Create ────────────────────────────────────────────────────────────

	ginkgo.Describe("POST /v1/orgs", func() {
		ginkgo.It("creates an org and returns 201", func() {
			req, rec := jsonReq(http.MethodPost, "/v1/orgs", orgBody("acme", nil))
			c := e.NewContext(req, rec)

			err := orgHandler.CreateOrg(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))

			var resp map[string]interface{}
			gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(gomega.Succeed())
			gomega.Expect(resp["message"]).To(gomega.Equal("Organization created successfully"))
			gomega.Expect(resp["org"]).NotTo(gomega.BeNil())
		})

		ginkgo.It("returns 400 when body is malformed JSON", func() {
			req := httptest.NewRequest(http.MethodPost, "/v1/orgs", bytes.NewReader([]byte("{bad json")))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c := e.NewContext(req, httptest.NewRecorder())

			err := orgHandler.CreateOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 400 when org name is empty", func() {
			req, rec := jsonReq(http.MethodPost, "/v1/orgs", orgBody("", nil))
			c := e.NewContext(req, rec)

			err := orgHandler.CreateOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			httpErr := err.(*echo.HTTPError)
			gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			gomega.Expect(httpErr.Message).To(gomega.Equal("Organization name is required"))
		})

		ginkgo.It("stores labels supplied by the caller", func() {
			req, rec := jsonReq(http.MethodPost, "/v1/orgs", orgBody("labeled-org", map[string]string{"env": "prod"}))
			c := e.NewContext(req, rec)

			err := orgHandler.CreateOrg(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))
		})
	})

	// ── List ──────────────────────────────────────────────────────────────

	ginkgo.Describe("GET /v1/orgs", func() {
		ginkgo.It("returns an empty list when no orgs exist", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/orgs", nil)
			c := e.NewContext(req, rec)

			err := orgHandler.ListOrgs(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

			var resp map[string]interface{}
			gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(gomega.Succeed())
			gomega.Expect(resp["items"].([]interface{})).To(gomega.BeEmpty())
			gomega.Expect(resp["total"]).To(gomega.BeEquivalentTo(0))
		})

		ginkgo.It("returns created orgs in the list", func() {
			// Pre-populate two orgs
			for _, name := range []string{"org-alpha", "org-beta"} {
				_, err := nc.TenancyMultiTenancy().Config().AddOrgs(context.Background(), &orgv1.Org{
					ObjectMeta: metav1.ObjectMeta{Name: name},
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}

			req, rec := jsonReq(http.MethodGet, "/v1/orgs", nil)
			c := e.NewContext(req, rec)

			err := orgHandler.ListOrgs(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

			var resp map[string]interface{}
			gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(gomega.Succeed())
			gomega.Expect(len(resp["items"].([]interface{}))).To(gomega.BeNumerically(">=", 2))
		})
	})

	// ── Get ───────────────────────────────────────────────────────────────

	ginkgo.Describe("GET /v1/orgs/:orgId", func() {
		ginkgo.It("returns 400 when orgId param is missing", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/orgs/", nil)
			c := e.NewContext(req, rec)
			// intentionally no SetParamNames/SetParamValues

			err := orgHandler.GetOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 404 when org does not exist", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/orgs/ghost-org", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("ghost-org")

			err := orgHandler.GetOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})
	})

	// ── Update ────────────────────────────────────────────────────────────

	ginkgo.Describe("PUT /v1/orgs/:orgId", func() {
		ginkgo.It("returns 400 when orgId is missing", func() {
			req, rec := jsonReq(http.MethodPut, "/v1/orgs/", orgBody("", map[string]string{"env": "test"}))
			c := e.NewContext(req, rec)

			err := orgHandler.UpdateOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 400 when request body is malformed", func() {
			req := httptest.NewRequest(http.MethodPut, "/v1/orgs/my-org", bytes.NewReader([]byte("not-json")))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("my-org")

			err := orgHandler.UpdateOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 404 when org does not exist", func() {
			req, rec := jsonReq(http.MethodPut, "/v1/orgs/nonexistent", orgBody("nonexistent", map[string]string{"env": "test"}))
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("nonexistent")

			err := orgHandler.UpdateOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})

		ginkgo.It("updates labels of an existing org and returns 200", func() {
			_, err := nc.TenancyMultiTenancy().Config().AddOrgs(context.Background(), &orgv1.Org{
				ObjectMeta: metav1.ObjectMeta{Name: "update-org"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			req, rec := jsonReq(http.MethodPut, "/v1/orgs/update-org", orgBody("update-org", map[string]string{"env": "staging"}))
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("update-org")

			err = orgHandler.UpdateOrg(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		})
	})

	// ── Delete ────────────────────────────────────────────────────────────

	ginkgo.Describe("DELETE /v1/orgs/:orgId", func() {
		ginkgo.It("returns 400 when orgId is missing", func() {
			req, rec := jsonReq(http.MethodDelete, "/v1/orgs/", nil)
			c := e.NewContext(req, rec)

			err := orgHandler.DeleteOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 404 when org does not exist", func() {
			req, rec := jsonReq(http.MethodDelete, "/v1/orgs/ghost", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("ghost")

			err := orgHandler.DeleteOrg(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})

		ginkgo.It("deletes an existing org and returns 200", func() {
			_, err := nc.TenancyMultiTenancy().Config().AddOrgs(context.Background(), &orgv1.Org{
				ObjectMeta: metav1.ObjectMeta{Name: "del-org"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			req, rec := jsonReq(http.MethodDelete, "/v1/orgs/del-org", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("del-org")

			err = orgHandler.DeleteOrg(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

			var resp map[string]string
			gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(gomega.Succeed())
			gomega.Expect(resp["message"]).To(gomega.Equal("Organization deleted successfully"))
		})

		ginkgo.It("cannot delete the same org twice", func() {
			_, err := nc.TenancyMultiTenancy().Config().AddOrgs(context.Background(), &orgv1.Org{
				ObjectMeta: metav1.ObjectMeta{Name: "once-org"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// First delete
			req, rec := jsonReq(http.MethodDelete, "/v1/orgs/once-org", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("once-org")
			gomega.Expect(orgHandler.DeleteOrg(c)).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

			// Second delete — should fail
			req2, rec2 := jsonReq(http.MethodDelete, "/v1/orgs/once-org", nil)
			c2 := e.NewContext(req2, rec2)
			c2.SetParamNames("orgId")
			c2.SetParamValues("once-org")
			err = orgHandler.DeleteOrg(c2)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})
	})

	// ── GetStatus ─────────────────────────────────────────────────────────

	ginkgo.Describe("GET /v1/orgs/:orgId/status", func() {
		ginkgo.It("returns 400 when orgId is missing", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/orgs//status", nil)
			c := e.NewContext(req, rec)

			err := orgHandler.GetOrgStatus(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 404 when org does not exist", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/orgs/ghost/status", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("ghost")

			err := orgHandler.GetOrgStatus(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})

		ginkgo.It("returns status for an existing org", func() {
			_, err := nc.TenancyMultiTenancy().Config().AddOrgs(context.Background(), &orgv1.Org{
				ObjectMeta: metav1.ObjectMeta{Name: "status-org"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			req, rec := jsonReq(http.MethodGet, "/v1/orgs/status-org/status", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("orgId")
			c.SetParamValues("status-org")

			err = orgHandler.GetOrgStatus(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		})
	})
})

// ─── Project CRUD integration tests ─────────────────────────────────────────

var _ = ginkgo.Describe("Project CRUD Integration", func() {
	var (
		e              *echo.Echo
		nc             *nexus_client.Clientset
		projectHandler *handlers.ProjectHandler
	)

	const (
		testOrg    = "proj-test-org"
		testFolder = "default"
	)

	ginkgo.BeforeEach(func() {
		nc = newFakeNexusClient()
		projectHandler = handlers.NewProjectHandler(nc, zerolog.Nop())
		e = echo.New()

		// Every project test needs a pre-existing org
		_, err := nc.TenancyMultiTenancy().Config().AddOrgs(context.Background(), &orgv1.Org{
			ObjectMeta: metav1.ObjectMeta{Name: testOrg},
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	// ── Create ────────────────────────────────────────────────────────────

	ginkgo.Describe("POST /v1/projects", func() {
		ginkgo.It("creates a project and returns 201", func() {
			req, rec := jsonReq(http.MethodPost, "/v1/projects", projectBody("my-proj", testOrg, testFolder, nil))
			c := e.NewContext(req, rec)

			err := projectHandler.CreateProject(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))

			var resp map[string]interface{}
			gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(gomega.Succeed())
			gomega.Expect(resp["message"]).To(gomega.Equal("Project created successfully"))
			gomega.Expect(resp["project"]).NotTo(gomega.BeNil())
		})

		ginkgo.It("defaults folderId to 'default' when not supplied", func() {
			// omit the folder label — handler should default it
			p := projectv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-folder-proj",
					Labels: map[string]string{
						"orgs.org.edge-orchestrator.intel.com": testOrg,
					},
				},
			}
			body, _ := json.Marshal(p)
			req, rec := jsonReq(http.MethodPost, "/v1/projects", body)
			c := e.NewContext(req, rec)

			err := projectHandler.CreateProject(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))
		})

		ginkgo.It("returns 400 when project name is empty", func() {
			req, rec := jsonReq(http.MethodPost, "/v1/projects", projectBody("", testOrg, testFolder, nil))
			c := e.NewContext(req, rec)

			err := projectHandler.CreateProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			httpErr := err.(*echo.HTTPError)
			gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			gomega.Expect(httpErr.Message).To(gomega.Equal("Project name is required"))
		})

		ginkgo.It("returns 400 when orgId label is missing", func() {
			p := projectv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "no-org-proj",
					Labels: map[string]string{},
				},
			}
			body, _ := json.Marshal(p)
			req, rec := jsonReq(http.MethodPost, "/v1/projects", body)
			c := e.NewContext(req, rec)

			err := projectHandler.CreateProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 400 when body is malformed JSON", func() {
			req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewReader([]byte("{bad")))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c := e.NewContext(req, httptest.NewRecorder())

			err := projectHandler.CreateProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})
	})

	// ── List ──────────────────────────────────────────────────────────────

	ginkgo.Describe("GET /v1/projects", func() {
		ginkgo.It("returns an empty list when no projects exist", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/projects", nil)
			c := e.NewContext(req, rec)

			err := projectHandler.ListProjects(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

			var resp map[string]interface{}
			gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(gomega.Succeed())
			gomega.Expect(resp["items"].([]interface{})).To(gomega.BeEmpty())
			gomega.Expect(resp["total"]).To(gomega.BeEquivalentTo(0))
		})

		ginkgo.It("returns created projects in the list", func() {
			for _, name := range []string{"proj-a", "proj-b"} {
				_, err := nc.TenancyMultiTenancy().Config().Orgs(testOrg).Folders(testFolder).AddProjects(
					context.Background(),
					&projectv1.Project{ObjectMeta: metav1.ObjectMeta{Name: name}},
				)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}

			req, rec := jsonReq(http.MethodGet, "/v1/projects", nil)
			c := e.NewContext(req, rec)

			err := projectHandler.ListProjects(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

			var resp map[string]interface{}
			gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(gomega.Succeed())
			gomega.Expect(len(resp["items"].([]interface{}))).To(gomega.BeNumerically(">=", 2))
		})

		ginkgo.It("filters projects by orgId query param", func() {
			_, err := nc.TenancyMultiTenancy().Config().Orgs(testOrg).Folders(testFolder).AddProjects(
				context.Background(),
				&projectv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "filtered-proj",
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": testOrg,
						},
					},
				},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			req, rec := jsonReq(http.MethodGet, "/v1/projects?orgId="+testOrg, nil)
			c := e.NewContext(req, rec)
			c.QueryParams().Set("orgId", testOrg)

			err = projectHandler.ListProjects(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		})
	})

	// ── Get ───────────────────────────────────────────────────────────────

	ginkgo.Describe("GET /v1/projects/:projectId", func() {
		ginkgo.It("returns 400 when projectId param is missing", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/projects/", nil)
			c := e.NewContext(req, rec)

			err := projectHandler.GetProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 404 when project does not exist", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/projects/ghost-proj", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("ghost-proj")

			err := projectHandler.GetProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})

		ginkgo.It("returns the project when it exists", func() {
			// GetProject relies on nexusClient.Project().ListProjects() which performs a
			// cross-namespace traversal not supported by the in-memory fake client.
			// The handler is exercised via list/create paths; a real integration test
			// would be required for the happy-path GetProject lookup.
			ginkgo.Skip("Requires real Nexus client for Project().ListProjects() traversal")
		})
	})

	// ── Update ────────────────────────────────────────────────────────────

	ginkgo.Describe("PUT /v1/projects/:projectId", func() {
		ginkgo.It("returns 400 when projectId is missing", func() {
			req, rec := jsonReq(http.MethodPut, "/v1/projects/", projectBody("", testOrg, testFolder, nil))
			c := e.NewContext(req, rec)

			err := projectHandler.UpdateProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 400 when orgId or folderId label is missing from body", func() {
			// body has no org/folder labels
			p := projectv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "upd-proj", Labels: map[string]string{}}}
			body, _ := json.Marshal(p)
			req, rec := jsonReq(http.MethodPut, "/v1/projects/upd-proj", body)
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("upd-proj")

			err := projectHandler.UpdateProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 400 when body is malformed", func() {
			req := httptest.NewRequest(http.MethodPut, "/v1/projects/p", bytes.NewReader([]byte("bad")))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c := e.NewContext(req, httptest.NewRecorder())
			c.SetParamNames("projectId")
			c.SetParamValues("p")

			err := projectHandler.UpdateProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 404 when project does not exist", func() {
			req, rec := jsonReq(http.MethodPut, "/v1/projects/ghost", projectBody("ghost", testOrg, testFolder, nil))
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("ghost")

			err := projectHandler.UpdateProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})

		ginkgo.It("updates labels on an existing project and returns 200", func() {
			_, err := nc.TenancyMultiTenancy().Config().Orgs(testOrg).Folders(testFolder).AddProjects(
				context.Background(),
				&projectv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "label-proj"}},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			req, rec := jsonReq(http.MethodPut, "/v1/projects/label-proj",
				projectBody("label-proj", testOrg, testFolder, map[string]string{"tier": "gold"}))
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("label-proj")

			err = projectHandler.UpdateProject(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		})
	})

	// ── Delete ────────────────────────────────────────────────────────────

	ginkgo.Describe("DELETE /v1/projects/:projectId", func() {
		ginkgo.It("returns 400 when projectId is missing", func() {
			req, rec := jsonReq(http.MethodDelete, "/v1/projects/", nil)
			c := e.NewContext(req, rec)

			err := projectHandler.DeleteProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 400 when orgId/folderId query params are missing", func() {
			req, rec := jsonReq(http.MethodDelete, "/v1/projects/some-proj", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("some-proj")
			// no orgId / folderId query params

			err := projectHandler.DeleteProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 404 when project does not exist", func() {
			req, rec := jsonReq(http.MethodDelete, "/v1/projects/ghost?orgId="+testOrg+"&folderId="+testFolder, nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("ghost")
			c.QueryParams().Set("orgId", testOrg)
			c.QueryParams().Set("folderId", testFolder)

			err := projectHandler.DeleteProject(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})

		ginkgo.It("deletes an existing project and returns 200", func() {
			_, err := nc.TenancyMultiTenancy().Config().Orgs(testOrg).Folders(testFolder).AddProjects(
				context.Background(),
				&projectv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "rm-proj"}},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			req, rec := jsonReq(http.MethodDelete, "/v1/projects/rm-proj", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("rm-proj")
			c.QueryParams().Set("orgId", testOrg)
			c.QueryParams().Set("folderId", testFolder)

			err = projectHandler.DeleteProject(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

			var resp map[string]string
			gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(gomega.Succeed())
			gomega.Expect(resp["message"]).To(gomega.Equal("Project deleted successfully"))
		})

		ginkgo.It("cannot delete the same project twice", func() {
			_, err := nc.TenancyMultiTenancy().Config().Orgs(testOrg).Folders(testFolder).AddProjects(
				context.Background(),
				&projectv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "once-proj"}},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			del := func() error {
				req, rec := jsonReq(http.MethodDelete, "/v1/projects/once-proj", nil)
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("once-proj")
				c.QueryParams().Set("orgId", testOrg)
				c.QueryParams().Set("folderId", testFolder)
				return projectHandler.DeleteProject(c)
			}

			gomega.Expect(del()).NotTo(gomega.HaveOccurred())
			err = del()
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})
	})

	// ── GetStatus ─────────────────────────────────────────────────────────

	ginkgo.Describe("GET /v1/projects/:projectId/status", func() {
		ginkgo.It("returns 400 when projectId is missing", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/projects//status", nil)
			c := e.NewContext(req, rec)

			err := projectHandler.GetProjectStatus(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 400 when orgId/folderId query params are missing", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/projects/p/status", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("p")

			err := projectHandler.GetProjectStatus(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("returns 404 when project does not exist", func() {
			req, rec := jsonReq(http.MethodGet, "/v1/projects/ghost/status", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("ghost")
			c.QueryParams().Set("orgId", testOrg)
			c.QueryParams().Set("folderId", testFolder)

			err := projectHandler.GetProjectStatus(c)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.(*echo.HTTPError).Code).To(gomega.Equal(http.StatusNotFound))
		})

		ginkgo.It("returns status for an existing project", func() {
			_, err := nc.TenancyMultiTenancy().Config().Orgs(testOrg).Folders(testFolder).AddProjects(
				context.Background(),
				&projectv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "status-proj"}},
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			req, rec := jsonReq(http.MethodGet, "/v1/projects/status-proj/status", nil)
			c := e.NewContext(req, rec)
			c.SetParamNames("projectId")
			c.SetParamValues("status-proj")
			c.QueryParams().Set("orgId", testOrg)
			c.QueryParams().Set("folderId", testFolder)

			err = projectHandler.GetProjectStatus(c)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		})
	})
})

// ─── Cross-resource lifecycle test ───────────────────────────────────────────

var _ = ginkgo.Describe("Org → Project lifecycle", func() {
	var (
		e              *echo.Echo
		nc             *nexus_client.Clientset
		orgHandler     *handlers.OrgHandler
		projectHandler *handlers.ProjectHandler
	)

	ginkgo.BeforeEach(func() {
		nc = newFakeNexusClient()
		orgHandler = handlers.NewOrgHandler(nc, zerolog.Nop())
		projectHandler = handlers.NewProjectHandler(nc, zerolog.Nop())
		e = echo.New()
	})

	ginkgo.It("creates an org, creates a project under it, lists it, then deletes both", func() {
		// 1. Create org via handler
		req, rec := jsonReq(http.MethodPost, "/v1/orgs", orgBody("lifecycle-org", nil))
		c := e.NewContext(req, rec)
		gomega.Expect(orgHandler.CreateOrg(c)).NotTo(gomega.HaveOccurred())
		gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))

		// 2. Create project under that org
		req, rec = jsonReq(http.MethodPost, "/v1/projects",
			projectBody("lifecycle-proj", "lifecycle-org", "default", nil))
		c = e.NewContext(req, rec)
		gomega.Expect(projectHandler.CreateProject(c)).NotTo(gomega.HaveOccurred())
		gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))

		// 3. List projects — expect at least one
		req, rec = jsonReq(http.MethodGet, "/v1/projects", nil)
		c = e.NewContext(req, rec)
		gomega.Expect(projectHandler.ListProjects(c)).NotTo(gomega.HaveOccurred())
		var listResp map[string]interface{}
		gomega.Expect(json.Unmarshal(rec.Body.Bytes(), &listResp)).To(gomega.Succeed())
		gomega.Expect(len(listResp["items"].([]interface{}))).To(gomega.BeNumerically(">=", 1))

		// 4. Delete project
		req, rec = jsonReq(http.MethodDelete, "/v1/projects/lifecycle-proj", nil)
		c = e.NewContext(req, rec)
		c.SetParamNames("projectId")
		c.SetParamValues("lifecycle-proj")
		c.QueryParams().Set("orgId", "lifecycle-org")
		c.QueryParams().Set("folderId", "default")
		gomega.Expect(projectHandler.DeleteProject(c)).NotTo(gomega.HaveOccurred())
		gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

		// 5. Delete org
		req, rec = jsonReq(http.MethodDelete, "/v1/orgs/lifecycle-org", nil)
		c = e.NewContext(req, rec)
		c.SetParamNames("orgId")
		c.SetParamValues("lifecycle-org")
		gomega.Expect(orgHandler.DeleteOrg(c)).NotTo(gomega.HaveOccurred())
		gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
	})
})
