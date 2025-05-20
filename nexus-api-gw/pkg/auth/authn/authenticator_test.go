// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package authn_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/auth/authn"
	tenancy_nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/stretchr/testify/assert"
)

const (
	// SharedSecretKey environment variable name for shared secret key for signing a token.
	SharedSecretKey         = "SHARED_SECRET_KEY"
	secretKey               = "randomSecretKey"
	writeRole               = "infra-manager-core-write-role"
	readRole                = "infra-manager-core-read-role"
	AllowMissingAuthClients = "ALLOW_MISSING_AUTH_CLIENTS"
	authclients             = "test-client"
)

// To create a request with an authorization header.
func createRequestWithAuthHeader(authScheme, authToken string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", authScheme, authToken))
	return req
}

// To create a request with a User-Agent header.
func createRequestWithUserAgent(userAgent string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("User-Agent", userAgent)
	return req
}

// To generates a valid JWT token for testing purposes.
func generateValidJWT(tb testing.TB) (jwtStr string, err error) {
	tb.Helper()
	claims := &jwt.MapClaims{
		"iss": "https://keycloak.kind.internal/realms/master",
		"exp": time.Now().Add(time.Hour).Unix(),
		"typ": "Bearer",
		"realm_access": map[string]interface{}{
			"roles": []string{
				writeRole,
				readRole,
			},
		},
	}
	tb.Setenv(SharedSecretKey, secretKey)
	tb.Setenv(AllowMissingAuthClients, authclients)
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims)
	jwtStr, err = token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return jwtStr, nil
}

func TestVerifyJWT(t *testing.T) {
	// Set up the environment variable for allowed missing auth clients.
	t.Setenv(AllowMissingAuthClients, "test-client")
	defer os.Unsetenv(AllowMissingAuthClients)

	jwtStr, err := generateValidJWT(t)
	if err != nil {
		t.Errorf("Error signing token: %v", err)
	}
	// Create an Echo instance for testing.
	e := echo.New()

	tests := []struct {
		name           string
		request        *http.Request
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "No Authorization header and no allowed missing auth client",
			request:        createRequestWithUserAgent("chrome"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing Authorization header",
		},
		{
			name:           "Invalid Authorization header format",
			request:        createRequestWithAuthHeader("invalid_format", "token"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "wrong Authorization header definition",
		},
		{
			name:           "Authorization header with Bearer scheme but invalid JWT token",
			request:        createRequestWithAuthHeader("Bearer", "invalid-token"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "JWT token is invalid or expired",
		},
		{
			name:           "Authorization header with non-Bearer scheme",
			request:        createRequestWithAuthHeader("Basic", "token"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Expecting \"Bearer\" Scheme to be sent",
		},
		{
			name:           "Authorization header with Bearer scheme with valid JWT token",
			request:        createRequestWithAuthHeader("Bearer", jwtStr),
			expectedStatus: http.StatusOK,
			expectedError:  "JWT token is valid, proceeding with processing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context with the request and recorder.
			c := e.NewContext(tt.request, httptest.NewRecorder())
			// Create a dummy next handler that returns OK status.

			nexusClient := tenancy_nexus_client.NewFakeClient()
			// Invoke interceptor.
			_, err := authn.VerifyJWT(c, nexusClient, false)

			errMsg, ok := err.Message.(string)
			if !ok {
				t.Error("err.Message is not of type string")
				return
			}

			fmt.Printf("%d.. %d.. %s\n", tt.expectedStatus, err.Code, errMsg)
			if tt.expectedStatus == err.Code {
				assert.Equal(t, tt.expectedStatus, err.Code)
			} else {
				var httpErr *echo.HTTPError
				if errors.As(err, &httpErr) {
					assert.Equal(t, http.StatusUnauthorized, httpErr.Code, "Expected an HTTP 401 Unauthorized error")
				} else {
					t.Errorf("Expected an echo.HTTPError, got %T", err)
				}
			}
		})
	}
}

func FuzzVerifyJWT(f *testing.F) {
	// Seed the fuzzer with initial test cases
	jwtStr, err := generateValidJWT(f)
	if err != nil {
		f.Errorf("Error signing token: %v", err)
	}
	f.Add("chrome", "invalid_format", "token", "Bearer", "invalid-token", "Basic", "Bearer", jwtStr)

	f.Fuzz(func(t *testing.T, userAgent, authScheme1, authToken1, authScheme2, authToken2,
		authScheme3, authToken3, validJWT string,
	) {
		// Set up the environment variable for allowed missing auth clients.
		t.Setenv(AllowMissingAuthClients, "test-client")
		defer os.Unsetenv(AllowMissingAuthClients)

		// Create an Echo instance for testing.
		e := echo.New()

		// Define the test cases with fuzzed inputs
		tests := []struct {
			name           string
			request        *http.Request
			expectedStatus int
			expectedError  string
		}{
			{
				name:           "No Authorization header and no allowed missing auth client",
				request:        createRequestWithUserAgent(userAgent),
				expectedStatus: http.StatusUnauthorized,
				expectedError:  "missing Authorization header",
			},
			{
				name:           "Invalid Authorization header format",
				request:        createRequestWithAuthHeader(authScheme1, authToken1),
				expectedStatus: http.StatusUnauthorized,
				expectedError:  "wrong Authorization header definition",
			},
			{
				name:           "Authorization header with Bearer scheme but invalid JWT token",
				request:        createRequestWithAuthHeader(authScheme2, authToken2),
				expectedStatus: http.StatusUnauthorized,
				expectedError:  "JWT token is invalid or expired",
			},
			{
				name:           "Authorization header with non-Bearer scheme",
				request:        createRequestWithAuthHeader(authScheme3, authToken3),
				expectedStatus: http.StatusUnauthorized,
				expectedError:  "Expecting \"Bearer\" Scheme to be sent",
			},
			{
				name:           "Authorization header with Bearer scheme with valid JWT token",
				request:        createRequestWithAuthHeader("Bearer", validJWT),
				expectedStatus: http.StatusOK,
				expectedError:  "JWT token is valid, proceeding with processing",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create a context with the request and recorder.
				c := e.NewContext(tt.request, httptest.NewRecorder())
				// Create a dummy next handler that returns OK status.

				nexusClient := tenancy_nexus_client.NewFakeClient()
				// Invoke interceptor.
				_, err := authn.VerifyJWT(c, nexusClient, false)

				errMsg, ok := err.Message.(string)
				if !ok {
					t.Error("err.Message is not of type string")
					return
				}

				fmt.Printf("%d.. %d.. %s\n", tt.expectedStatus, err.Code, errMsg)
				if tt.expectedStatus == err.Code {
					assert.Equal(t, tt.expectedStatus, err.Code)
				} else {
					var httpErr *echo.HTTPError
					if errors.As(err, &httpErr) {
						assert.Equal(t, http.StatusUnauthorized, httpErr.Code, "Expected an HTTP 401 Unauthorized error")
					} else {
						t.Errorf("Expected an echo.HTTPError, got %T", err)
					}
				}
			})
		}
	})
}
