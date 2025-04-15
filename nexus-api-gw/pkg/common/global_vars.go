// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package common

var (
	Mode          string
	TENANCY       = false
	AUTHZDISABLED = false
	SSLEnabled    string
)

func IsModeAdmin() bool {
	if Mode == "" {
		return false
	}
	return Mode == "admin"
}

func IsTenancyMode() bool {
	return TENANCY
}

func IsHTTPSEnabled() bool {
	if SSLEnabled == "" {
		return false
	}
	return SSLEnabled == "true"
}

func IsAuthzDisabled() bool {
	return AUTHZDISABLED
}

var (
	CustomEndpoints   = map[string][]string{"allspark-ui": {"/login", "/*.js/", "/home", "/allspark-static/*"}}
	CustomEndpointSvc = "allspark-ui"
)
