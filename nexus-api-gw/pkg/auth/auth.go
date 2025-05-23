// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/auth/authn"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/auth/authz"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/common"
	tenancy_nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/rs/zerolog/log"
)

type Authenticator interface {
	VerifyJWT(c echo.Context, tenancyNexusClient *tenancy_nexus_client.Clientset, skipAuth bool) (authn.JwtData, *echo.HTTPError)
	VerifyAuthorization(jwtData authn.JwtData) *echo.HTTPError
	AuthenticateAndAuthorize(c echo.Context, tenancyNexusClient *tenancy_nexus_client.Clientset) (authn.JwtData, *echo.HTTPError)
}

type DefaultAuthenticator struct{}

func (a *DefaultAuthenticator) VerifyJWT(c echo.Context,
	tenancyNexusClient *tenancy_nexus_client.Clientset, skipAuth bool,
) (authn.JwtData, *echo.HTTPError) {
	return authn.VerifyJWT(c, tenancyNexusClient, skipAuth)
}

func (a *DefaultAuthenticator) VerifyAuthorization(jwtData authn.JwtData) *echo.HTTPError {
	return authz.VerifyAuthorization(jwtData)
}

func (a *DefaultAuthenticator) AuthenticateAndAuthorize(c echo.Context,
	tenancyNexusClient *tenancy_nexus_client.Clientset,
) (authn.JwtData, *echo.HTTPError) {
	var JwtClaims authn.JwtData
	var httpErr *echo.HTTPError
	if common.IsTenancyMode() {
		JwtClaims, httpErr = authn.VerifyJWT(c, tenancyNexusClient, false)
		if httpErr.Code != http.StatusOK {
			log.Warn().Msg("JWT Authentication Verification Failed")
			return JwtClaims, httpErr
		}

		if !common.IsAuthzDisabled() {
			httpErr = authz.VerifyAuthorization(JwtClaims)
			if httpErr.Code != http.StatusAccepted {
				log.Warn().Msg("JWT Authorization Verification Failed")
				return JwtClaims, httpErr
			}
		}
	}
	return JwtClaims, nil
}
