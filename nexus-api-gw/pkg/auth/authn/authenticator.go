// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/cache"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/reconciler"
	tenancy_nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
)

// Regex patterns.
const (
	// Match projects or orgs.
	ProjectOrgPattern = `(?:/v[a-z0-9\-]+/projects/([^/]+))|(?:/v[a-z0-9\-]+/orgs/([^/]+))`
	// Match string beginning with /v1/projects/ pattern.
	ProjectPattern = `^/v[a-z0-9\-]+/projects(/.*)?$`
	// Match string beginning with /v1/projects/ pattern.
	ProjectOnlyPattern = `^/v[a-z0-9\-]+/projects(/[^/]+/?)?$`
	// Match string beginning with /v1/orgs/ pattern.
	OrgPattern = `^/v[a-z0-9\-]+/orgs(/[\w\-]+/?)?$`
	// Match string beginning with /v1/orgs/ pattern.
	OrgURIPattern = `^/v[a-zA-Z0-9]+/orgs/[^/]+(/[^/]+(/.*)?)$`
	// Regular expression to match the pattern.
	UserRolePattern = `([a-f0-9\-]+)_([a-f0-9\-]+)_(m|member-role)`
	// Regular expression to match the pattern.
	ProjectRolePattern = `([a-f0-9\-]+)_project-(read|write|update|delete)-role`
	// TenancyManager Reconcile Period.
	tmReconcileTime = 600 * time.Second
	OrgCache        = "orgCache"
	ProjectCache    = "projectCache"
)

// Compile the regex pattern.
var (
	re         = regexp.MustCompile(ProjectOrgPattern)
	projre     = regexp.MustCompile(ProjectPattern)
	projonlyre = regexp.MustCompile(ProjectOnlyPattern)
	orgre      = regexp.MustCompile(OrgPattern)
	orgurire   = regexp.MustCompile(OrgURIPattern)
	usrRole    = regexp.MustCompile(UserRolePattern)
	projRole   = regexp.MustCompile(ProjectRolePattern)
	appName    = "nexus-api-gw-authn"
	log        = logging.GetLogger(appName)
)

// Authentication constants.
const (
	authPairLen = 2
	authKey     = "authorization"
	bearer      = "bearer"
)

// Role and access structures.
type Roles struct {
	Roles []string `json:"roles"`
}

// Define the RealmAccess struct.
type RealmAccess struct {
	Roles []string `json:"roles"`
}

// Define the main TokenData struct with nested structs.
//
//nolint:tagliatelle // This struct has dependency on the keycloak token.
type TokenData struct {
	PreferredUsername string      `json:"preferred_username"`
	RealmAccess       RealmAccess `json:"realm_access"`
}

// Define the JwtData struct is used by AuthZ.
type JwtData struct {
	URN               string
	Method            string
	Claims            TokenData
	ActiveOrgID       string
	ActiveProjectID   string
	ActiveOrgDeleted  bool
	ActiveProjDeleted bool
	OrgName           string
}

func VerifyAuthenticationMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

// ExtractOrgUIDFromToken extracts the organization UID from the roles in the token.
func extractOrgsUIDFromToken(projects TokenData) string {
	for _, role := range projects.RealmAccess.Roles {
		if matches := usrRole.FindStringSubmatch(role); matches != nil {
			return matches[1]
		}
		if matches := projRole.FindStringSubmatch(role); matches != nil {
			return matches[1]
		}
	}
	log.InfraError("No roles found to extract Orgs from JWT. Roles from JWT: %#v", projects.RealmAccess.Roles).Msg("")
	return ""
}

// ExtractOrgProjectValues extracts organization and project values from a given path.
func extractOrgProjectValues(path string) (string, string) {
	if matches := re.FindStringSubmatch(path); matches != nil {
		// Directly return the matched results, ensuring they exist. Adding consts to avoid Magic number lint error
		const one, two = 1, 2
		org, project := "", ""
		if len(matches) > one {
			project = matches[one]
		}
		if len(matches) > two {
			org = matches[two]
		}
		return project, org
	}

	// Log an warning message if no matches were found
	log.Warn().Msgf("Failed  to extract Org & Project name from URI Path: '%s'", path)
	return "", ""
}

// MatchesProjPattern checks if a URL starts with /v*/projects/.
func MatchesProjPattern(url string) bool {
	return projre.MatchString(url)
}

// MatchesProjOnlyPattern checks if a URL starts with /v*/projects/.
func MatchesProjOnlyPattern(url string) bool {
	return projonlyre.MatchString(url)
}

// MatchesOrgPattern checks if a URL starts with /v1/orgs/.
func MatchesOrgPattern(url string) bool {
	return orgre.MatchString(url)
}

// MatchesOrgURIPattern checks if a URL starts with /v1/orgs/*/license or /v1/orgs/*/networks.
func MatchesOrgURIPattern(url string) bool {
	return orgurire.MatchString(url)
}

// GetActiveOrgDetails retrieves organization details from the cache.
func getActiveOrgDetails(tenancyNC *tenancy_nexus_client.Clientset, uid string) (string, string, bool) {
	return getFromGlobalCache(OrgCache, uid, tenancyNC)
}

