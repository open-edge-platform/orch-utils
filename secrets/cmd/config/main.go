// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/open-edge-platform/orch-utils/secrets/internal/retry"
	"github.com/open-edge-platform/orch-utils/secrets"
	"github.com/open-edge-platform/orch-utils/secrets/internal"
	"github.com/open-edge-platform/orch-utils/secrets/kubernetes"
	"github.com/open-edge-platform/orch-utils/secrets/vault"
)

// Values are injected when compiled through `make docker-build-secrets-config`.
var (
	// Version is the application version.
	Version = "development"
	// Revision is the application git revision.
	Revision = "unknown"
)

var (
	log            *zap.SugaredLogger
	kubeconfigPath string
)

func initializeConfigFromFlag(config *secrets.Config) {
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Optional file path to the cluster kubeconfig")
	flag.BoolVar(&config.AutoInit, "autoInit", false, "Initialize Vault and store seal keys in vault-keys secret")
	flag.BoolVar(&config.AutoUnseal, "autoUnseal", false, "Use AWS KMS to auto-unseal vault")
	// Authentication
	flag.StringVar(&config.AuthOrchSvcsRoleMaxTTL, "authOrchSvcsRoleMaxTTL", "1h", "Orchestrator services auth role token max TTL")                              //nolint: lll
	flag.StringVar(&config.AuthOIDCIdPAddr, "authOIDCIdPAddr", "http://platform-keycloak", "OIDC identity provider base address")                                //nolint: lll
	flag.StringVar(&config.AuthOIDCIdPDiscoveryURL, "authOIDCIdPDiscoveryURL", "http://platform-keycloak/realms/master", "OIDC identity provider discovery URL") //nolint: lll
	flag.StringVar(&config.AuthOIDCRoleMaxTTL, "authOIDCRoleMaxTTL", "1h", "OIDC JWT token max TTL")
	flag.Parse()
}

func newKubernetesCli() (*k8s.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig from file: %w", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("get in-cluster kubeconfig: %w", err)
		}
	}

	cli, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create client from kubeconfig: %w", err)
	}

	return cli, nil
}

func shutdownIstioProxy() {
	log.Infof("Checking if istio proxy is valid...")
	dialer := &net.Dialer{Timeout: time.Second * 10}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:15000")
	if err != nil {
		log.Info("Unable to find istio proxy, nothing to do.")
		return
	}
	log.Info("Shutting down istio proxy...")
	defer conn.Close()
	context, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	req, err := http.NewRequestWithContext(context, "POST", "http://127.0.0.1:15000/quitquitquit", nil)
	if err != nil {
		log.Errorf("Unable to create request: %s", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Unable to stop istio proxy: %s", err)
	}
	resp.Body.Close()
}

func vaultPodAddrs(ctx context.Context, k8sCli *k8s.Clientset) ([]string, error) {
	var addrs []string

	if err := retry.UntilItSucceeds(
		ctx,
		func() error {
			// Clear any previous addresses
			addrs = []string{}

			// Get the total number of desired replicas
			set, err := k8sCli.AppsV1().StatefulSets("orch-platform").Get(ctx, "vault", metav1.GetOptions{})
			if err != nil {
				log.Errorf("Error get stateful set: %s", err)
				return fmt.Errorf("get stateful set: %w", err)
			}
			if set.Spec.Replicas == nil {
				log.Errorf("Error stateful set replicas is nil and should never be")
				return fmt.Errorf("stateful set replicas is nil")
			}

			pods, err := k8sCli.CoreV1().Pods("orch-platform").List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Errorf("Error get pods: %s", err)
				return fmt.Errorf("list pods: %w", err)
			}

			// Get pod IPs to build list of addresses
			for _, pod := range pods.Items {
				if strings.HasPrefix(pod.Name, "vault-") && !strings.HasPrefix(pod.Name, "vault-agent") {
					if pod.Status.PodIP == "" {
						continue
					}

					addrs = append(addrs, fmt.Sprintf("http://%s:8200", pod.Status.PodIP))
				}
			}

			// Retry if pods might not have been scheduled yet or not ready
			if len(addrs) != int(*set.Spec.Replicas) {
				log.Errorf("Error running pods (%d) != desired (%d), will retry", len(addrs), int(*set.Spec.Replicas))
				return fmt.Errorf("running pods (%d) != desired (%d)", len(addrs), int(*set.Spec.Replicas))
			}

			return nil
		},
		5*time.Second,
	); err != nil {
		return nil, fmt.Errorf("retry: %w", err)
	}

	return addrs, nil
}

func main() {
	var code int
	defer func() { os.Exit(code) }()

	config := &secrets.Config{}

	logLevel := zap.LevelFlag("logLevel", zap.InfoLevel, "Log level (default \"info\")")
	initializeConfigFromFlag(config)

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level.SetLevel(*logLevel)

	logger, err := zapConfig.Build()
	if err != nil {
		log.Errorf("Error building log config: %s", err)
		code = 1
		return
	}
	defer logger.Sync() //nolint: errcheck

	log = logger.Sugar()

	log.Infof("Version: %s, Revision: %s", Version, Revision)

	k8sCli, err := newKubernetesCli()
	if err != nil {
		log.Errorf("Error creating kubernetes client: %s", err)
		code = 1
		return
	}

	ctx := context.Background()

	storageSvc, err := kubernetes.NewStorageService(k8sCli)
	if err != nil {
		log.Errorf("Error creating kubernetes storage client: %s", err)
		code = 1
		return
	}

	vaultAddrs, err := vaultPodAddrs(ctx, k8sCli)
	if err != nil {
		log.Errorf("Error getting vault pod addresses: %s", err)
		code = 1
		return
	}

	vaultSvc, err := vault.NewSecretsProviderService(log, vaultAddrs, config)
	if err != nil {
		log.Errorf("Error creating vault client: %s", err)
		code = 1
		return
	}

	if err := internal.Configure(ctx, log, config, vaultSvc, storageSvc); err != nil {
		log.Errorf("Error initializing vault: %s", err)
		code = 1
		return
	}

	log.Infof("Vault successfully configured and running")

	// We are running this program in a Kubernetes Job, the Job will
	// keeps running until all containers exited.
	// Here we need to ensure the istio sidecar container also stop so
	// the Kubernetes Job will be marked as "completed" state.
	shutdownIstioProxy()
}
