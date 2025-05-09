// Copyright The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	servicegrouptsmtanzuvmwarecomv1 "/build/apis/servicegroup.tsm.tanzu.vmware.com/v1"
	versioned "/build/client/clientset/versioned"
	internalinterfaces "/build/client/informers/externalversions/internalinterfaces"
	v1 "/build/client/listers/servicegroup.tsm.tanzu.vmware.com/v1"
	time "time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// SvcGroupInformer provides access to a shared informer and lister for
// SvcGroups.
type SvcGroupInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.SvcGroupLister
}

type svcGroupInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewSvcGroupInformer constructs a new informer for SvcGroup type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewSvcGroupInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredSvcGroupInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredSvcGroupInformer constructs a new informer for SvcGroup type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredSvcGroupInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServicegroupTsmV1().SvcGroups().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServicegroupTsmV1().SvcGroups().Watch(context.TODO(), options)
			},
		},
		&servicegrouptsmtanzuvmwarecomv1.SvcGroup{},
		resyncPeriod,
		indexers,
	)
}

func (f *svcGroupInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredSvcGroupInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *svcGroupInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&servicegrouptsmtanzuvmwarecomv1.SvcGroup{}, f.defaultInformer)
}

func (f *svcGroupInformer) Lister() v1.SvcGroupLister {
	return v1.NewSvcGroupLister(f.Informer().GetIndexer())
}
