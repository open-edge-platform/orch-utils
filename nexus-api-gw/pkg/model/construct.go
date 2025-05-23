// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"
	"sync"

	"github.com/vmware-tanzu/graph-framework-for-microservices/nexus/nexus"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const constDefaultChanSize = 100

var (
	RestURIChan = make(chan []nexus.RestURIs, constDefaultChanSize)
	CrdTypeChan = make(chan string, constDefaultChanSize)

	CrdTypeToRestUris      = make(map[string][]nexus.RestURIs)
	crdTypeToRestUrisMutex = &sync.Mutex{}

	// CRD name to CRD type (Gns.gns => gns.vmware.org).
	URIToCRDType      = make(map[string]string)
	uriToCRDTypeMutex = &sync.Mutex{}

	// URI to info about this URI.
	URIToURIInfo      = make(map[string]RestURIInfo)
	URIToURIInfoMutex = &sync.Mutex{}

	// CRD Type to NodeInfo (gns.vmware.org => NodeInfo{}).
	CrdTypeToNodeInfo      = make(map[string]NodeInfo)
	crdTypeToNodeInfoMutex = &sync.Mutex{}

	// CRD Type to k8s spec (gns.vmware.org => CustomResourceDefinitionSpec).
	CrdTypeToSpec      = make(map[string]apiextensionsv1.CustomResourceDefinitionSpec)
	crdTypeToSpecMutex = &sync.Mutex{}

	DatamodelsChan                = make(chan string, constDefaultChanSize)
	DatamodelToDatamodelInfo      = make(map[string]DatamodelInfo)
	DatamodelToDatamodelInfoMutex = &sync.Mutex{}
)

func ConstructDatamodel(eventType EventType, name string, unstructuredObj *unstructured.Unstructured) {
	DatamodelToDatamodelInfoMutex.Lock()
	defer DatamodelToDatamodelInfoMutex.Unlock()

	if eventType == Delete {
		delete(DatamodelToDatamodelInfo, name)
		return
	}
	obj := unstructuredObj.Object

	spec, ok := obj["spec"].(map[string]interface{})
	if !ok {
		fmt.Println("obj[spec] is not of type (map[string]interface{})")
		return
	}
	if title, ok := spec["title"]; ok {
		titleInString, ok := title.(string)
		if !ok {
			fmt.Println("title is not of type string")
			return
		}
		datamodelName := name
		DatamodelToDatamodelInfo[datamodelName] = DatamodelInfo{
			Title: titleInString,
		}

		DatamodelsChan <- datamodelName
	}
}

func ConstructMapURIToCRDType(eventType EventType, crdType string, apiURIs []nexus.RestURIs) {
	uriToCRDTypeMutex.Lock()
	defer uriToCRDTypeMutex.Unlock()

	if eventType == Delete {
		for uri, cType := range URIToCRDType {
			if cType == crdType {
				delete(URIToCRDType, uri)
			}
		}
	}

	for _, u := range apiURIs {
		URIToCRDType[u.Uri] = crdType
	}
}

func ConstructMapCRDTypeToNode(eventType EventType, crdType, name string, parentHierarchy []string,
	children, links map[string]NodeHelperChild, isSingleton bool, description string, deferredDelete bool,
) {
	crdTypeToNodeInfoMutex.Lock()
	defer crdTypeToNodeInfoMutex.Unlock()

	if eventType == Delete {
		delete(CrdTypeToNodeInfo, crdType)
	}

	CrdTypeToNodeInfo[crdType] = NodeInfo{
		Name:            name,
		ParentHierarchy: parentHierarchy,
		Children:        children,
		Links:           links,
		IsSingleton:     isSingleton,
		Description:     description,
		DeferredDelete:  deferredDelete,
	}

	// Push new CRD Type to chan.
	CrdTypeChan <- crdType
}

func GetCRDTypeToNodeInfo(crdType string) (NodeInfo, bool) {
	crdTypeToNodeInfoMutex.Lock()
	defer crdTypeToNodeInfoMutex.Unlock()

	info, ok := CrdTypeToNodeInfo[crdType]
	return info, ok
}

func GetDatamodel(name string) (DatamodelInfo, bool) {
	DatamodelToDatamodelInfoMutex.Lock()
	defer DatamodelToDatamodelInfoMutex.Unlock()

	info, ok := DatamodelToDatamodelInfo[name]
	return info, ok
}

func ConstructMapCRDTypeToSpec(eventType EventType, crdType string, spec apiextensionsv1.CustomResourceDefinitionSpec) {
	crdTypeToSpecMutex.Lock()
	defer crdTypeToSpecMutex.Unlock()

	if eventType == Delete {
		delete(CrdTypeToSpec, crdType)
	}
	CrdTypeToSpec[crdType] = spec
}

func GetRestUris(crdType string) ([]nexus.RestURIs, bool) {
	crdTypeToRestUrisMutex.Lock()
	defer crdTypeToRestUrisMutex.Unlock()

	uris, ok := CrdTypeToRestUris[crdType]
	return uris, ok
}

func ConstructMapCRDTypeToRestUris(eventType EventType, crdType string, restSpec nexus.RestAPISpec) {
	crdTypeToRestUrisMutex.Lock()
	defer crdTypeToRestUrisMutex.Unlock()

	if eventType == Delete {
		delete(CrdTypeToRestUris, crdType)
		return
	}

	CrdTypeToRestUris[crdType] = restSpec.Uris

	// Push new uris to chan.
	RestURIChan <- restSpec.Uris
}

func ConstructMapURIToURIInfo(eventType EventType, m map[string]RestURIInfo) {
	URIToURIInfoMutex.Lock()
	defer URIToURIInfoMutex.Unlock()

	if eventType == Delete {
		for k := range m {
			delete(URIToURIInfo, k)
		}
	}
	for k, v := range m {
		URIToURIInfo[k] = v
	}
}

func GetURIInfo(uriPath string) (RestURIInfo, bool) {
	URIToURIInfoMutex.Lock()
	defer URIToURIInfoMutex.Unlock()
	info, ok := URIToURIInfo[uriPath]
	return info, ok
}
