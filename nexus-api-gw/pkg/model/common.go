// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"strings"

	"github.com/vmware-tanzu/graph-framework-for-microservices/nexus/nexus"
)

type EventType string

const (
	Upsert EventType = "Upsert"
	Delete EventType = "Delete"
)

//nolint:tagliatelle // This struct has dependency on nexus's field name.
type NexusAnnotation struct {
	Name                 string                     `json:"name,omitempty"`
	Hierarchy            []string                   `json:"hierarchy,omitempty"`
	Children             map[string]NodeHelperChild `json:"children,omitempty"`
	Links                map[string]NodeHelperChild `json:"links,omitempty"`
	NexusRestAPIGen      nexus.RestAPISpec          `json:"nexus-rest-api-gen,omitempty"`
	NexusRestAPIMappings map[string]string          `json:"nexus-rest-api-mappings,omitempty"`
	IsSingleton          bool                       `json:"is_singleton,omitempty"`
	Description          string                     `json:"description,omitempty"`
	DeferredDelete       bool                       `json:"deferred-delete,omitempty"`
}

type NodeHelperChild struct {
	FieldName    string `json:"fieldName"`
	FieldNameGvk string `json:"fieldNameGvk"`
	IsNamed      bool   `json:"isNamed"`
}

type NodeInfo struct {
	Name            string
	ParentHierarchy []string
	Children        map[string]NodeHelperChild
	Links           map[string]NodeHelperChild
	IsSingleton     bool
	Description     string
	DeferredDelete  bool
}

type RestURIInfo struct {
	TypeOfURI URIType
}

type URIType int

const (
	DefaultURI URIType = iota
	SingleLinkURI
	NamedLinkURI
	StatusURI
)

func ConstructEchoPathParamURL(uri string) string {
	replacer := strings.NewReplacer("{", ":", "}", "")
	return replacer.Replace(uri)
}

type DatamodelInfo struct {
	Title string
}

// LinkGvk : This model used to carry fully qualified object <gvk> and
// hierarchy information.
type LinkGvk struct {
	Group     string   `json:"group,omitempty" yaml:"group,omitempty"`
	Kind      string   `json:"kind,omitempty" yaml:"kind,omitempty"`
	Name      string   `json:"name,omitempty" yaml:"name,omitempty"`
	Hierarchy []string `json:"hierarchy,omitempty" yaml:"hierarchy,omitempty"`
}
