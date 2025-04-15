// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package authz_test

import (
	"net/http"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/auth/authn"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/auth/authz"
)

var _ = ginkgo.Describe("Authorization", func() {
	ginkgo.Describe("IAM Admin Persona - Tenancy Orgs", func() {
		var jwtClaims authn.JwtData
		ginkgo.Context("valid request metadata", func() {
			ginkgo.It("should return accepted for orgs list", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1/orgs",
					Method: "get",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"org-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for fetching an org", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1/orgs/test-org",
					Method: "get",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"org-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for updating an org", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1/orgs/test-org",
					Method: "put",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"org-write-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for creating an org", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1/orgs/test-org2",
					Method: "put",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"org-write-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for deleting an org", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1/orgs/test-org2",
					Method: "delete",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"org-delete-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
		})
		ginkgo.Context("version changes in api", func() {
			ginkgo.It("should return accepted for v1beta", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1beta/orgs",
					Method: "get",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"org-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return unauthorized for alphav1", func() {
				jwtClaims = authn.JwtData{
					URN:    "/alphav1/orgs",
					Method: "get",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"org-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
		ginkgo.Context("invalid/unavailable user claims", func() {
			ginkgo.It("should return unauthorized for org delete", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1/orgs/testorg",
					Method: "delete",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"org-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for orgs list", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1/orgs",
					Method: "get",
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
	})
	ginkgo.Describe("Org Admin Persona - Tenancy Projects", func() {
		var jwtClaims authn.JwtData
		ginkgo.Context("valid request metadata", func() {
			ginkgo.It("should return accepted for projects list", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62-e3315546a473",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62-e3315546a473_project-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for projects list with member access without org admin role", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for projects list with member access "+
				"without org admin role and active project", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for projects list with multiple member roles "+
				"without org admin role and active project", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
								"f829cb3a-6a90-11ef-9f62_5eef3a98-db80-b537-11ef_member-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for fetching a project", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62-e3315546a473",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62-e3315546a473_project-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for fetching a project with member access without org admin role", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return unauthorized for fetching a project without correct member access and org role", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"kam25k2a-9sk1-09nk-1gt7_db803a98-5eef-11ef-b537_member-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for fetching a project without correct member access and org role", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_kam25k2a-9sk1-09nk-1gt7_member-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return accepted for updating a project", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj",
					Method:      "put",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62-e3315546a473",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62-e3315546a473_project-write-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return unauthorized for updating a project with member access", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj",
					Method:      "put",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return accepted for creating a project", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj2",
					Method:      "put",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62-e3315546a473",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62-e3315546a473_project-write-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for deleting a project", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj2",
					Method:      "delete",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62-e3315546a473",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62-e3315546a473_project-delete-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return unauthorized for updating a project with member access", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj",
					Method:      "delete",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
		ginkgo.Context("version changes in apis", func() {
			ginkgo.It("should return accepted for v2alpha", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v2alpha/projects",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62-e3315546a473",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62-e3315546a473_project-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return unauthorized for v2-alpha", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v2-alpha/projects",
					Method:      "get",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62-e3315546a473",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62-e3315546a473_project-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
		ginkgo.Context("invalid/unavailable user claims", func() {
			ginkgo.It("should return unauthorized for project update", func() {
				jwtClaims = authn.JwtData{
					URN:         "/v1/projects/test-proj",
					Method:      "put",
					ActiveOrgID: "f829cb3a-6a90-11ef-9f62-e3315546a473",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62-e3315546a473_project-read-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for projects list", func() {
				jwtClaims = authn.JwtData{
					URN:    "/v1/projects",
					Method: "get",
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
	})
	ginkgo.Describe("OEP User Persona", func() {
		var jwtClaims authn.JwtData
		ginkgo.Context("valid request metadata", func() {
			ginkgo.It("should return accepted for fetching a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-west",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			// TODO: remove this test, it is for checking old member role
			ginkgo.It("should return accepted for fetching a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-west",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for updating a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-east",
					Method:          "put",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for creating a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions",
					Method:          "post",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for deleting a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-west",
					Method:          "delete",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
		})
		// TODO: remove below
		ginkgo.Context("valid request metadata using old member role", func() {
			ginkgo.It("should return accepted for fetching a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-west",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			// TODO: remove this test, it is for checking old member role
			ginkgo.It("should return accepted for fetching a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-west",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for fetching a region with short member role name", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-west",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for updating a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-east",
					Method:          "put",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for creating a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions",
					Method:          "post",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for deleting a region", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-west",
					Method:          "delete",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
		})
		// TODO: remove above
		ginkgo.Context("version changes in apis", func() {
			ginkgo.It("should return accepted for v1alpha", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1alpha/projects/test-proj/regions/us-west",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return unauthorized for v-1", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v-1/projects/test-proj/regions/us-west",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
		ginkgo.Context("invalid request metadata", func() {
			ginkgo.It("should return unauthorized for invalid ActiveProjectId", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "f829cb3a-6a90-11ef-9f62",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for missing ActiveOrgId", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions",
					Method:          "get",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for unavailable user claim", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62-e3315546a473",
					ActiveProjectID: "db803a98-5eef-11ef-b537-7f429e622bce",
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
	})
	ginkgo.Describe("OEP Network tests", func() {
		var jwtClaims authn.JwtData
		ginkgo.Context("valid request metadata", func() {
			ginkgo.It("should return accepted for getting a network", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m",
								"db803a98-5eef-11ef-b537_ao-rw",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for getting a network with short member role name", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m",
								"db803a98-5eef-11ef-b537_app-deployment-manager-read-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for getting all networks", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m",
								"db803a98-5eef-11ef-b537_ao-rw",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			ginkgo.It("should return accepted for creating a network", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "put",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m",
								"db803a98-5eef-11ef-b537_ao-rw",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			// TODO: remove
			ginkgo.It("should return accepted for getting a network with new role and old member", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
								"db803a98-5eef-11ef-b537_ao-rw",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			// TODO: remove
			ginkgo.It("should return accepted for getting a network with old role and old member", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
								"db803a98-5eef-11ef-b537_app-deployment-manager-read-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			// TODO: remove
			ginkgo.It("should return accepted for getting a network with old role and new member", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m",
								"db803a98-5eef-11ef-b537_app-deployment-manager-read-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
			// TODO: remove
			ginkgo.It("should return accepted for creating a network with old role", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "put",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_member-role",
								"db803a98-5eef-11ef-b537_app-deployment-manager-write-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusAccepted))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusAccepted)))
			})
		})
		ginkgo.Context("invalid request metadata", func() {
			ginkgo.It("should return unauthorized for creating a network without a member role", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "put",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"db803a98-5eef-11ef-b537_interconnect-write-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return refused for getting a network without read role", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return refused for getting all networks without read role", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return refused for creating a network without write role", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "put",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return refused for creating a network with read but without write role", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/networks/net1",
					Method:          "put",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{
								"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m",
								"db803a98-5eef-11ef-b537_app-deployment-manager-read-role",
							},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
	})
	ginkgo.Describe("Invalid AuthZ requests", func() {
		var jwtClaims authn.JwtData
		ginkgo.Context("errors in request metadata to authz", func() {
			ginkgo.It("should return unauthorized for no input", func() {
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for missing method", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v1/projects/test-proj/regions/us-west",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for missing URN", func() {
				jwtClaims = authn.JwtData{
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for urn not in scope", func() {
				jwtClaims = authn.JwtData{
					URN:             "/",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
			ginkgo.It("should return unauthorized for version as v.1", func() {
				jwtClaims = authn.JwtData{
					URN:             "/v.1/projects/test-proj/regions/us-west",
					Method:          "get",
					ActiveOrgID:     "f829cb3a-6a90-11ef-9f62",
					ActiveProjectID: "db803a98-5eef-11ef-b537",
					Claims: authn.TokenData{
						RealmAccess: authn.RealmAccess{
							Roles: []string{"f829cb3a-6a90-11ef-9f62_db803a98-5eef-11ef-b537_m"},
						},
					},
				}
				httpError := authz.VerifyAuthorization(jwtClaims)
				gomega.Expect(httpError.Code).To(gomega.Equal(http.StatusUnauthorized))
				gomega.Expect(httpError.Message).To(gomega.Equal(http.StatusText(http.StatusUnauthorized)))
			})
		})
	})
})

func FuzzVerifyAuthorization(f *testing.F) {
	// Seed the fuzzer with initial valid input
	f.Add("/v1/orgs", "get", "org-read-role")
	f.Fuzz(func(t *testing.T, urn string, method string, role string) {
		jwtClaims := authn.JwtData{
			URN:    urn,
			Method: method,
			Claims: authn.TokenData{
				RealmAccess: authn.RealmAccess{
					Roles: []string{role},
				},
			},
		}
		httpError := authz.VerifyAuthorization(jwtClaims)
		if httpError.Code != http.StatusAccepted && httpError.Code != http.StatusUnauthorized {
			t.Errorf("unexpected error code: got %v, want %v or %v",
				httpError.Code, http.StatusAccepted, http.StatusUnauthorized)
		}
		if httpError.Message != http.StatusText(http.StatusAccepted) &&
			httpError.Message != http.StatusText(http.StatusUnauthorized) {
			t.Errorf("unexpected error message: got %v, want %v or %v",
				httpError.Message, http.StatusText(http.StatusAccepted), http.StatusText(http.StatusUnauthorized))
		}
	})
}

func FuzzVerifyAuthzForProjects(f *testing.F) {
	// Seed the fuzzer with initial valid input
	f.Add("/v1/projects", "get", "abcd", "efgh")
	f.Fuzz(func(t *testing.T, urn, method, activeOrgId, activeProjectId string) {
		jwtClaims := authn.JwtData{
			URN:             urn,
			Method:          method,
			ActiveOrgID:     activeOrgId,
			ActiveProjectID: activeProjectId,
			Claims: authn.TokenData{
				RealmAccess: authn.RealmAccess{
					Roles: []string{activeOrgId + "_project-read-role", activeOrgId + "_" + activeProjectId + "_m"},
				},
			},
		}
		httpError := authz.VerifyAuthorization(jwtClaims)
		if httpError.Code != http.StatusAccepted && httpError.Code != http.StatusUnauthorized {
			t.Errorf("unexpected error code: got %v, want %v or %v",
				httpError.Code, http.StatusAccepted, http.StatusUnauthorized)
		}
		if httpError.Message != http.StatusText(http.StatusAccepted) &&
			httpError.Message != http.StatusText(http.StatusUnauthorized) {
			t.Errorf("unexpected error message: got %v, want %v or %v",
				httpError.Message, http.StatusText(http.StatusAccepted), http.StatusText(http.StatusUnauthorized))
		}
	})
}

func FuzzAuthzForProjectsMemberRole(f *testing.F) {
	// Seed the fuzzer with initial valid input
	f.Add("/v1/projects?member-type=true", "get", "abcd", "efgh")
	f.Fuzz(func(t *testing.T, urn, method, activeOrgId, activeProjectId string) {
		jwtClaims := authn.JwtData{
			URN:             urn,
			Method:          method,
			ActiveOrgID:     activeOrgId,
			ActiveProjectID: activeProjectId,
			Claims: authn.TokenData{
				RealmAccess: authn.RealmAccess{
					Roles: []string{activeOrgId + "_project-read-role", activeOrgId + "_" + activeProjectId + "_m"},
				},
			},
		}
		httpError := authz.VerifyAuthorization(jwtClaims)
		if httpError.Code != http.StatusAccepted && httpError.Code != http.StatusUnauthorized {
			t.Errorf("unexpected error code: got %v, want %v or %v",
				httpError.Code, http.StatusAccepted, http.StatusUnauthorized)
		}
		if httpError.Message != http.StatusText(http.StatusAccepted) &&
			httpError.Message != http.StatusText(http.StatusUnauthorized) {
			t.Errorf("unexpected error message: got %v, want %v or %v",
				httpError.Message, http.StatusText(http.StatusAccepted), http.StatusText(http.StatusUnauthorized))
		}
	})
}
