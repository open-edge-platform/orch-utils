// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// KubeClient is a minimal in-cluster REST client used to read and create
// Kubernetes Secrets. It uses the pod's service-account token and CA bundle
// mounted by kubelet at /var/run/secrets/kubernetes.io/serviceaccount.
//
// Only the small surface needed for tenant-admin bootstrap is implemented;
// we intentionally avoid the heavy k8s.io/client-go dependency.
type KubeClient struct {
	host    string
	token   string
	client  *http.Client
	defaultNS string
}

const (
	saTokenFile     = "/var/run/secrets/kubernetes.io/serviceaccount/token" //nolint:gosec // not a credential, just a path
	saCAFile        = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	saNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// NewInClusterKubeClient builds a KubeClient using the in-cluster service
// account. Returns an error if the SA token, CA, or kube API env vars are
// missing — i.e., when not running inside a pod.
func NewInClusterKubeClient() (*KubeClient, error) {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, fmt.Errorf("not running in-cluster (KUBERNETES_SERVICE_HOST/PORT not set)")
	}

	tokenBytes, err := os.ReadFile(saTokenFile)
	if err != nil {
		return nil, fmt.Errorf("read SA token: %w", err)
	}
	caBytes, err := os.ReadFile(saCAFile)
	if err != nil {
		return nil, fmt.Errorf("read SA CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("parse SA CA bundle")
	}

	ns, _ := os.ReadFile(saNamespaceFile)

	return &KubeClient{
		host:  fmt.Sprintf("https://%s:%s", host, port),
		token: strings.TrimSpace(string(tokenBytes)),
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
			},
		},
		defaultNS: strings.TrimSpace(string(ns)),
	}, nil
}

// DefaultNamespace returns the namespace the pod is running in.
func (k *KubeClient) DefaultNamespace() string {
	if k.defaultNS != "" {
		return k.defaultNS
	}
	return "default"
}

type secretManifest struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   secretMetadata    `json:"metadata"`
	Type       string            `json:"type"`
	Data       map[string]string `json:"data"` // base64-encoded values
}

type secretMetadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// GetSecretValue retrieves a single key's (raw) value from a Secret in the
// given namespace. Returns (value, true, nil) on success, ("", false, nil)
// if the Secret or key does not exist, and (_, _, err) on transport errors.
func (k *KubeClient) GetSecretValue(ctx context.Context, namespace, name, key string) (string, bool, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/secrets/%s", k.host, namespace, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+k.token)
	req.Header.Set("Accept", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("get secret %s/%s: %s: %s", namespace, name, resp.Status, string(body))
	}

	var m secretManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", false, fmt.Errorf("decode secret: %w", err)
	}
	enc, ok := m.Data[key]
	if !ok {
		return "", false, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", false, fmt.Errorf("decode secret data: %w", err)
	}
	return string(decoded), true, nil
}

// CreateOpaqueSecret creates an Opaque Secret in the given namespace.
// Returns nil if the Secret already exists (409 Conflict treated as success
// to keep the bootstrap path idempotent).
func (k *KubeClient) CreateOpaqueSecret(
	ctx context.Context,
	namespace, name string,
	labels map[string]string,
	data map[string]string,
) error {
	enc := make(map[string]string, len(data))
	for k, v := range data {
		enc[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	manifest := secretManifest{
		APIVersion: "v1",
		Kind:       "Secret",
		Metadata:   secretMetadata{Name: name, Namespace: namespace, Labels: labels},
		Type:       "Opaque",
		Data:       enc,
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s/secrets", k.host, namespace)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+k.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return nil // already exists; idempotent
	}
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create secret %s/%s: %s: %s", namespace, name, resp.Status, string(rb))
	}
	return nil
}
