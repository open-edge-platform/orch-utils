// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"
	v1 "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/apis/config.tsm.tanzu.vmware.com/v1"
	gnstsmtanzuvmwarecomv1 "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/apis/gns.tsm.tanzu.vmware.com/v1"
	optionalparentpathparamtsmtanzuvmwarecomv1 "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/apis/optionalparentpathparam.tsm.tanzu.vmware.com/v1"
	policypkgtsmtanzuvmwarecomv1 "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/apis/policypkg.tsm.tanzu.vmware.com/v1"
	roottsmtanzuvmwarecomv1 "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/apis/root.tsm.tanzu.vmware.com/v1"
	servicegrouptsmtanzuvmwarecomv1 "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/apis/servicegroup.tsm.tanzu.vmware.com/v1"

	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=config.tsm.tanzu.vmware.com, Version=v1
	case v1.SchemeGroupVersion.WithResource("configs"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ConfigTsm().V1().Configs().Informer()}, nil
	case v1.SchemeGroupVersion.WithResource("domains"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ConfigTsm().V1().Domains().Informer()}, nil
	case v1.SchemeGroupVersion.WithResource("footypeabcs"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ConfigTsm().V1().FooTypeABCs().Informer()}, nil

		// Group=gns.tsm.tanzu.vmware.com, Version=v1
	case gnstsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("barchilds"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.GnsTsm().V1().BarChilds().Informer()}, nil
	case gnstsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("dnses"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.GnsTsm().V1().Dnses().Informer()}, nil
	case gnstsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("foos"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.GnsTsm().V1().Foos().Informer()}, nil
	case gnstsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("gnses"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.GnsTsm().V1().Gnses().Informer()}, nil
	case gnstsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("ignorechilds"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.GnsTsm().V1().IgnoreChilds().Informer()}, nil

		// Group=optionalparentpathparam.tsm.tanzu.vmware.com, Version=v1
	case optionalparentpathparamtsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("optionalparentpathparams"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.OptionalparentpathparamTsm().V1().OptionalParentPathParams().Informer()}, nil

		// Group=policypkg.tsm.tanzu.vmware.com, Version=v1
	case policypkgtsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("acpconfigs"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.PolicypkgTsm().V1().ACPConfigs().Informer()}, nil
	case policypkgtsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("accesscontrolpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.PolicypkgTsm().V1().AccessControlPolicies().Informer()}, nil
	case policypkgtsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("vmpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.PolicypkgTsm().V1().VMpolicies().Informer()}, nil

		// Group=root.tsm.tanzu.vmware.com, Version=v1
	case roottsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("roots"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.RootTsm().V1().Roots().Informer()}, nil

		// Group=servicegroup.tsm.tanzu.vmware.com, Version=v1
	case servicegrouptsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("svcgroups"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ServicegroupTsm().V1().SvcGroups().Informer()}, nil
	case servicegrouptsmtanzuvmwarecomv1.SchemeGroupVersion.WithResource("svcgrouplinkinfos"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ServicegroupTsm().V1().SvcGroupLinkInfos().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
