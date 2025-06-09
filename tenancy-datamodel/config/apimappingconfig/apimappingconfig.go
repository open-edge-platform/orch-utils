// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package apimappingconfig

import (
	"github.com/open-edge-platform/orch-utils/tenancy-datamodel/nexus/base/nexus"
)

// NOTE: This struct must be kept in sync with the same config in
// tenancy-api-mapping/pkg/config/types.go (which has YAML format struct tags
// instead of JSON)

//nolint:tagliatelle // Per requirement.
type APIMappingConfig struct {
	nexus.Node

	SpecGenEnabled bool         `json:"specGenEnabled"`
	RepoConf       RepoConf     `json:"repoConf,omitempty"`
	DownloadConf   DownloadConf `json:"downloadConf,omitempty"`
	Mappings       []Mapping    `json:"mappings"`
	Backend        Backend      `json:"backend"`
}

//nolint:tagliatelle // Per requirement.
type RepoConf struct {
	URL          string `json:"url"`
	Tag          string `json:"tag"`
	SpecFilePath string `json:"specFilePath"`
}

//nolint:tagliatelle // Per requirement.
type DownloadConf struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

//nolint:tagliatelle // Per requirement.
type Mapping struct {
	ExternalURI string `json:"externalURI"`
	ServiceURI  string `json:"serviceURI"`
}

type Backend struct {
	Service string `json:"service"`
	Port    uint32 `json:"port"`
}
