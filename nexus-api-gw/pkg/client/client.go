// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/vmware-tanzu/graph-framework-for-microservices/common-library/pkg/nexus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var (
	Client              dynamic.Interface
	CoreClient          kubernetes.Interface
	Host                string
	HostScheme          = "http"
	HostTLSClientConfig rest.TLSClientConfig
	appName             = "nexus-api-gw-client"
	log                 = logging.GetLogger(appName)
)

var NexusClient *nexus_client.Clientset

func New(config *rest.Config) (err error) {
	if config != nil {
		HostTLSClientConfig = config.TLSClientConfig
	}

	if kubeAPIHostPort, isSpecified := os.LookupEnv("KUBEAPI_ENDPOINT"); isSpecified {
		parsedURI, err := url.Parse(kubeAPIHostPort)
		if err != nil {
			return fmt.Errorf("parsing URI %v failed with error %w", kubeAPIHostPort, err)
		}

		if parsedURI.Scheme != "" {
			HostScheme = parsedURI.Scheme
		}
		Host = fmt.Sprintf("%s://%s", HostScheme, parsedURI.Host)

		log.Debug().Msgf("kubeApiHostPort: %+v", kubeAPIHostPort)
		log.Debug().Msgf("parsedURI: %+v", parsedURI)
		log.Debug().Msgf("Host: %s", Host)
	} else {
		Host = config.Host
	}

	config.Burst = 1500
	config.QPS = 1000
	Client, err = dynamic.NewForConfig(config)
	if err != nil {
		log.InfraErr(err).Msg("error building dynamic client")
		return err
	}
	CoreClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	return nil
}

func NewNexusClient(config *rest.Config) error {
	// Create a datamodel client handle.
	var err error
	NexusClient, err = nexus_client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create nexus client: %w", err)
	}
	return nil
}

func CreateObject(gvr schema.GroupVersionResource, kind, hashedName string, labels map[string]string,
	body map[string]interface{}, finalizers []string,
) error {
	labelsUnstructured := map[string]interface{}{}
	for k, v := range labels {
		labelsUnstructured[k] = v
	}
	finalizersUnstructured := make([]interface{}, len(finalizers))
	for i, f := range finalizers {
		finalizersUnstructured[i] = f
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gvr.GroupVersion().String(),
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":       hashedName,
				"labels":     labelsUnstructured,
				"finalizers": finalizersUnstructured,
			},
			"spec": body,
		},
	}

	// Create resource
	_, err := Client.Resource(gvr).Create(context.TODO(), obj, metav1.CreateOptions{})
	return err
}