func getFromGlobalCache(objType, key string, tenancyNC *tenancy_nexus_client.Clientset) (string, string, bool) {
	if strings.EqualFold(objType, OrgCache) {
		if value, ok := cache.GlobalOrgCache.Get(key); ok {
			return value.UID, value.Name, value.Deleted
		}
		log.Warn().Msgf("Key not found in Org Cache: %s", key)
		log.Info().Msgf("Updating Cache")
		tDM := reconciler.NewTenancyManager(tenancyNC, tmReconcileTime)
		tDM.PeriodicReconciler(context.Background())
		if value, ok := cache.GlobalOrgCache.Get(key); ok {
			return value.UID, value.Name, value.Deleted
		}
		log.InfraError("Key not found in Org: %s", key).Msg("")
	}
	if strings.EqualFold(objType, ProjectCache) {
		if value, ok := cache.GlobalProjectCache.Get(key); ok {
			return value.UID, "", value.Deleted
		}
		log.Warn().Msgf("Key not found in Project Cache: %s", key)
		log.Info().Msgf("Updating Cache")
		tDM := reconciler.NewTenancyManager(tenancyNC, tmReconcileTime)
		tDM.PeriodicReconciler(context.Background())
		if value, ok := cache.GlobalProjectCache.Get(key); ok {
			return value.UID, "", value.Deleted
		}
		log.InfraError("Key not found in Project: %s", key).Msg("")
	}
	return "", "", false
}

// GetActiveProjectIDs retrieves project ID from the cache.
func getActiveProjectIDs(tenancyNC *tenancy_nexus_client.Clientset, orgID, projName string) (string, bool) {
	key := fmt.Sprintf("%s_%s", orgID, projName)
	uid, _, deleted := getFromGlobalCache(ProjectCache, key, tenancyNC)
	return uid, deleted
}

// ParseClaimForAuthZ parses the claim map for authorization info.
func parseClaimForAuthZ(tenancyNC *tenancy_nexus_client.Clientset, uri, method string, claimsMap jwt.MapClaims) JwtData {
	var jwtData JwtData

	// Marshal claimsMap to JSON
	jsonData, err := json.Marshal(claimsMap)
	if err != nil {
		log.InfraErr(err).Msg("unmarshall error for JWT ClaimsMap object")
		return jwtData
	}

	// Unmarshal JSON to jwtData.Claims
	if err = json.Unmarshal(jsonData, &jwtData.Claims); err != nil {
		log.InfraErr(err).Msg("unmarshall error of jwt claims")
		return jwtData
	}

	// Set URN and Method
	jwtData.URN, jwtData.Method = uri, method

	// Extract project and organization names
	projName, orgName := extractOrgProjectValues(uri)

	// Process project pattern
	if MatchesProjPattern(uri) && orgName == "" {
		orgUID := extractOrgsUIDFromToken(jwtData.Claims)
		projID, deleted := getActiveProjectIDs(tenancyNC, orgUID, projName)
		jwtData.ActiveProjectID = projID
		jwtData.ActiveProjDeleted = deleted
		_, orgName, jwtData.ActiveOrgDeleted = getActiveOrgDetails(tenancyNC, orgUID)
		jwtData.ActiveOrgID = orgUID
		log.Debug().Msgf("orgUID=%v, orgName=%s, extracted from jwt", orgUID, orgName)
	}

	// Set organization name
	jwtData.OrgName = orgName
	return jwtData
}

func VerifyJWT(c echo.Context, tenancyNC *tenancy_nexus_client.Clientset, backendservice bool) (JwtData, *echo.HTTPError) {
	var jwtData JwtData

	authHeader := getAuthHeader(c)
	if authHeader == "" {
		log.Error().Msg("Authorization header is missingin jwt")
		return jwtData, newHTTPError(http.StatusUnauthorized, "Authorization header is missing")
	}

	authScheme, authToken, err := parseAuthHeader(authHeader)
	if err != nil {
		log.Error().Msgf("Error while parsing jwt: error=%s", err.Error())
		return jwtData, newHTTPError(http.StatusUnauthorized, err.Error())
	}

	if !strings.EqualFold(authScheme, bearer) {
		log.Error().Msgf("Invalid expected Authorization: \"Bearer\" scheme")
		return jwtData, newHTTPError(http.StatusUnauthorized, "Expecting \"Bearer\" scheme")
	}

	claims, err := validateJWT(authToken)
	if err != nil {
		log.Error().Msgf("Given JWT token is invalid or expired: error=%s", err.Error())
		return jwtData, newHTTPError(http.StatusUnauthorized, "JWT token is invalid or expired")
	}

	claimsMap, isMap := claims.(jwt.MapClaims)
	if !isMap {
		log.Error().Msgf("Failed to convert claims to a jwt map for parsing")
		return jwtData, newHTTPError(http.StatusForbidden, "Error converting claims to a map")
	}

	jwtData = parseClaimForAuthZ(tenancyNC, strings.Split(c.Request().RequestURI, "?")[0], c.Request().Method, claimsMap)

	if err := validateJWTData(tenancyNC, jwtData, c.Request().RequestURI, c.Request().Method, backendservice); err != nil {
		log.Error().Msgf("Failed to validate the jwt with err: %s", err.Error())
		return jwtData, err
	}

	log.Debug().Msgf("User URI '%s', request authenticated with valid jwt", jwtData.URN)
	return jwtData, &echo.HTTPError{
		Code:    http.StatusOK,
		Message: "User Request Authenticated",
	}
}

