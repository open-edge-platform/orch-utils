// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"

	amcv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/apimappingconfig.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DisplayNameLblConst = "nexus/display_name"

var NexusApps = []string{
	"nexus-tenant-runtime",
	"tsm-tenant-runtime",
}

var AllowedStates = []string{
	"Synced",
}

// Project represents the structure you want to store.
type Project struct {
	UID     string
	Name    string
	Org     Org
	Deleted bool
}

// Org represents organization details.
type Org struct {
	Name    string
	UID     string
	Deleted bool
}

// API Remapping.
type APIMappingVO struct {
	ServiceURI string        `json:"serviceUri" yaml:"serviceUri"`
	Backend    amcv1.Backend `json:"backend" yaml:"backend"`
}

func GetConfigNode(nexusClient *nexus_client.Clientset, name string) (configNode *nexus_client.ConfigConfig, err error) {
	configNodes, err := nexusClient.Config().ListConfigs(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", DisplayNameLblConst, name),
	})
	if err != nil {
		return &nexus_client.ConfigConfig{}, err
	}
	for _, configNode := range configNodes {
		return configNode, nil
	}
	return &nexus_client.ConfigConfig{}, fmt.Errorf("config node with name %s not found", name)
}

func GetRuntimeNode(nexusClient *nexus_client.Clientset, name string) (configNode *nexus_client.RuntimeRuntime, err error) {
	runtimeNodes, err := nexusClient.Runtime().ListRuntimes(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", DisplayNameLblConst, name),
	})
	if err != nil {
		return &nexus_client.RuntimeRuntime{}, err
	}
	for _, runtimeNode := range runtimeNodes {
		return runtimeNode, nil
	}
	return &nexus_client.RuntimeRuntime{}, fmt.Errorf("config node with name %s not found", name)
}
