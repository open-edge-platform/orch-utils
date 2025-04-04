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

	datamodel "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultKey = "default"
const DisplayNameLabel = "nexus/display_name"
const IsNameHashedLabel = "nexus/is_name_hashed"

//nolint:lll // Generated code. Length depends on actual graph depth.
func GetCRDParentsMap() map[string][]string {
	return map[string][]string{
		"accesscontrolpolicies.policypkg.tsm.tanzu.vmware.com":                  {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com", "gnses.gns.tsm.tanzu.vmware.com"},
		"acpconfigs.policypkg.tsm.tanzu.vmware.com":                             {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com", "gnses.gns.tsm.tanzu.vmware.com", "accesscontrolpolicies.policypkg.tsm.tanzu.vmware.com"},
		"barchilds.gns.tsm.tanzu.vmware.com":                                    {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com", "gnses.gns.tsm.tanzu.vmware.com"},
		"configs.config.tsm.tanzu.vmware.com":                                   {"roots.root.tsm.tanzu.vmware.com"},
		"dnses.gns.tsm.tanzu.vmware.com":                                        {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com"},
		"domains.config.tsm.tanzu.vmware.com":                                   {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com"},
		"foos.gns.tsm.tanzu.vmware.com":                                         {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com", "gnses.gns.tsm.tanzu.vmware.com"},
		"footypeabcs.config.tsm.tanzu.vmware.com":                               {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com"},
		"gnses.gns.tsm.tanzu.vmware.com":                                        {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com"},
		"ignorechilds.gns.tsm.tanzu.vmware.com":                                 {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com", "gnses.gns.tsm.tanzu.vmware.com"},
		"optionalparentpathparams.optionalparentpathparam.tsm.tanzu.vmware.com": {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com", "gnses.gns.tsm.tanzu.vmware.com"},
		"roots.root.tsm.tanzu.vmware.com":                                       {},
		"svcgrouplinkinfos.servicegroup.tsm.tanzu.vmware.com":                   {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com"},
		"svcgroups.servicegroup.tsm.tanzu.vmware.com":                           {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com", "gnses.gns.tsm.tanzu.vmware.com"},
		"vmpolicies.policypkg.tsm.tanzu.vmware.com":                             {"roots.root.tsm.tanzu.vmware.com", "configs.config.tsm.tanzu.vmware.com"},
	}
}

//nolint:gocyclo,funlen,cyclop // Generated code. Length depends on actual graph depth.
func GetObjectByCRDName(dmClient *datamodel.Clientset, crdName, name string) interface{} {
	if crdName == "accesscontrolpolicies.policypkg.tsm.tanzu.vmware.com" {
		obj, err := dmClient.PolicypkgTsmV1().AccessControlPolicies().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "acpconfigs.policypkg.tsm.tanzu.vmware.com" {
		obj, err := dmClient.PolicypkgTsmV1().ACPConfigs().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "barchilds.gns.tsm.tanzu.vmware.com" {
		obj, err := dmClient.GnsTsmV1().BarChilds().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "configs.config.tsm.tanzu.vmware.com" {
		obj, err := dmClient.ConfigTsmV1().Configs().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "dnses.gns.tsm.tanzu.vmware.com" {
		obj, err := dmClient.GnsTsmV1().Dnses().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "domains.config.tsm.tanzu.vmware.com" {
		obj, err := dmClient.ConfigTsmV1().Domains().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "foos.gns.tsm.tanzu.vmware.com" {
		obj, err := dmClient.GnsTsmV1().Foos().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "footypeabcs.config.tsm.tanzu.vmware.com" {
		obj, err := dmClient.ConfigTsmV1().FooTypeABCs().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "gnses.gns.tsm.tanzu.vmware.com" {
		obj, err := dmClient.GnsTsmV1().Gnses().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "ignorechilds.gns.tsm.tanzu.vmware.com" {
		obj, err := dmClient.GnsTsmV1().IgnoreChilds().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "optionalparentpathparams.optionalparentpathparam.tsm.tanzu.vmware.com" {
		obj, err := dmClient.OptionalparentpathparamTsmV1().OptionalParentPathParams().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "roots.root.tsm.tanzu.vmware.com" {
		obj, err := dmClient.RootTsmV1().Roots().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "svcgrouplinkinfos.servicegroup.tsm.tanzu.vmware.com" {
		obj, err := dmClient.ServicegroupTsmV1().SvcGroupLinkInfos().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "svcgroups.servicegroup.tsm.tanzu.vmware.com" {
		obj, err := dmClient.ServicegroupTsmV1().SvcGroups().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil
		}
		return obj
	}
	if crdName == "vmpolicies.policypkg.tsm.tanzu.vmware.com" {
		obj, err := dmClient.PolicypkgTsmV1().VMpolicies().Get(context.TODO(), name, metav1.GetOptions{})
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
