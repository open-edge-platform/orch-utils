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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	gnstsmtanzuvmwarecomv1 "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/apis/gns.tsm.tanzu.vmware.com/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGnses implements GnsInterface
type FakeGnses struct {
	Fake *FakeGnsTsmV1
}

var gnsesResource = schema.GroupVersionResource{Group: "gns.tsm.tanzu.vmware.com", Version: "v1", Resource: "gnses"}

var gnsesKind = schema.GroupVersionKind{Group: "gns.tsm.tanzu.vmware.com", Version: "v1", Kind: "Gns"}

// Get takes name of the gns, and returns the corresponding gns object, and an error if there is any.
func (c *FakeGnses) Get(ctx context.Context, name string, options v1.GetOptions) (result *gnstsmtanzuvmwarecomv1.Gns, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(gnsesResource, name), &gnstsmtanzuvmwarecomv1.Gns{})
	if obj == nil {
		return nil, err
	}
	return obj.(*gnstsmtanzuvmwarecomv1.Gns), err
}

// List takes label and field selectors, and returns the list of Gnses that match those selectors.
func (c *FakeGnses) List(ctx context.Context, opts v1.ListOptions) (result *gnstsmtanzuvmwarecomv1.GnsList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(gnsesResource, gnsesKind, opts), &gnstsmtanzuvmwarecomv1.GnsList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &gnstsmtanzuvmwarecomv1.GnsList{ListMeta: obj.(*gnstsmtanzuvmwarecomv1.GnsList).ListMeta}
	for _, item := range obj.(*gnstsmtanzuvmwarecomv1.GnsList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested gnses.
func (c *FakeGnses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(gnsesResource, opts))
}

// Create takes the representation of a gns and creates it.  Returns the server's representation of the gns, and an error, if there is any.
func (c *FakeGnses) Create(ctx context.Context, gns *gnstsmtanzuvmwarecomv1.Gns, opts v1.CreateOptions) (result *gnstsmtanzuvmwarecomv1.Gns, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(gnsesResource, gns), &gnstsmtanzuvmwarecomv1.Gns{})
	if obj == nil {
		return nil, err
	}
	return obj.(*gnstsmtanzuvmwarecomv1.Gns), err
}

// Update takes the representation of a gns and updates it. Returns the server's representation of the gns, and an error, if there is any.
func (c *FakeGnses) Update(ctx context.Context, gns *gnstsmtanzuvmwarecomv1.Gns, opts v1.UpdateOptions) (result *gnstsmtanzuvmwarecomv1.Gns, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(gnsesResource, gns), &gnstsmtanzuvmwarecomv1.Gns{})
	if obj == nil {
		return nil, err
	}
	return obj.(*gnstsmtanzuvmwarecomv1.Gns), err
}

// Delete takes name of the gns and deletes it. Returns an error if one occurs.
func (c *FakeGnses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(gnsesResource, name, opts), &gnstsmtanzuvmwarecomv1.Gns{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGnses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(gnsesResource, listOpts)

	_, err := c.Fake.Invokes(action, &gnstsmtanzuvmwarecomv1.GnsList{})
	return err
}

// Patch applies the patch and returns the patched gns.
func (c *FakeGnses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *gnstsmtanzuvmwarecomv1.Gns, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(gnsesResource, name, pt, data, subresources...), &gnstsmtanzuvmwarecomv1.Gns{})
	if obj == nil {
		return nil, err
	}
	return obj.(*gnstsmtanzuvmwarecomv1.Gns), err
}
