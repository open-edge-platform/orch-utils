// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package echoserver_test

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/auth/authn"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	nc "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestEchoServer(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "EchoServer Suite")
}

var jwtOrgName string

var (
	URIToCRDType = map[string]string{
		"/v1/orgs":                                                         "orgs.org.edge-orchestrator.intel.com",
		"/v1/orgs/{org.Org}/status":                                        "orgs.org.edge-orchestrator.intel.com",
		"/v1/orgs/{org.Org}/Folders":                                       "orgs.org.edge-orchestrator.intel.com",
		"/v1/orgs/{org.Org}/License":                                       "orgs.org.edge-orchestrator.intel.com",
		"/v1/projects/{project.Project}":                                   "projects.project.edge-orchestrator.intel.com",
		"/v1/orgs/{org.Org}/licenses":                                      "licenses.license.edge-orchestrator.intel.com",
		"/v1/projects/{project.Project}/networks/{network.Network}":        "networks.network.edge-orchestrator.intel.com",
		"/v1/projects/{project.Project}/status":                            "projects.project.edge-orchestrator.intel.com",
		"/v1/projects/{project.Project}/networks":                          "networks.network.edge-orchestrator.intel.com",
		"/v1/projects":                                                     "projects.project.edge-orchestrator.intel.com",
		"/v1/projects/{project.Project}/Networks":                          "projects.project.edge-orchestrator.intel.com",
		"/v1/projects/{project.Project}/networks/{network.Network}/status": "networks.network.edge-orchestrator.intel.com",
		"/v1/orgs/{org.Org}":                                               "orgs.org.edge-orchestrator.intel.com",
		"/v1/orgs/{org.Org}/licenses/{license.License}":                    "licenses.license.edge-orchestrator.intel.com",
		"/v1/orgs/{org.Org}/licenses/{license.License}/status":             "licenses.license.edge-orchestrator.intel.com",
	}
	GRVToListKind = map[schema.GroupVersionResource]string{
		{
			Group:    "gns.vmware.org",
			Version:  "v1",
			Resource: "globalnamespaces",
		}: "GlobalNamespaceList",
		{
			Group:    "root.vmware.org",
			Version:  "v1",
			Resource: "roots",
		}: "RootList",
		{
			Group:    "orgchart.vmware.org",
			Version:  "v1",
			Resource: "leaders",
		}: "LeaderList",
		{
			Group:    "org.edge-orchestrator.intel.com",
			Version:  "v1",
			Resource: "orgs",
		}: "OrgList", {
			Group:    "project.edge-orchestrator.intel.com",
			Version:  "v1",
			Resource: "projects",
		}: "ProjectList",
	}

	CrdTypeToNodeInfo = map[string]model.NodeInfo{
		"orgs.org.edge-orchestrator.intel.com": {
			Name: "org.Org",
			ParentHierarchy: []string{
				"multitenancies.tenancy.edge-orchestrator.intel.com",
				"configs.config.edge-orchestrator.intel.com",
			},
			Children: map[string]model.NodeHelperChild{
				"folders.folder.edge-orchestrator.intel.com": {
					FieldName:    "Folders",
					FieldNameGvk: "foldersGvk",
					IsNamed:      true,
				},
				"licenses.license.edge-orchestrator.intel.com": {
					FieldName:    "License",
					FieldNameGvk: "licenseGvk",
					IsNamed:      false,
				},
			},
			IsSingleton:    false,
			DeferredDelete: true,
		},
		"projects.project.edge-orchestrator.intel.com": {
			Name: "project.Project",
			ParentHierarchy: []string{
				"multitenancies.tenancy.edge-orchestrator.intel.com",
				"configs.config.edge-orchestrator.intel.com",
				"orgs.org.edge-orchestrator.intel.com", "folders.folder.edge-orchestrator.intel.com",
			},
			Children: map[string]model.NodeHelperChild{
				"networks.network.edge-orchestrator.intel.com": {
					FieldName:    "Networks",
					FieldNameGvk: "networksGvk",
					IsNamed:      true,
				},
			},
			IsSingleton:    false,
			DeferredDelete: true,
		},
		"licenses.license.edge-orchestrator.intel.com": {
			Name: "license.License",
			ParentHierarchy: []string{
				"multitenancies.tenancy.edge-orchestrator.intel.com",
				"configs.config.edge-orchestrator.intel.com",
				"orgs.org.edge-orchestrator.intel.com",
			},
			Children:       map[string]model.NodeHelperChild{},
			IsSingleton:    false,
			DeferredDelete: false,
		},
	}
)

type MockAuthenticator struct{}

func (a *MockAuthenticator) VerifyJWT(_ echo.Context, _ *nc.Clientset, _ bool) (authn.JwtData, *echo.HTTPError) {
	return authn.JwtData{
		OrgName: jwtOrgName,
	}, nil
}

func (a *MockAuthenticator) VerifyAuthorization(_ authn.JwtData) *echo.HTTPError {
	return nil
}

func (a *MockAuthenticator) AuthenticateAndAuthorize(_ echo.Context,
	_ *nc.Clientset,
) (authn.JwtData, *echo.HTTPError) {
	return authn.JwtData{
		OrgName: jwtOrgName,
	}, nil
}

func constructOrgGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "org.edge-orchestrator.intel.com",
		Version:  "v1",
		Resource: "orgs",
	}
}

func constructUnstructuredOrg(hashedName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "org.edge-orchestrator.intel.com/v1",
			"kind":       "Org",
			"metadata": map[string]interface{}{
				"name":            hashedName,
				"resourceVersion": "1",
			},
		},
	}
}
