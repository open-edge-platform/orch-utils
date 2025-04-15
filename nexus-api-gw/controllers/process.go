// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"net/http"

	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/model"
	"github.com/vmware-tanzu/graph-framework-for-microservices/nexus/nexus"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/json"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var (
	appName = "nexus-api-gw-controllers"
	log     = logging.GetLogger(appName)
)

func (r *CustomResourceDefinitionReconciler) ProcessAnnotation(crdType string,
	annotations map[string]string, eventType model.EventType,
) error {
	n := model.NexusAnnotation{}

	if eventType != model.Delete {
		apiInfo, ok := annotations["nexus"]
		if !ok {
			return nil
		}

		// unmarshall to nexus annotation struct
		err := json.Unmarshal([]byte(apiInfo), &n)
		if err != nil {
			log.InfraErr(err).Msg("Error unmarshaling Nexus annotation")
			return err
		}
	}

	children := make(map[string]model.NodeHelperChild)
	if n.Children != nil {
		children = n.Children
	}

	links := make(map[string]model.NodeHelperChild)
	if n.Links != nil {
		links = n.Links
	}

	urisMap := make(map[string]model.RestURIInfo)

	// add child, link and status URIs for each GET method
	var newUris []nexus.RestURIs
	ConstructNewURIs(n, urisMap, &newUris)

	log.Debug().Msgf("New uris %v\n", newUris)

	n.NexusRestAPIGen.Uris = append(n.NexusRestAPIGen.Uris, newUris...)

	// It has stored the URI with the CRD type and CRD type with the Node Info.
	model.ConstructMapURIToURIInfo(eventType, urisMap)
	model.ConstructMapURIToCRDType(eventType, crdType, n.NexusRestAPIGen.Uris)
	model.ConstructMapCRDTypeToNode(eventType, crdType, n.Name, n.Hierarchy,
		children, links, n.IsSingleton, n.Description, n.DeferredDelete,
	)
	model.ConstructMapCRDTypeToRestUris(eventType, crdType, n.NexusRestAPIGen)

	// Restart echo server
	log.Debug().Msg("Restarting echo server...")
	r.StopCh <- struct{}{}

	for cType, uris := range model.CrdTypeToRestUris {
		model.RestURIChan <- uris
		model.CrdTypeChan <- cType
	}
	return nil
}

func (r *CustomResourceDefinitionReconciler) ProcessCrdSpec(crdType string,
	spec apiextensionsv1.CustomResourceDefinitionSpec, eventType model.EventType,
) error {
	// It has stored the CRD type with the CRD spec
	model.ConstructMapCRDTypeToSpec(eventType, crdType, spec)
	return nil
}

// ConstructNewURIs constructs the new URIs from ['status', 'children', 'links'] and store it in cache.
func ConstructNewURIs(n model.NexusAnnotation, urisMap map[string]model.RestURIInfo, newUris *[]nexus.RestURIs) {
	for _, uri := range n.NexusRestAPIGen.Uris {
		urisMap[uri.Uri] = model.RestURIInfo{
			TypeOfURI: model.DefaultURI,
		}
		for method := range uri.Methods {
			if method == http.MethodGet {
				statusURIPath := uri.Uri + "/status"
				addStatusURI(statusURIPath, model.StatusURI, urisMap, newUris)

				for _, c := range []map[string]model.NodeHelperChild{n.Children, n.Links} {
					processChildOrLink(c, uri, urisMap, newUris)
				}
			}
		}
	}
}

func processChildOrLink(nodes map[string]model.NodeHelperChild, uri nexus.RestURIs,
	urisMap map[string]model.RestURIInfo, newUris *[]nexus.RestURIs,
) {
	for _, n := range nodes {
		uriPath := uri.Uri + "/" + n.FieldName
		var t model.URIType
		if n.IsNamed {
			t = model.NamedLinkURI
		} else {
			t = model.SingleLinkURI
		}
		addURI(uriPath, t, urisMap, newUris)
	}
}

// addURI adds the uriPath </root/{orgchart.Root}/leader/{management.Leader}/HR> to the urisMap and to the uris list.
func addURI(uriPath string, typeOfURI model.URIType, urisMap map[string]model.RestURIInfo, uris *[]nexus.RestURIs) {
	newURI := nexus.RestURIs{
		Uri: uriPath,
		Methods: map[nexus.HTTPMethod]nexus.HTTPCodesResponse{
			http.MethodGet: nexus.DefaultHTTPGETResponses,
		},
	}
	urisMap[uriPath] = model.RestURIInfo{
		TypeOfURI: typeOfURI,
	}
	*uris = append(*uris, newURI)
}

func addStatusURI(uriPath string, typeOfURI model.URIType, urisMap map[string]model.RestURIInfo, uris *[]nexus.RestURIs) {
	newURI := nexus.RestURIs{
		Uri: uriPath,
		Methods: map[nexus.HTTPMethod]nexus.HTTPCodesResponse{
			http.MethodGet: nexus.DefaultHTTPGETResponses,
			http.MethodPut: nexus.DefaultHTTPPUTResponses,
		},
	}
	urisMap[uriPath] = model.RestURIInfo{
		TypeOfURI: typeOfURI,
	}
	*uris = append(*uris, newURI)
}
