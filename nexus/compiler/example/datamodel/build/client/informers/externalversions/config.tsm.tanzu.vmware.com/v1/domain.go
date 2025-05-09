// Copyright The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	configtsmtanzuvmwarecomv1 "/build/apis/config.tsm.tanzu.vmware.com/v1"
	versioned "/build/client/clientset/versioned"
	internalinterfaces "/build/client/informers/externalversions/internalinterfaces"
	v1 "/build/client/listers/config.tsm.tanzu.vmware.com/v1"
	time "time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// DomainInformer provides access to a shared informer and lister for
// Domains.
type DomainInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.DomainLister
}

type domainInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewDomainInformer constructs a new informer for Domain type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewDomainInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredDomainInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredDomainInformer constructs a new informer for Domain type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredDomainInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigTsmV1().Domains().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigTsmV1().Domains().Watch(context.TODO(), options)
			},
		},
		&configtsmtanzuvmwarecomv1.Domain{},
		resyncPeriod,
		indexers,
	)
}

func (f *domainInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredDomainInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *domainInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&configtsmtanzuvmwarecomv1.Domain{}, f.defaultInformer)
}

func (f *domainInformer) Lister() v1.DomainLister {
	return v1.NewDomainLister(f.Informer().GetIndexer())
}
