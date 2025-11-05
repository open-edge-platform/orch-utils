// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package secrets

// Config contains values used to configure the provider services.
type Config struct {
	AutoInit   bool
	AutoUnseal bool

	AuthOrchSvcsRoleMaxTTL  string
	AuthOIDCIdPAddr         string
	AuthOIDCIdPDiscoveryURL string
	AuthOIDCRoleMaxTTL      string
	AuthOIDCBoundIssuer     string
	AuthOIDCIssuerURL       string
}
