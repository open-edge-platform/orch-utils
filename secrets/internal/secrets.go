// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/open-edge-platform/orch-utils/secrets/internal/retry"
	"github.com/open-edge-platform/orch-utils/secrets"
)

// VaultKeysKubernetesSecretName is the name of the Kubernetes secret that contains encryption keys for unsealing Vault.
const VaultKeysKubernetesSecretName = "vault-keys"

func Configure( //nolint: cyclop
	ctx context.Context,
	log *zap.SugaredLogger,
	config *secrets.Config,
	secretsProviderSvc secrets.ProviderService,
	storageSvc secrets.StorageService,
) error {
	// Get the initialized status of Vault, retry in case Vault is not up yet
	var (
		initialized bool
		err         error
	)
	if err := retry.UntilItSucceeds(
		ctx,
		func() error {
			initialized, err = secretsProviderSvc.Initialized()
			if err != nil {
				log.Debugf("Error getting Vault initialized status, will retry: %s", err)
				return fmt.Errorf("get Vault initialized status: %w", err)
			}
			return nil
		},
		5*time.Second,
	); err != nil {
		return fmt.Errorf("retry: %w", err)
	}

	// Only initialize Vault iff it was not initialized before and the auto-init flag is set
	if !initialized && config.AutoInit {
		if err := initializeAndPersistKeys(ctx, log, secretsProviderSvc, storageSvc); err != nil {
			return fmt.Errorf("initialize and authenticate client: %w", err)
		}
	}

	// Wait for secret containing token to be created
	if err := retry.UntilItSucceeds(
		ctx,
		func() error {
			log.Infof("Trying to authenticate Vault client using token in secret %s...", VaultKeysKubernetesSecretName)

			if err := authenticateVaultUsingTokenInStorage(ctx, secretsProviderSvc, storageSvc); err != nil {
				log.Errorf("Error using token in secret %s, will retry: %s", VaultKeysKubernetesSecretName, err)
				return fmt.Errorf("authenticate with Vault token in secret %s: %w", VaultKeysKubernetesSecretName, err)
			}

			return nil
		},
		10*time.Second,
	); err != nil {
		return fmt.Errorf("retry wait for %s secret: %w", VaultKeysKubernetesSecretName, err)
	}

	if err := ConfigureAuth(ctx, log, config, secretsProviderSvc); err != nil {
		return fmt.Errorf("configure auth methods: %w", err)
	}

	// Revoke root token and delete secret if Vault was manually initialized
	if !config.AutoInit {
		if err := secretsProviderSvc.RevokeToken(); err != nil {
			return fmt.Errorf("revoke token: %w", err)
		}

		if err := storageSvc.Delete(
			ctx,
			"orch-platform",
			VaultKeysKubernetesSecretName,
		); err != nil {
			return fmt.Errorf("delete secret %s: %w", VaultKeysKubernetesSecretName, err)
		}
	}

	return nil
}

func authenticateVaultUsingTokenInStorage(
	ctx context.Context,
	secretsProviderSvc secrets.ProviderService,
	storageSvc secrets.StorageService,
) error {
	// Get secret
	values, err := storageSvc.Get(ctx, "orch-platform", VaultKeysKubernetesSecretName)
	if err != nil {
		return fmt.Errorf("get Vault keys: %w", err)
	}

	vaultKeys, ok := values[VaultKeysKubernetesSecretName]
	if !ok {
		return fmt.Errorf("keys not found") // Should never happen
	}

	var keys struct {
		RootToken string `json:"root_token"`
	}
	if err := json.Unmarshal([]byte(vaultKeys), &keys); err != nil {
		return fmt.Errorf("unmarshal keys: %w", err)
	}
	if keys.RootToken == "" {
		return fmt.Errorf("root_token must not be empty")
	}

	// Set token
	secretsProviderSvc.SetToken(keys.RootToken)

	return nil
}

func initializeAndPersistKeys(
	ctx context.Context,
	log *zap.SugaredLogger,
	secretsProviderSvc secrets.ProviderService,
	storageSvc secrets.StorageService,
) error {
	vaultKeys, err := secretsProviderSvc.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("initialize Vault: %w", err)
	}
	log.Info("Vault initialized. Saving Vault keys...")

	// Store Vault keys as a Kubernetes secret, retry forever on errors or the keys will be lost forever
	if err := retry.UntilItSucceeds(
		ctx,
		func() error {
			if err := storageSvc.Put(
				ctx,
				"orch-platform",
				VaultKeysKubernetesSecretName,
				map[string]string{
					VaultKeysKubernetesSecretName: vaultKeys,
				},
			); err != nil {
				log.Errorf("Error storing Vault keys, will retry: %s", err)
				return fmt.Errorf("store Vault keys: %w", err)
			}
			return nil
		},
		2*time.Second,
	); err != nil {
		return fmt.Errorf("retry: %w", err)
	}
	log.Infof("Vault keys saved as secret with name '%s'", VaultKeysKubernetesSecretName)

	return nil
}

func ConfigureAuth(
	ctx context.Context,
	log *zap.SugaredLogger,
	config *secrets.Config,
	secretsProviderSvc secrets.ProviderService,
) error {
	if err := secretsProviderSvc.CreateOrchSvcSecretsStore(); err != nil {
		return fmt.Errorf("create orch-svc secrets store: %w", err)
	}
	log.Info("Created Orchestrator service secrets store")

	log.Infof("Waiting for OIDC IdP to become ready...")
	if err := retry.UntilItSucceeds(
		ctx,
		func() error {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				config.AuthOIDCIdPDiscoveryURL,
				nil,
			)
			if err != nil {
				return nil
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Debugf("Get Keycloak config, will retry: %s", err.Error())
				return fmt.Errorf("get Keycloak config: %w", err)
			}
			defer resp.Body.Close()

			return nil
		},
		10*time.Second,
	); err != nil {
		return fmt.Errorf("wait for Keycloak: %w", err)
	}

	if err := secretsProviderSvc.CreateOIDCAuth(); err != nil {
		return fmt.Errorf("configure OIDC auth: %w", err)
	}
	log.Info("Configured OIDC IdP auth")

	return nil
}
