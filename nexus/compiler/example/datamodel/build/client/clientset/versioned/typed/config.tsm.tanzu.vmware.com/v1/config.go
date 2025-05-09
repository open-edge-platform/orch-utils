// Copyright The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	v1 "/build/apis/config.tsm.tanzu.vmware.com/v1"
	scheme "/build/client/clientset/versioned/scheme"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ConfigsGetter has a method to return a ConfigInterface.
// A group's client should implement this interface.
type ConfigsGetter interface {
	Configs() ConfigInterface
}

// ConfigInterface has methods to work with Config resources.
type ConfigInterface interface {
	Create(ctx context.Context, config *v1.Config, opts metav1.CreateOptions) (*v1.Config, error)
	Update(ctx context.Context, config *v1.Config, opts metav1.UpdateOptions) (*v1.Config, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Config, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ConfigList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Config, err error)
	ConfigExpansion
}

// configs implements ConfigInterface
type configs struct {
	client rest.Interface
}

// newConfigs returns a Configs
func newConfigs(c *ConfigTsmV1Client) *configs {
	return &configs{
		client: c.RESTClient(),
	}
}

// Get takes name of the config, and returns the corresponding config object, and an error if there is any.
func (c *configs) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Config, err error) {
	result = &v1.Config{}
	err = c.client.Get().
		Resource("configs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Configs that match those selectors.
func (c *configs) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ConfigList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ConfigList{}
	err = c.client.Get().
		Resource("configs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested configs.
func (c *configs) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("configs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a config and creates it.  Returns the server's representation of the config, and an error, if there is any.
func (c *configs) Create(ctx context.Context, config *v1.Config, opts metav1.CreateOptions) (result *v1.Config, err error) {
	result = &v1.Config{}
	err = c.client.Post().
		Resource("configs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(config).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a config and updates it. Returns the server's representation of the config, and an error, if there is any.
func (c *configs) Update(ctx context.Context, config *v1.Config, opts metav1.UpdateOptions) (result *v1.Config, err error) {
	result = &v1.Config{}
	err = c.client.Put().
		Resource("configs").
		Name(config.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(config).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the config and deletes it. Returns an error if one occurs.
func (c *configs) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("configs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *configs) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("configs").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched config.
func (c *configs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Config, err error) {
	result = &v1.Config{}
	err = c.client.Patch(pt).
		Resource("configs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
