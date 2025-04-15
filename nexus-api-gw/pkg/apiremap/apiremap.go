// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package apiremap

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/cache"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/common"
	amcV1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/apimappingconfig.edge-orchestrator.intel.com/v1"
	tenancy_nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

// Define the structs.
type BackendInfo struct {
	SvcName string `json:"svcName"`
	PortStr uint32 `json:"portStr"`
}

type Output struct {
	ServiceURI  string            `json:"serviceUri"`
	Backendinfo BackendInfo       `json:"backendinfo"`
	PathParams  map[string]string `json:"pathParams"`
}

// Define the struct.
type Input struct {
	RequestURI string      `json:"requestUri"`
	Headers    http.Header `json:"headers"`
}

var (
	appName = "nexus-api-gw-apiremap"
	log     = logging.GetLogger(appName)
)

func TenancyAPIRemapping(input Input) (Output, error) {
	log.Debug().Msg("Invoking the TenancyAPIRemapping\n")

	reqArr := strings.Split(input.RequestURI, "?")
	if result, matched := matchTemplate(reqArr[0]); matched {
		log.Debug().Msgf("Input: %s\nOutput: %v\n\n", reqArr, result)

		uri := fillTemplate(result.ServiceURI, result.PathParams)
		log.Debug().Msgf("URL formed : %s", uri)
		if len(reqArr) > 1 && !strings.EqualFold("", strings.Trim(reqArr[1], " ")) {
			result.ServiceURI = fmt.Sprintf("%s?%s", uri, reqArr[1])
		} else {
			result.ServiceURI = uri
		}
		log.Debug().Msgf("URL to call : %s", result.ServiceURI)
		return result, nil
	}
	return Output{}, errors.New("API mapping not found")
}

// TODO: enhance matching algo.
func matchTemplate(inputURL string) (Output, bool) {
	for _, entry := range cache.GetAllAPIRemapCache() {
		params, matched := match(entry.ExternalURI, inputURL)
		if matched {
			result := Output{
				ServiceURI: entry.ServiceURI,
				Backendinfo: BackendInfo{
					SvcName: entry.Backend.Service,
					PortStr: entry.Backend.Port,
				},
				PathParams: params,
			}
			return result, true
		}
	}
	return Output{}, false
}

func match(template, url string) (map[string]string, bool) {
	templateParts := strings.Split(template, "/")
	urlParts := strings.Split(url, "/")

	// Check if the number of parts in the template and URL are the same
	if len(templateParts) != len(urlParts) {
		return nil, false
	}

	params := make(map[string]string)
	for i, part := range templateParts {
		// Check if the part is a parameter (enclosed in {})
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			if urlParts[i] == "" {
				// Fail if the URL part corresponding to a parameter is empty
				return nil, false
			}
			paramName := part[1 : len(part)-1]
			params[paramName] = urlParts[i]
		} else if part != urlParts[i] {
			// If the part is not a parameter and does not match the URL part, return false
			return nil, false
		}
	}
	return params, true
}

func fillTemplate(template string, params map[string]string) string {
	re := regexp.MustCompile(`\{(\w+)\}`)
	result := re.ReplaceAllStringFunc(template, func(match string) string {
		key := match[1 : len(match)-1]
		if value, ok := params[key]; ok {
			return value
		}
		return match // If the key is not found, return the original placeholder
	})
	return result
}

func SubscribeToAPIMappingsEvents(tnc *tenancy_nexus_client.Clientset) {
	tenancy := tnc.TenancyMultiTenancy()
	tenancy.Subscribe()
	tenancy.Config().Subscribe()
	tenancy.Config().APIMappings("*").Subscribe()

	if _, err := tenancy.Config().APIMappings("*").RegisterAddCallback(StoreMappingToCache); err != nil {
		log.InfraErr(err).Msgf("error registering add callback for apimappings")
	}
	if _, err := tenancy.Config().APIMappings("*").RegisterDeleteCallback(removeMappingFromCache); err != nil {
		log.InfraErr(err).Msgf("error registering delete callback for apimappings")
	}
}

func StoreMappingToCache(amc *tenancy_nexus_client.ApimappingconfigAPIMappingConfig) {
	log.Debug().Msgf("received add event for 'APIMappings'| name:%s displayName: %s", amc.Name, amc.DisplayName())

	// Iterate over each API mapping
	log.Debug().Msgf("size of 'Mappings' %v", len(amc.Spec.Mappings))
	for _, api := range amc.Spec.Mappings {
		cache.APIRemapCache.Set(api.ExternalURI, common.APIMappingVO{
			ServiceURI: api.ServiceURI,
			Backend: amcV1.Backend{
				Service: amc.Spec.Backend.Service,
				Port:    amc.Spec.Backend.Port,
			},
		})
	}

	log.Debug().Msgf("storeMappingToCache | Size of cache : %d", len(cache.GetAllAPIRemapCache()))
}

func removeMappingFromCache(amc *tenancy_nexus_client.ApimappingconfigAPIMappingConfig) {
	log.Debug().Msgf("received delete event for 'APIMappings'| name:%s displayName: %s", amc.Name, amc.DisplayName())

	for _, mapping := range amc.Spec.Mappings {
		cache.APIRemapCache.Delete(mapping.ExternalURI)
	}
	log.Debug().Msgf("removeMappingFromCache | Size of cache : %d", len(cache.GetAllAPIRemapCache()))
}
