// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package helper

//nolint:gci // generated imports.
import (
	"context"
	//nolint:gosec // only useful fixed strig hashing and not for security.
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/elliotchance/orderedmap"

	datamodel "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultKey = "default"
const DisplayNameLabel = "nexus/display_name"
const IsNameHashedLabel = "nexus/is_name_hashed"

//nolint:lll // Generated code. Length depends on actual graph depth.
func GetCRDParentsMap() map[string][]string {
	return map[string][]string{
		"apimappingconfigs.apimappingconfig.edge-orchestrator.intel.com":         {"multitenancies.tenancy.edge-orchestrator.intel.com", "configs.config.edge-orchestrator.intel.com"},
		"configs.config.edge-orchestrator.intel.com":                             {"multitenancies.tenancy.edge-orchestrator.intel.com"},
		"folders.folder.edge-orchestrator.intel.com":                             {"multitenancies.tenancy.edge-orchestrator.intel.com", "configs.config.edge-orchestrator.intel.com", "orgs.org.edge-orchestrator.intel.com"},
		"multitenancies.tenancy.edge-orchestrator.intel.com":                     {},
		"networks.network.edge-orchestrator.intel.com":                           {"multitenancies.tenancy.edge-orchestrator.intel.com", "configs.config.edge-orchestrator.intel.com", "orgs.org.edge-orchestrator.intel.com", "folders.folder.edge-orchestrator.intel.com", "projects.project.edge-orchestrator.intel.com"},
		"orgactivewatchers.orgactivewatcher.edge-orchestrator.intel.com":         {"multitenancies.tenancy.edge-orchestrator.intel.com", "runtimes.runtime.edge-orchestrator.intel.com", "runtimeorgs.runtimeorg.edge-orchestrator.intel.com"},
		"orgs.org.edge-orchestrator.intel.com":                                   {"multitenancies.tenancy.edge-orchestrator.intel.com", "configs.config.edge-orchestrator.intel.com"},
		"orgwatchers.orgwatcher.edge-orchestrator.intel.com":                     {"multitenancies.tenancy.edge-orchestrator.intel.com", "configs.config.edge-orchestrator.intel.com"},
		"projectactivewatchers.projectactivewatcher.edge-orchestrator.intel.com": {"multitenancies.tenancy.edge-orchestrator.intel.com", "runtimes.runtime.edge-orchestrator.intel.com", "runtimeorgs.runtimeorg.edge-orchestrator.intel.com", "runtimefolders.runtimefolder.edge-orchestrator.intel.com", "runtimeprojects.runtimeproject.edge-orchestrator.intel.com"},
		"projects.project.edge-orchestrator.intel.com":                           {"multitenancies.tenancy.edge-orchestrator.intel.com", "configs.config.edge-orchestrator.intel.com", "orgs.org.edge-orchestrator.intel.com", "folders.folder.edge-orchestrator.intel.com"},
		"projectwatchers.projectwatcher.edge-orchestrator.intel.com":             {"multitenancies.tenancy.edge-orchestrator.intel.com", "configs.config.edge-orchestrator.intel.com"},
		"runtimefolders.runtimefolder.edge-orchestrator.intel.com":               {"multitenancies.tenancy.edge-orchestrator.intel.com", "runtimes.runtime.edge-orchestrator.intel.com", "runtimeorgs.runtimeorg.edge-orchestrator.intel.com"},
		"runtimeorgs.runtimeorg.edge-orchestrator.intel.com":                     {"multitenancies.tenancy.edge-orchestrator.intel.com", "runtimes.runtime.edge-orchestrator.intel.com"},
		"runtimeprojects.runtimeproject.edge-orchestrator.intel.com":             {"multitenancies.tenancy.edge-orchestrator.intel.com", "runtimes.runtime.edge-orchestrator.intel.com", "runtimeorgs.runtimeorg.edge-orchestrator.intel.com", "runtimefolders.runtimefolder.edge-orchestrator.intel.com"},
		"runtimes.runtime.edge-orchestrator.intel.com":                           {"multitenancies.tenancy.edge-orchestrator.intel.com"},
	}
}

//nolint:gocyclo,funlen,cyclop // Generated code. Length depends on actual graph depth.
func GetObjectByCRDName(dmClient *datamodel.Clientset, crdName, name string) interface{} {
	if crdName == "apimappingconfigs.apimappingconfig.edge-orchestrator.intel.com" {
		obj, err := dmClient.ApimappingconfigEdgeV1().APIMappingConfigs().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "configs.config.edge-orchestrator.intel.com" {
		obj, err := dmClient.ConfigEdgeV1().Configs().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "folders.folder.edge-orchestrator.intel.com" {
		obj, err := dmClient.FolderEdgeV1().Folders().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "multitenancies.tenancy.edge-orchestrator.intel.com" {
		obj, err := dmClient.TenancyEdgeV1().MultiTenancies().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "networks.network.edge-orchestrator.intel.com" {
		obj, err := dmClient.NetworkEdgeV1().Networks().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "orgactivewatchers.orgactivewatcher.edge-orchestrator.intel.com" {
		obj, err := dmClient.OrgactivewatcherEdgeV1().OrgActiveWatchers().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "orgs.org.edge-orchestrator.intel.com" {
		obj, err := dmClient.OrgEdgeV1().Orgs().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "orgwatchers.orgwatcher.edge-orchestrator.intel.com" {
		obj, err := dmClient.OrgwatcherEdgeV1().OrgWatchers().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "projectactivewatchers.projectactivewatcher.edge-orchestrator.intel.com" {
		obj, err := dmClient.ProjectactivewatcherEdgeV1().ProjectActiveWatchers().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "projects.project.edge-orchestrator.intel.com" {
		obj, err := dmClient.ProjectEdgeV1().Projects().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "projectwatchers.projectwatcher.edge-orchestrator.intel.com" {
		obj, err := dmClient.ProjectwatcherEdgeV1().ProjectWatchers().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "runtimefolders.runtimefolder.edge-orchestrator.intel.com" {
		obj, err := dmClient.RuntimefolderEdgeV1().RuntimeFolders().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "runtimeorgs.runtimeorg.edge-orchestrator.intel.com" {
		obj, err := dmClient.RuntimeorgEdgeV1().RuntimeOrgs().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "runtimeprojects.runtimeproject.edge-orchestrator.intel.com" {
		obj, err := dmClient.RuntimeprojectEdgeV1().RuntimeProjects().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "runtimes.runtime.edge-orchestrator.intel.com" {
		obj, err := dmClient.RuntimeEdgeV1().Runtimes().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}

	return nil
}

func ParseCRDLabels(crdName string, labels map[string]string) *orderedmap.OrderedMap {
	parents := GetCRDParentsMap()[crdName]

	m := orderedmap.NewOrderedMap()
	for _, parent := range parents {
		if label, ok := labels[parent]; ok {
			m.Set(parent, label)
		} else {
			m.Set(parent, DefaultKey)
		}
	}

	return m
}

func GetHashedName(crdName string, labels map[string]string, name string) string {
	orderedLabels := ParseCRDLabels(crdName, labels)

	var output string
	for i, key := range orderedLabels.Keys() {
		value, _ := orderedLabels.Get(key)

		output += fmt.Sprintf("%s:%s", key, value)
		if i < orderedLabels.Len()-1 {
			output += "/"
		}
	}

	output += fmt.Sprintf("%s:%s", crdName, name)
	//nolint:gosec // only useful fixed strig hashing and not for security.
	h := sha1.New()
	_, _ = h.Write([]byte(output))
	return hex.EncodeToString(h.Sum(nil))
}
