// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
	"go.uber.org/zap"

	"github.com/open-edge-platform/orch-utils/internal/retry"
	"github.com/open-edge-platform/orch-utils/secrets"
)

// ProviderService uses Hashicorp's Vault as a backing provider for managing secrets.
type ProviderService struct {
	addrs  []string
	client *vault.Client
	log    *zap.SugaredLogger
	config *secrets.Config
}

var _ secrets.ProviderService = &ProviderService{}

// NewSecretsProviderService returns a ProviderService struct.
func NewSecretsProviderService(
	log *zap.SugaredLogger,
	addrs []string,
	config *secrets.Config,
) (*ProviderService, error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("vault addresss cannot be empty")
	}

	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = addrs[0] // Pick any one. In HA mode, standby instances will redirect to primary

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("initialize Vault client: %w", err)
	}

	return &ProviderService{
		addrs:  addrs,
		client: client,
		log:    log,
		config: config,
	}, nil
}

// SetToken sets the authentication token in the Vault client.
func (svc *ProviderService) SetToken(t string) {
	svc.client.SetToken(t)
}

// RevokeToken revokes the currently authenticated token.
func (svc *ProviderService) RevokeToken() error {
	return svc.client.Auth().Token().RevokeSelf("")
}

// Initialized returns true if Vault is already initialized.
func (svc *ProviderService) Initialized() (bool, error) {
	return svc.client.Sys().InitStatus()
}

// Unseal Vault instance using the keys from initialization. In HA mode, each Vault instance must be unsealed.
func (svc *ProviderService) unsealVault(ctx context.Context, initResp *vault.InitResponse) error {
	for _, addr := range svc.addrs {
		if err := svc.client.SetAddress(addr); err != nil {
			return fmt.Errorf("set address %s: %w", addr, err)
		}

		var status *vault.SealStatusResponse

		// Retry unseal on failures since the keys will be lost forever if we exit now
		for _, key := range initResp.Keys {
			if err := retry.UntilItSucceeds(
				ctx,
				func() error {
					var err error

					if status, err = svc.client.Sys().Unseal(key); err != nil {
						svc.log.Errorf("Error unseal %s, will retry in 5 seconds: %s", addr, err)
						return fmt.Errorf("unseal: %w", err)
					}

					return nil
				},
				5*time.Second,
			); err != nil {
				return fmt.Errorf("retry: %w", err)
			}
		}

		// Stop when Vault instance becomes unsealed
		if !status.Sealed {
			continue
		}
	}
	svc.log.Info("Vault unsealed")
	return nil
}