func getAuthHeader(c echo.Context) string {
	authHeader := c.Request().Header.Get("authorization")
	if authHeader == "" {
		authHeader = c.Request().Header.Get("Authorization")
	}
	return authHeader
}

func parseAuthHeader(authHeader string) (string, string, error) {
	authPair := strings.Split(authHeader, " ")
	if len(authPair) != authPairLen {
		return "", "", fmt.Errorf("wrong Authorization header definition")
	}
	return authPair[0], authPair[1], nil
}

func validateJWT(authToken string) (jwt.Claims, error) {
	jwtAuth := new(auth.JwtAuthenticator)
	return jwtAuth.ParseAndValidate(authToken)
}

func validateJWTData(
	tenancyNC *tenancy_nexus_client.Clientset,
	jwtData JwtData,
	uri, method string,
	backendservice bool,
) *echo.HTTPError {
	if err := validateBackendService(jwtData, uri, backendservice); err != nil {
		log.Error().Msgf("Failed to validateBackendService with error='%s'", err.Error())
		return err
	}

	if err := validateOrgData(tenancyNC, jwtData, uri, method); err != nil {
		log.Error().Msgf("Failed to validateOrgData with error='%s'", err.Error())
		return err
	}

	if err := validateDeletionStatus(jwtData, method); err != nil {
		log.Error().Msgf("Failed to validateDeletionStatus with error='%s'", err.Error())
		return err
	}

	return nil
}

func validateBackendService(jwtData JwtData, uri string, backendservice bool) *echo.HTTPError {
	if (MatchesProjOnlyPattern(uri) || jwtData.ActiveProjectID == "") && MatchesProjPattern(uri) && backendservice {
		log.InfraError("Unable to determine the Active ID for the project('%s') . Please check the JWT.", uri).Msg("")
		return newHTTPError(http.StatusBadRequest, "Unable to determine the Active ID for the project.")
	}
	log.Debug().
		Msgf("validateBackendService success. backendservice: %t, ActiveProjectID: %s,  uri: %s, MatchesProjPattern: %t",
			backendservice,
			jwtData.ActiveProjectID,
			uri,
			MatchesProjPattern(uri))
	return nil
}

func validateOrgData(tenancyNC *tenancy_nexus_client.Clientset, jwtData JwtData, uri, method string) *echo.HTTPError {
	if (jwtData.ActiveOrgID == "" || jwtData.OrgName == "") && MatchesProjPattern(uri) {
		log.InfraError("Unable to determine the organizations for URI request'%s'. Please check the JWT.", method).Msg("")
		return newHTTPError(http.StatusBadRequest, "Unable to determine the organizations.")
	}

	if MatchesOrgURIPattern(uri) {
		ctx := context.Background()
		orgObj, err := tenancyNC.TenancyMultiTenancy().Config().GetOrgs(ctx, jwtData.OrgName)
		if apierrors.IsNotFound(err) {
			log.Error().Msgf(
				"Org Not found. Please check the JWT. Method: %s, OrgName: %s, orgObj: %#v",
				method, jwtData.OrgName, orgObj)
			return newHTTPError(http.StatusConflict, "Unable to determine the organizations")
		}
	}

	log.Debug().
		Msgf("validateOrgData success. ActiveOrgID: %s, OrgName: %s,  uri: %s, MatchesProjPattern: %t",
			jwtData.ActiveOrgID,
			jwtData.OrgName,
			uri,
			MatchesProjPattern(uri))
	return nil
}

func validateDeletionStatus(jwtData JwtData, method string) *echo.HTTPError {
	if (jwtData.ActiveOrgDeleted || jwtData.ActiveProjDeleted) && method != http.MethodGet {
		log.InfraError(
			"Operation not supported. Requested resource 'URI: %s' is marked for delete. Please check the JWT.",
			jwtData.URN).Msg("")
		return newHTTPError(http.StatusBadRequest, "Operation not supported. Requested resource is marked for delete.")
	}
	log.Debug().
		Msgf(
			"validateDeletionStatus success. ActiveOrgDeleted: %t, ActiveProjDeleted: %t,  uri: %s",
			jwtData.ActiveOrgDeleted,
			jwtData.ActiveProjDeleted,
			jwtData.URN)
	return nil
}

func newHTTPError(code int, message string) *echo.HTTPError {
	return &echo.HTTPError{
		Code:    code,
		Message: fmt.Sprintf("%s. %s", message, http.StatusText(code)),
	}
}
