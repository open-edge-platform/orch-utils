// Copyright The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "/build/apis/optionalparentpathparam.tsm.tanzu.vmware.com/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// OptionalParentPathParamLister helps list OptionalParentPathParams.
// All objects returned here must be treated as read-only.
type OptionalParentPathParamLister interface {
	// List lists all OptionalParentPathParams in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.OptionalParentPathParam, err error)
	// Get retrieves the OptionalParentPathParam from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.OptionalParentPathParam, error)
	OptionalParentPathParamListerExpansion
}

// optionalParentPathParamLister implements the OptionalParentPathParamLister interface.
type optionalParentPathParamLister struct {
	indexer cache.Indexer
}

// NewOptionalParentPathParamLister returns a new OptionalParentPathParamLister.
func NewOptionalParentPathParamLister(indexer cache.Indexer) OptionalParentPathParamLister {
	return &optionalParentPathParamLister{indexer: indexer}
}

// List lists all OptionalParentPathParams in the indexer.
func (s *optionalParentPathParamLister) List(selector labels.Selector) (ret []*v1.OptionalParentPathParam, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.OptionalParentPathParam))
	})
	return ret, err
}

// Get retrieves the OptionalParentPathParam from the index for a given name.
func (s *optionalParentPathParamLister) Get(name string) (*v1.OptionalParentPathParam, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("optionalparentpathparam"), name)
	}
	return obj.(*v1.OptionalParentPathParam), nil
}
