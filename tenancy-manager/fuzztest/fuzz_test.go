/*
 * Copyright (C) 2025 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package fuzztest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	configv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/config.edge-orchestrator.intel.com/v1"
	orgv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/org.edge-orchestrator.intel.com/v1"
	projectv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/project.edge-orchestrator.intel.com/v1"
	tenancyv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/tenancy.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/tenancy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

const (
	defaultName = "default"
	finalizer   = "nexus.com/nexus-deferred-delete"
)

var (
	configClient      *nexus_client.ConfigConfig
	tenancyReconciler *tenancy.Reconciler
)

func constructOrgObj(name string) *orgv1.Org {
	return &orgv1.Org{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			ResourceVersion:   "1",
			Finalizers:        []string{finalizer},
			DeletionTimestamp: &metav1.Time{Time: time.Now().UTC()},
		},
	}
}

func constructOrgGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "org.edge-orchestrator.intel.com",
		Version:  "v1",
		Resource: "orgs",
	}
}

func constructProjectGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "project.edge-orchestrator.intel.com",
		Version:  "v1",
		Resource: "projects",
	}
}

func constructUnstructuredOrg(hashedName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "org.edge-orchestrator.intel.com/v1",
			"kind":       "orgs",
			"metadata": map[string]interface{}{
				"name":            hashedName,
				"resourceVersion": "1",
			},
		},
	}
}

func constructUnstructuredProject(hashedName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "project.edge-orchestrator.intel.com/v1",
			"kind":       "projects",
			"metadata": map[string]interface{}{
				"name":            hashedName,
				"resourceVersion": "1",
			},
		},
	}
}

func constructProjectObj(name string) *projectv1.Project {
	return &projectv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			ResourceVersion:   "1",
			Finalizers:        []string{finalizer},
			DeletionTimestamp: &metav1.Time{Time: time.Now().UTC()},
		},
	}
}

func FuzzTenancyOrgProjectCreate(f *testing.F) {
	f.Add("foo", "bar")
	f.Add("f@0", "b@r")
	f.Add("F00", "B4R")

	nexusClient := nexus_client.NewFakeClient()
	nexusClient.DynamicClient = fake.NewSimpleDynamicClient(runtime.NewScheme())

	tenancy.Testing = true
	tenancyReconciler = tenancy.NewReconciler(nexusClient, nil)

	tenancyClient, err := nexusClient.AddTenancyMultiTenancy(context.Background(), &tenancyv1.MultiTenancy{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultName,
		},
	})
	if err != nil {
		f.Fatalf("unexpected error: %v", err)
	}
	configClient, err = tenancyClient.AddConfig(context.Background(), &configv1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultName,
		},
	})
	if err != nil {
		f.Fatalf("unexpected error: %v", err)
	}

	config := nexusClient.TenancyMultiTenancy().Config()
	_, err = config.Orgs("*").RegisterAddCallback(tenancyReconciler.ProcessOrgsAdd)
	if err != nil {
		f.Fatalf("unexpected error: %v", err)
	}
	_, err = config.Orgs("*").Folders("*").Projects("*").
		RegisterAddCallback(tenancyReconciler.ProcessProjectsAdd)
	if err != nil {
		f.Fatalf("unexpected error: %v", err)
	}

	f.Fuzz(func(t *testing.T, org, project string) {
		createOrg(t, nexusClient, org)
		createProject(t, nexusClient, org, project)
	})
}

func createOrg(t *testing.T, nexusClient *nexus_client.Clientset, org string) {
	t.Helper()
	fmt.Printf("Creating org: %v\n", org)
	configOrg, err := configClient.AddOrgs(context.Background(), constructOrgObj(org))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fmt.Printf("Name: %v\n", configOrg.GetName())

	_, err = nexusClient.DynamicClient.Resource(constructOrgGVR()).
		Create(context.Background(), constructUnstructuredOrg(configOrg.GetName()), metav1.CreateOptions{})
	if err != nil && !nexus_client.IsAlreadyExists(err) {
		t.Fatalf("unexpected error: %v", err)
	}

	err = waitUntilOrgCreation(tenancyReconciler, org)
	if err != nil {
		t.Fatalf("org creation didn't complete")
	}
	fmt.Println("Finished creating org")
}

func createProject(t *testing.T, nexusClient *nexus_client.Clientset, org, project string) {
	t.Helper()
	fmt.Printf("Creating project: %v\n", project)
	configProject, err := tenancyReconciler.Client.TenancyMultiTenancy().Config().Orgs(org).
		Folders(defaultName).AddProjects(context.Background(), constructProjectObj(project))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fmt.Printf("Name: %v\n", configProject.GetName())

	_, err = nexusClient.DynamicClient.Resource(constructProjectGVR()).
		Create(context.Background(), constructUnstructuredProject(configProject.GetName()), metav1.CreateOptions{})
	if err != nil && !nexus_client.IsAlreadyExists(err) {
		t.Fatalf("unexpected error: %v", err)
	}

	err = waitUntilProjectCreation(tenancyReconciler, org, project)
	if err != nil {
		t.Fatalf("project creation didn't complete")
	}
	fmt.Println("Finished creating project")
}

func waitUntilOrgCreation(tenancyReconciler *tenancy.Reconciler, org string) error {
	fmt.Println("Waiting until org creation is completed")
	timeout := time.After(300 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Println("Getting runtime org")
			obj, err := tenancyReconciler.Client.TenancyMultiTenancy().Runtime().GetOrgs(context.Background(), org)
			if err == nil {
				fmt.Printf("Runtime org fetched successfully: %v", obj.DisplayName())
				return nil
			}
		case <-timeout:
			return fmt.Errorf("org %s creation timed out", org)
		}
	}
}

func waitUntilProjectCreation(tenancyReconciler *tenancy.Reconciler, org, project string) error {
	fmt.Println("Waiting until project creation is completed")
	timeout := time.After(300 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Println("Getting runtime project")
			obj, err := tenancyReconciler.Client.TenancyMultiTenancy().Runtime().Orgs(org).Folders(defaultName).
				GetProjects(context.Background(), project)
			if err == nil {
				fmt.Printf("Runtime project fetched successfully: %v", obj.DisplayName())
				return nil
			}
			fmt.Println("error getting project", err.Error())
		case <-timeout:
			return fmt.Errorf("project %s creation timed out", project)
		}
	}
}
