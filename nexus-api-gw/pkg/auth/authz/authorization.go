// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/auth/authn"
	"github.com/open-policy-agent/opa/rego"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	RegoPolicyPath  = "authz.rego"
	RegoQueryString = "data.authz.allow"
	ProjectBasePath = "/usr/local/"
)

type Policy struct {
	Query *rego.PreparedEvalQuery
}

type PolicyResult struct {
	Allow bool   `json:"allow"`
	Claim string `json:"claim"`
}

type PolicyInput struct {
	Resource  string   `json:"resource"`
	Method    string   `json:"method"`
	OrgID     string   `json:"orgId"`
	ProjectID string   `json:"projectId"`
	Roles     []string `json:"roles"`
}

var (
	policy  *Policy
	ctx     context.Context
	appName = "nexus-api-gw-authz"
	log     = logging.GetLogger(appName)
)

//nolint:gochecknoinits // Using init for bootstrapping is a valid exception.
func init() {
	ctx = context.Background()
	regoPath := ProjectBasePath + RegoPolicyPath

	if testing.Testing() {
		_, filename, _, _ := runtime.Caller(0)
		regoPath = filepath.Join(path.Dir(filename), RegoPolicyPath)
	}

	q, err := rego.New(
		rego.Query(RegoQueryString),
		rego.Load([]string{regoPath}, nil),
	).PrepareForEval(ctx)
	if err != nil {
		log.Fatal().Msgf("Error preparing Rego query: %v", err)
	}

	policy = &Policy{
		Query: &q,
	}
}

func VerifyAuthorization(jwtClaims authn.JwtData) *echo.HTTPError {
	policyInput := PolicyInput{
		Resource:  strings.Split(jwtClaims.URN, "?")[0],
		Method:    strings.ToLower(jwtClaims.Method),
		OrgID:     jwtClaims.ActiveOrgID,
		ProjectID: jwtClaims.ActiveProjectID,
		Roles:     jwtClaims.Claims.RealmAccess.Roles,
	}
	log.Debug().Msgf("policyInput for authz rego - %v", policyInput)
	results, err := policy.Query.Eval(ctx, rego.EvalInput(policyInput))
	if err != nil {
		log.InfraErr(err).Msg("Error evaluating Rego query")
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Error evaluating Rego query..%s", http.StatusText(http.StatusInternalServerError)),
		}
	}

	outputJSON, err := json.Marshal(results[0].Expressions[0].Value)
	if err != nil {
		log.InfraErr(err).Msg("Error marshaling result to JSON")
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Error marshaling result to JSON..%s", http.StatusText(http.StatusInternalServerError)),
		}
	}
	var resp PolicyResult
	if err := json.Unmarshal(outputJSON, &resp); err != nil {
		log.InfraErr(err).Msg("Error unmarshalling result to struct")
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Error unmarshalling result to struct..%s", http.StatusText(http.StatusInternalServerError)),
		}
	}

	if resp.Allow {
		return &echo.HTTPError{
			Code:    http.StatusAccepted,
			Message: http.StatusText(http.StatusAccepted),
		}
	}

	return &echo.HTTPError{
		Code:    http.StatusUnauthorized,
		Message: http.StatusText(http.StatusUnauthorized),
	}
}
