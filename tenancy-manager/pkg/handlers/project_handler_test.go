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
	projectv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/project.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/handlers"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

var _ = ginkgo.Describe("ProjectHandler", func() {
	var (
		handler     *handlers.ProjectHandler
		e           *echo.Echo
		nexusClient *nexus_client.Clientset
	)

	ginkgo.BeforeEach(func() {
		// Create fake Nexus client
		nexusClient = nexus_client.NewFakeClient()
		nexusClient.DynamicClient = fake.NewSimpleDynamicClient(runtime.NewScheme())

		// Create handler
		logger := zerolog.Nop()
		handler = handlers.NewProjectHandler(nexusClient, logger)

		// Create Echo instance
		e = echo.New()

		// Create test organization and folder
		orgObj := &orgv1.Org{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Org",
				APIVersion: "org.edge-orchestrator.intel.com/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-org",
			},
			Spec: orgv1.OrgSpec{},
		}
		_, _ = nexusClient.TenancyMultiTenancy().Config().AddOrgs(context.Background(), orgObj)
	})

	ginkgo.Describe("CreateProject", func() {
		ginkgo.Context("when request is valid", func() {
			ginkgo.It("should create project successfully", func() {
				projectReq := projectv1.Project{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Project",
						APIVersion: "project.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-project",
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": "test-org",
							"folders.folder.edge-orchestrator.intel.com": "default",
						},
					},
					Spec: projectv1.ProjectSpec{},
				}
				body, _ := json.Marshal(projectReq)
				req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.CreateProject(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))

				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(response["message"]).To(gomega.Equal("Project created successfully"))
			})
		})

		ginkgo.Context("when folderId is not provided", func() {
			ginkgo.It("should default to 'default' folderId", func() {
				projectReq := projectv1.Project{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Project",
						APIVersion: "project.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-project-default",
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": "test-org",
						},
					},
					Spec: projectv1.ProjectSpec{},
				}
				body, _ := json.Marshal(projectReq)
				req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.CreateProject(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusCreated))

				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				project := response["project"].(map[string]interface{})
			metadata := project["metadata"].(map[string]interface{})
			labels := metadata["labels"].(map[string]interface{})
			gomega.Expect(labels["folders.folder.edge-orchestrator.intel.com"]).To(gomega.Equal("default"))
			})
		})

		ginkgo.Context("when request body is invalid", func() {
			ginkgo.It("should return bad request error", func() {
				req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewReader([]byte("invalid json")))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.CreateProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})

		ginkgo.Context("when project name is missing", func() {
			ginkgo.It("should return bad request error", func() {
				projectReq := projectv1.Project{
					Spec: projectv1.ProjectSpec{					},
				}
				body, _ := json.Marshal(projectReq)
				req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.CreateProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
				gomega.Expect(httpErr.Message).To(gomega.Equal("Project name is required"))
			})
		})

		ginkgo.Context("when orgId is missing", func() {
			ginkgo.It("should return bad request error", func() {
				projectReq := projectv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-project",
					},
					Spec: projectv1.ProjectSpec{},
				}
				body, _ := json.Marshal(projectReq)
				req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.CreateProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			gomega.Expect(httpErr.Message).To(gomega.ContainSubstring("OrgId is required"))
			})
		})
	})

	ginkgo.Describe("ListProjects", func() {
		ginkgo.BeforeEach(func() {
			// Create test projects
			projectObj1 := &projectv1.Project{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Project",
					APIVersion: "project.edge-orchestrator.intel.com/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "list-project-1",
					Labels: map[string]string{
						"orgs.org.edge-orchestrator.intel.com": "test-org",
						"folders.folder.edge-orchestrator.intel.com": "default",
					},
				},
				Spec: projectv1.ProjectSpec{},
			}
			projectObj2 := &projectv1.Project{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Project",
					APIVersion: "project.edge-orchestrator.intel.com/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "list-project-2",
					Labels: map[string]string{
						"orgs.org.edge-orchestrator.intel.com": "test-org",
						"folders.folder.edge-orchestrator.intel.com": "folder1",
					},
				},
				Spec: projectv1.ProjectSpec{},
			}
			_, _ = nexusClient.TenancyMultiTenancy().Config().Orgs("test-org").Folders("default").AddProjects(context.Background(), projectObj1)
			_, _ = nexusClient.TenancyMultiTenancy().Config().Orgs("test-org").Folders("folder1").AddProjects(context.Background(), projectObj2)
		})

		ginkgo.Context("when listing all projects", func() {
			ginkgo.It("should return all projects", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.ListProjects(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				items := response["items"].([]interface{})
				gomega.Expect(len(items)).To(gomega.BeNumerically(">=", 2))
			})
		})

		ginkgo.Context("when filtering by orgId", func() {
			ginkgo.It("should return projects for specified org", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/projects?orgId=test-org", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.ListProjects(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				items := response["items"].([]interface{})
				gomega.Expect(len(items)).To(gomega.BeNumerically(">=", 2))
			})
		})

		ginkgo.Context("when filtering by orgId and folderId", func() {
			ginkgo.It("should return projects for specified org and folder", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/projects?orgId=test-org&folderId=default", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.ListProjects(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				items := response["items"].([]interface{})
				gomega.Expect(len(items)).To(gomega.BeNumerically(">=", 1))

				// Verify all returned projects have the correct folderId
				for _, item := range items {
					project := item.(map[string]interface{})
					metadata := project["metadata"].(map[string]interface{})
					labels := metadata["labels"].(map[string]interface{})
					gomega.Expect(labels["folders.folder.edge-orchestrator.intel.com"]).To(gomega.Equal("default"))
				}
			})
		})
	})

	ginkgo.Describe("GetProject", func() {
		ginkgo.Context("when project exists", func() {
			// Skip: Fake client doesn't properly persist objects for GetByName retrieval
			ginkgo.It("should return project [Integration]", func() {
				ginkgo.Skip("Requires real Nexus client for GetProjectByName")
				// Create test project
				projectObj := &projectv1.Project{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Project",
						APIVersion: "project.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "get-test-project",
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": "test-org",
							"folders.folder.edge-orchestrator.intel.com": "default",
						},
					},
					Spec: projectv1.ProjectSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().
					Orgs("test-org").Folders("default").AddProjects(context.Background(), projectObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				req := httptest.NewRequest(http.MethodGet, "/v1/projects/get-test-project", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("get-test-project")

				err = handler.GetProject(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response projectv1.Project
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(response.Name).To(gomega.Equal("get-test-project"))
			})
		})

		ginkgo.Context("when project does not exist", func() {
			ginkgo.It("should return not found error", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/projects/nonexistent", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("nonexistent")

				err := handler.GetProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})

		ginkgo.Context("when projectId is missing", func() {
			ginkgo.It("should return bad request error", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/projects/", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := handler.GetProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})
	})

	ginkgo.Describe("UpdateProject", func() {
		ginkgo.Context("when project exists", func() {
			ginkgo.It("should update project successfully", func() {
				// Create test project
				projectObj := &projectv1.Project{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Project",
						APIVersion: "project.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "update-test-project",
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": "test-org",
							"folders.folder.edge-orchestrator.intel.com": "default",
						},
					},
					Spec: projectv1.ProjectSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().
					Orgs("test-org").Folders("default").AddProjects(context.Background(), projectObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				updateReq := projectv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": "test-org",
							"folders.folder.edge-orchestrator.intel.com": "default",
						},
					},
					Spec: projectv1.ProjectSpec{},
				}
				body, _ := json.Marshal(updateReq)
				req := httptest.NewRequest(http.MethodPut, "/v1/projects/update-test-project", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("update-test-project")

				err = handler.UpdateProject(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
			})
		})

		ginkgo.Context("when project does not exist", func() {
			ginkgo.It("should return not found error", func() {
				updateReq := projectv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": "test-org",
							"folders.folder.edge-orchestrator.intel.com": "default",
						},
					},
					Spec: projectv1.ProjectSpec{},
				}
				body, _ := json.Marshal(updateReq)
				req := httptest.NewRequest(http.MethodPut, "/v1/projects/nonexistent", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("nonexistent")

				err := handler.UpdateProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})

		ginkgo.Context("when orgId or folderId is missing", func() {
			ginkgo.It("should return bad request error", func() {
				updateReq := projectv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"env": "test"},
					},
					Spec: projectv1.ProjectSpec{},
				}
				body, _ := json.Marshal(updateReq)
				req := httptest.NewRequest(http.MethodPut, "/v1/projects/test-project", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("test-project")

				err := handler.UpdateProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})
	})

	ginkgo.Describe("DeleteProject", func() {
		ginkgo.Context("when project exists", func() {
			ginkgo.It("should delete project successfully", func() {
				// Create test project
				projectObj := &projectv1.Project{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Project",
						APIVersion: "project.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "delete-test-project",
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": "test-org",
							"folders.folder.edge-orchestrator.intel.com": "default",
						},
					},
					Spec: projectv1.ProjectSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().
					Orgs("test-org").Folders("default").AddProjects(context.Background(), projectObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				req := httptest.NewRequest(http.MethodDelete, "/v1/projects/delete-test-project?orgId=test-org&folderId=default", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("delete-test-project")

				err = handler.DeleteProject(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))

				var response map[string]string
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(response["message"]).To(gomega.Equal("Project deleted successfully"))
			})
		})

		ginkgo.Context("when project does not exist", func() {
			ginkgo.It("should return not found error", func() {
				req := httptest.NewRequest(http.MethodDelete, "/v1/projects/nonexistent?orgId=test-org&folderId=default", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("nonexistent")

				err := handler.DeleteProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})

		ginkgo.Context("when orgId or folderId is missing", func() {
			ginkgo.It("should return bad request error", func() {
				req := httptest.NewRequest(http.MethodDelete, "/v1/projects/test-project", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("test-project")

				err := handler.DeleteProject(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})
	})

	ginkgo.Describe("GetProjectStatus", func() {
		ginkgo.Context("when project exists", func() {
			ginkgo.It("should return project status", func() {
				// Create test project
				projectObj := &projectv1.Project{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Project",
						APIVersion: "project.edge-orchestrator.intel.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "status-test-project",
						Labels: map[string]string{
							"orgs.org.edge-orchestrator.intel.com": "test-org",
							"folders.folder.edge-orchestrator.intel.com": "default",
						},
					},
					Spec: projectv1.ProjectSpec{},
				}
				_, err := nexusClient.TenancyMultiTenancy().Config().
					Orgs("test-org").Folders("default").AddProjects(context.Background(), projectObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				req := httptest.NewRequest(http.MethodGet, "/v1/projects/status-test-project/status?orgId=test-org&folderId=default", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("status-test-project")

				err = handler.GetProjectStatus(c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
			})
		})

		ginkgo.Context("when project does not exist", func() {
			ginkgo.It("should return not found error", func() {
				req := httptest.NewRequest(http.MethodGet, "/v1/projects/nonexistent/status?orgId=test-org&folderId=default", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("projectId")
				c.SetParamValues("nonexistent")

				err := handler.GetProjectStatus(c)
				gomega.Expect(err).To(gomega.HaveOccurred())
				httpErr, ok := err.(*echo.HTTPError)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(httpErr.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})
	})
})
