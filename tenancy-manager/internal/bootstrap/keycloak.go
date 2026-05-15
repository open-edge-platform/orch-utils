// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// KeycloakClient is a small client for the subset of the Keycloak Admin
// REST API needed by the tenant-admin bootstrap.
type KeycloakClient struct {
	baseURL string
	realm   string
	token   string
	http    *http.Client
}

// NewKeycloakClient logs in to the master realm using admin credentials and
// returns a client bearing the obtained access token.
func NewKeycloakClient(ctx context.Context, baseURL, realm, adminUser, adminPass string) (*KeycloakClient, error) {
	httpc := &http.Client{Timeout: 30 * time.Second}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", "admin-cli")
	form.Set("username", adminUser)
	form.Set("password", adminPass)

	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", baseURL, realm)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("keycloak login: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("keycloak login: %s: %s", resp.Status, string(body))
	}

	var tr struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("keycloak login: empty access token")
	}

	return &KeycloakClient{
		baseURL: baseURL,
		realm:   realm,
		token:   tr.AccessToken,
		http:    httpc,
	}, nil
}

func (k *KeycloakClient) adminURL(format string, args ...any) string {
	return fmt.Sprintf("%s/admin/realms/%s%s", k.baseURL, k.realm, fmt.Sprintf(format, args...))
}

func (k *KeycloakClient) do(ctx context.Context, method, urlStr string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, rdr)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+k.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := k.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("keycloak %s %s: %s: %s", method, urlStr, resp.Status, string(rb))
	}
	if out != nil && resp.StatusCode != http.StatusNoContent && resp.ContentLength != 0 {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil && !errors.Is(err, io.EOF) {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// KCUser is a minimal representation of a Keycloak user.
type KCUser struct {
	ID            string `json:"id,omitempty"`
	Username      string `json:"username"`
	Email         string `json:"email,omitempty"`
	FirstName     string `json:"firstName,omitempty"`
	LastName      string `json:"lastName,omitempty"`
	Enabled       bool   `json:"enabled"`
	EmailVerified bool   `json:"emailVerified"`
}

// FindUserByUsername looks up a user by exact username. Returns (nil, nil)
// if not found.
func (k *KeycloakClient) FindUserByUsername(ctx context.Context, username string) (*KCUser, error) {
	u := k.adminURL("/users?username=%s&exact=true", url.QueryEscape(username))
	var users []KCUser
	if _, err := k.do(ctx, http.MethodGet, u, nil, &users); err != nil {
		return nil, err
	}
	for i := range users {
		if strings.EqualFold(users[i].Username, username) {
			return &users[i], nil
		}
	}
	return nil, nil
}

// CreateUser creates a user in the realm and returns its UUID. On 409
// Conflict (already exists) it transparently fetches and returns the
// existing user's UUID.
func (k *KeycloakClient) CreateUser(ctx context.Context, user KCUser) (string, error) {
	u := k.adminURL("/users")
	code, err := k.do(ctx, http.MethodPost, u, user, nil)
	if err != nil && code != http.StatusConflict {
		return "", err
	}
	// Either way, look up the resulting user.
	existing, lookupErr := k.FindUserByUsername(ctx, user.Username)
	if lookupErr != nil {
		return "", lookupErr
	}
	if existing == nil {
		return "", fmt.Errorf("user %s not found after create", user.Username)
	}
	return existing.ID, nil
}

// SetPassword sets a non-temporary password for the given user.
func (k *KeycloakClient) SetPassword(ctx context.Context, userID, password string) error {
	u := k.adminURL("/users/%s/reset-password", userID)
	body := map[string]any{
		"type":      "password",
		"value":     password,
		"temporary": false,
	}
	_, err := k.do(ctx, http.MethodPut, u, body, nil)
	return err
}

// KCGroup is a minimal representation of a Keycloak group.
type KCGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// GetGroupByPath resolves a group by its full path (e.g. "/<orgUUID>_Project-Manager-Group").
func (k *KeycloakClient) GetGroupByPath(ctx context.Context, path string) (*KCGroup, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := k.adminURL("/group-by-path%s", path)
	var g KCGroup
	if _, err := k.do(ctx, http.MethodGet, u, nil, &g); err != nil {
		return nil, err
	}
	if g.ID == "" {
		return nil, fmt.Errorf("group %q not found", path)
	}
	return &g, nil
}

// AddUserToGroup adds the user to the given group. Idempotent — Keycloak
// silently no-ops if the membership already exists.
func (k *KeycloakClient) AddUserToGroup(ctx context.Context, userID, groupID string) error {
	u := k.adminURL("/users/%s/groups/%s", userID, groupID)
	_, err := k.do(ctx, http.MethodPut, u, nil, nil)
	return err
}