// Initialize initializes the backing Vault instance and must be called first before any operations can be executed.
func (svc *ProviderService) Initialize(ctx context.Context) (string, error) {
	var initRequest *vault.InitRequest
	if svc.config.AutoUnseal {
		initRequest = &vault.InitRequest{
			RecoveryShares:    1,
			RecoveryThreshold: 1,
		}
	} else {
		initRequest = &vault.InitRequest{
			SecretShares:    1,
			SecretThreshold: 1,
		}
	}

	initResp, err := svc.client.Sys().Init(initRequest)
	if err != nil {
		return "", fmt.Errorf("initialize vault: %w", err)
	}
	svc.log.Info("Initialized Vault")

	if !svc.config.AutoUnseal {
		if err := svc.unsealVault(ctx, initResp); err != nil {
			return "", fmt.Errorf("unseal vault: %w", err)
		}
	}

	// Reset address back to initial instance
	if err := svc.client.SetAddress(svc.addrs[0]); err != nil {
		return "", fmt.Errorf("reset address: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(&initResp); err != nil {
		return "", fmt.Errorf("encode keys: %w", err)
	}

	return buf.String(), nil
}

func (svc *ProviderService) enableKubernetesAuth() error {
	if err := svc.client.Sys().EnableAuthWithOptions(
		"kubernetes",
		&vault.MountInput{
			Type: "kubernetes",
			Config: vault.MountConfigInput{
				DefaultLeaseTTL: "1h",
				MaxLeaseTTL:     "1h",
			},
		},
	); err != nil && !strings.Contains(err.Error(), "already in use") {
		return fmt.Errorf("enable Kubernetes auth: %w", err)
	}

	if _, err := svc.client.Logical().Write(
		"auth/kubernetes/config",
		map[string]interface{}{
			"kubernetes_host": "https://kubernetes.default.svc",
		},
	); err != nil {
		return fmt.Errorf("configure Kubernetes host: %w", err)
	}

	// Rate limit login requests
	if _, err := svc.client.Logical().Write(
		"sys/quotas/rate-limit/kubernetes",
		map[string]interface{}{
			"path":     "auth/kubernetes/*",
			"interval": "1s",  // Duration to enforce rate limiting
			"rate":     100.0, // Requests per interval
		},
	); err != nil {
		return fmt.Errorf("create rate limt: %w", err)
	}

	return nil
}

func (svc *ProviderService) CreateOrchSvcSecretsStore() error {
	if err := svc.enableKubernetesAuth(); err != nil {
		return fmt.Errorf("enable kubernetes auth: %w", err)
	}

	if err := svc.client.Sys().Mount(
		"secret",
		&vault.MountInput{Type: "kv-v2"},
	); err != nil && !strings.Contains(err.Error(), "already in use") {
		return fmt.Errorf("enable secret engine: %w", err)
	}

	// K8s pods with the orch-svc account are allowed to manage secrets and certificates issued by the Edge Node CA
	orchSvcPolicy := `
path "secret/*" {
	capabilities = ["create", "read", "update", "patch", "delete", "list"]
}
`

	if err := svc.client.Sys().PutPolicy(
		"orch-svc",
		orchSvcPolicy,
	); err != nil {
		return fmt.Errorf("create orch-svc policy: %w", err)
	}

	if _, err := svc.client.Logical().Write(
		"auth/kubernetes/role/orch-svc",
		map[string]interface{}{
			"bound_service_account_names": []string{
				"orch-svc",
				"alerting-monitor",
			},
			"bound_service_account_namespaces": []string{ // Create binding only in namespaces that need it
				"harbor-oci",
				"orch-app",
				"orch-cluster",
				"orch-infra",
				"orch-platform",
			},
			"policies":               "orch-svc",
			"token_ttl":              svc.config.AuthOrchSvcsRoleMaxTTL,
			"token_max_ttl":          svc.config.AuthOrchSvcsRoleMaxTTL,
			"token_explicit_max_ttl": svc.config.AuthOrchSvcsRoleMaxTTL,
		},
	); err != nil {
		return fmt.Errorf("create orch-svc auth role: %w", err)
	}

	return nil
}

func (svc *ProviderService) CreateOIDCAuth() error {
	if err := svc.client.Sys().EnableAuthWithOptions(
		"jwt",
		&vault.MountInput{
			Type: "jwt",
			Config: vault.MountConfigInput{
				DefaultLeaseTTL: svc.config.AuthOIDCRoleMaxTTL,
				MaxLeaseTTL:     svc.config.AuthOIDCRoleMaxTTL,
			},
		},
	); err != nil && !strings.Contains(err.Error(), "already in use") {
		return fmt.Errorf("enable JWT auth: %w", err)
	}

	// Authenticated entities with the "secrets-root-role" from IdP have root access to all paths
	if err := svc.client.Sys().PutPolicy(
		"secretsRootPolicy",
		"path \"*\" { capabilities = [\"create\", \"read\", \"update\", \"patch\", \"delete\", \"list\"]}",
	); err != nil {
		return fmt.Errorf("writing secretsRoot policy: %w", err)
	}

	// Write secretsRootPolicy to Client
	if _, err := svc.client.Logical().Write(
		"auth/jwt/role/secretsRoot",
		map[string]interface{}{
			"allowed_redirect_uris": svc.config.AuthOIDCIdPDiscoveryURL,
			"token_policies":        "secretsRootPolicy",
			"role_type":             "jwt",
			"user_claim":            "sub",
			"bound_claims": map[string][]string{
				"/realm_access/roles": {"secrets-root-role"},
			},
			"token_ttl":              svc.config.AuthOIDCRoleMaxTTL,
			"token_max_ttl":          svc.config.AuthOIDCRoleMaxTTL,
			"token_explicit_max_ttl": svc.config.AuthOIDCRoleMaxTTL,
		},
	); err != nil {
		return fmt.Errorf("writing secretsRoot role: %w", err)
	}

	// Build JWT config with optional parameters
	jwtConfig := map[string]interface{}{
		"oidc_discovery_url": svc.config.AuthOIDCIdPDiscoveryURL,
		"oidc_client_id":     "", // Empty instructs Vault to perform JWT authentication flow only
		"oidc_client_secret": "", // Empty instructs Vault to perform JWT authentication flow only
		"default_role":       "secretsRoot",
	}

	// Add optional parameters if configured
	if svc.config.AuthOIDCBoundIssuer != "" {
		jwtConfig["bound_issuer"] = svc.config.AuthOIDCBoundIssuer
	}
	// Use external issuer URL if specified (for cases where discovery URL differs from token issuer)
	if svc.config.AuthOIDCIssuerURL != "" {
		jwtConfig["oidc_discovery_url"] = svc.config.AuthOIDCIssuerURL
	}

	if _, err := svc.client.Logical().Write(
		"auth/jwt/config",
		jwtConfig,
	); err != nil {
		return fmt.Errorf("write JWT config: %w", err)
	}

	// Rate limit login requests
	if _, err := svc.client.Logical().Write(
		"sys/quotas/rate-limit/jwt",
		map[string]interface{}{
			"path":     "auth/jwt/*",
			"interval": "60s", // Duration to enforce rate limiting
			"rate":     100.0, // Requests per interval
		},
	); err != nil {
		return fmt.Errorf("create rate limt: %w", err)
	}

	return nil
}
