// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package declarative

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/utils"
)

type EndpointContext struct {
	echo.Context

	SpecURI      string
	Method       string     // e.g. PUT
	KindName     string     // e.g. GlobalNamespace
	ResourceName string     // e.g. globalnamespaces
	GroupName    string     // e.g. vmware.org
	CrdName      string     // e.g. globalnamespaces.vmware.org
	Params       [][]string // e.g. [id, projectId, gnsId]
	Identifier   string     // e.g. id or fqdn

	Single bool // used to identify which k8s endpoint we should use (resource/:name or resource/)

	SchemaName string // OpenAPI.components.schema name used to create yaml spec
	ShortName  string
	ShortURI   string
	URI        string
}

const (
	resourcePattern          = "/apis/%s/v1/%s"
	resourceShortPattern     = "/apis/v1/%s"
	resourceNamePattern      = resourcePattern + "/:name"
	resourceNameShortPattern = resourceShortPattern + "/:name"
)

func SetupContext(uri, method string, item *openapi3.Operation) *EndpointContext {
	kindName := GetExtensionVal(item, NexusKindName)
	groupName := GetExtensionVal(item, NexusGroupName)
	shortName := GetExtensionVal(item, NexusShortName)
	resourceName := strings.ToLower(utils.ToPlural(kindName))
	crdName := resourceName + "." + groupName
	requiredParams := extractURIParams(uri)
	identifier := GetExtensionVal(item, "x-nexus-identifier")

	path := fmt.Sprintf(resourcePattern, groupName, resourceName)
	shortPath := fmt.Sprintf(resourceShortPattern, shortName)
	single := false
	if identifier != "" && method != http.MethodPut {
		single = true
		path = fmt.Sprintf(resourceNamePattern, groupName, resourceName)
		shortPath = fmt.Sprintf(resourceNameShortPattern, shortName)
	}

	schemaName := ""
	if item.RequestBody != nil && item.RequestBody.Value != nil {
		mediaType := item.RequestBody.Value.Content.Get("application/json")
		if mediaType != nil {
			schemaName = openapi3.DefaultRefNameResolver(mediaType.Schema.Ref)
		}
	}

	if shortName == "" {
		shortPath = ""
	}

	return &EndpointContext{
		SpecURI:      uri,
		KindName:     kindName,
		ResourceName: resourceName,
		GroupName:    groupName,
		CrdName:      crdName,
		Params:       requiredParams,
		Identifier:   identifier,
		Single:       single,
		URI:          path,
		Method:       method,
		SchemaName:   schemaName,
		ShortName:    shortName,
		ShortURI:     shortPath,
	}
}

func IsArrayResponse(op *openapi3.Operation) bool {
	if op == nil {
		return false
	}

	resp := op.Responses.Get(http.StatusOK)
	if resp == nil {
		return false
	}

	mediaType := resp.Value.Content.Get("application/json")
	if mediaType == nil {
		return false
	}

	if mediaType.Schema.Value.Type == "array" {
		return true
	}

	return false
}

func extractURIParams(uri string) [][]string {
	r := regexp.MustCompile(`{([^{}]+)}`)
	params := r.FindAllStringSubmatch(uri, -1)
	if len(params) == 0 {
		return [][]string{}
	}
	return params
}