func GetObject(gvr schema.GroupVersionResource, hashedName string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	obj, err := Client.Resource(gvr).Get(context.TODO(), hashedName, opts)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func DeleteObject(gvr schema.GroupVersionResource, crdType string, crdInfo model.NodeInfo, hashedName string) error {
	// Get object
	obj, err := Client.Resource(gvr).Get(context.TODO(), hashedName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	labels := obj.GetLabels()

	// Delete all children
	listOpts := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", crdType, labels["nexus/display_name"])}
	for k := range crdInfo.Children {
		err = DeleteChildren(k, listOpts)
		if err != nil {
			return err
		}
	}

	// Delete object
	err = Client.Resource(gvr).Delete(context.TODO(), hashedName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func DeleteChildren(crdType string, listOpts metav1.ListOptions) error {
	crdInfo := model.CrdTypeToNodeInfo[crdType]
	for k := range crdInfo.Children {
		err := DeleteChildren(k, listOpts)
		if err != nil {
			return err
		}
	}

	parts := strings.Split(crdType, ".")
	gvr := schema.GroupVersionResource{
		Group:    strings.Join(parts[1:], "."),
		Version:  "v1",
		Resource: parts[0],
	}
	err := Client.Resource(gvr).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, listOpts)
	if err != nil {
		return err
	}

	return nil
}

type PatchOp struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type Patch []PatchOp

func (p Patch) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func GetParent(parentCrdType string, parentCrdInfo model.NodeInfo, labels map[string]string) (*unstructured.Unstructured, error) {
	parentParts := strings.Split(parentCrdType, ".")
	gvr := schema.GroupVersionResource{
		Group:    strings.Join(parentParts[1:], "."),
		Version:  "v1",
		Resource: parentParts[0],
	}

	parentName := labels[parentCrdType]
	hashedParentName := nexus.GetHashedName(parentCrdType, parentCrdInfo.ParentHierarchy, labels, parentName)
	return GetObject(gvr, hashedParentName, metav1.GetOptions{})
}

// TODO: build PatchOP in common-library.
func UpdateParentWithAddedChild(parentCrdType string, parentCrdInfo model.NodeInfo,
	labels map[string]string, childCrdInfo model.NodeInfo, childCrdType, childName, childHashedName string,
) error {
	var (
		patchType types.PatchType
		marshaled []byte
	)

	parentParts := strings.Split(parentCrdType, ".")
	gvr := schema.GroupVersionResource{
		Group:    strings.Join(parentParts[1:], "."),
		Version:  "v1",
		Resource: parentParts[0],
	}

	parentName := labels[parentCrdType]
	hashedParentName := nexus.GetHashedName(parentCrdType, parentCrdInfo.ParentHierarchy, labels, parentName)
	childGvk := parentCrdInfo.Children[childCrdType]

	childParts := strings.Split(childCrdType, ".")
	group := strings.Join(childParts[1:], ".")
	childNameParts := strings.Split(childCrdInfo.Name, ".")

	if childGvk.IsNamed {
		payload := fmt.Sprintf(`{
			"spec": {
				"%s": {
					"%s": {
						"name": "%s",
						"kind": "%s",
						"group": "%s"
					}
				}
			}
		}`, childGvk.FieldNameGvk, childName, childHashedName, childNameParts[1], group)

		patchType = types.MergePatchType
		marshaled = []byte(payload)
	} else {
		var patch Patch
		patchOp := PatchOp{
			Op:   "replace",
			Path: "/spec/" + childGvk.FieldNameGvk,
			Value: map[string]interface{}{
				"name":  childHashedName,
				"group": group,
				"kind":  childNameParts[1],
			},
		}
		patch = append(patch, patchOp)
		patchBytes, err := patch.Marshal()
		if err != nil {
			return err
		}
		marshaled = patchBytes
		patchType = types.JSONPatchType
	}

	_, err := Client.Resource(gvr).Patch(context.TODO(),
		hashedParentName, patchType, marshaled, metav1.PatchOptions{},
	)
	if err != nil {
		log.InfraErr(err).Msgf("UpdateParentWithAddedChild: failed to patch %v", hashedParentName)
		return err
	}

	return nil
}

func UpdateParentWithRemovedChild(parentCrdType string, parentCrdInfo model.NodeInfo,
	labels map[string]string, childCrdType, childName string,
) error {
	parentParts := strings.Split(parentCrdType, ".")
	gvr := schema.GroupVersionResource{
		Group:    strings.Join(parentParts[1:], "."),
		Version:  "v1",
		Resource: parentParts[0],
	}

	parentName := labels[parentCrdType]
	hashedParentName := nexus.GetHashedName(parentCrdType, parentCrdInfo.ParentHierarchy, labels, parentName)
	childGvk := parentCrdInfo.Children[childCrdType]

	var patchOp PatchOp
	if childGvk.IsNamed {
		patchOp = PatchOp{
			Op:   "remove",
			Path: "/spec/" + childGvk.FieldNameGvk + "/" + childName,
		}
	} else {
		patchOp = PatchOp{
			Op:   "remove",
			Path: "/spec/" + childGvk.FieldNameGvk,
		}
	}

	var patch Patch
	patch = append(patch, patchOp)

	marshaled, err := patch.Marshal()
	if err != nil {
		return err
	}

	_, err = Client.Resource(gvr).Patch(context.TODO(), hashedParentName, types.JSONPatchType, marshaled, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
