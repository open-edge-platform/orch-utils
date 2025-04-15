// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/config"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	DefaultNamespace = "default"

	DisplayNameLabelConst = "nexus/display_name"
)

var (
	appName = "nexus-api-gw-utils"
	log     = logging.GetLogger(appName)
)

func IsFileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func IsServerConfigValid(conf *config.Config) bool {
	if conf != nil {
		if conf.Server.Address != "" && conf.Server.CertPath != "" && conf.Server.KeyPath != "" {
			return true
		}
	}
	return false
}

func DumpReq(req *http.Request) {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	log.Debug().Msg(string(requestDump))
}

func GetDatamodelName(crdType string) string {
	return strings.Join(strings.Split(crdType, ".")[2:], ".")
}

func GetCrdType(kind, groupName string) string {
	return GetGroupResourceName(kind) + "." + groupName // eg roots.root.helloworld.com
}

func GetGroupResourceName(kind string) string {
	return strings.ToLower(ToPlural(kind)) // eg roots
}

// GetParentHierarchy constructs the parent in the format <roots.orgchart.vmware.org:default>.
func GetParentHierarchy(parents []string, labels map[string]string) []string {
	var hierarchy []string
	for _, parent := range parents {
		for key, val := range labels {
			if parent == key {
				hierarchy = append(hierarchy, key+":"+val)
			}
		}
	}
	return hierarchy
}

/*
	ConstructGVR constructs group, version, resource for a CRD Type.

Eg: For a given CRD type: roots.vmware.org and ApiVersion: vmware.org/v1,

	      group => vmware.org
		  resource => roots
		  version => v1
*/
func ConstructGVR(crdType string) schema.GroupVersionResource {
	parts := strings.Split(crdType, ".")
	return schema.GroupVersionResource{
		Group:    strings.Join(parts[1:], "."),
		Version:  "v1",
		Resource: parts[0],
	}
}
