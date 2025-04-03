/*
 * Copyright (C) 2025 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	config_helper "github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/tenancy"
	"github.com/rs/zerolog"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	appName = "tenancy-manager"
	log     = logging.GetLogger(appName)
)

func main() {
	var kubeconfig string
	flag.StringVar(&kubeconfig, "k", "", "Absolute path to the kubeconfig file. Defaults to ~/.kube/config.")
	useServiceAccount := flag.Bool("serviceaccount", false, "use serviceaccount")
	flag.Parse()

	// Setup log level
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "error"
	}
	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Msgf("Failed to configure logging: %v\n", err)
	}
	zerolog.SetGlobalLevel(lvl)

	config, err := config_helper.LoadConfig("/etc/config/config.yaml")
	if err != nil {
		config = config_helper.GetDefaultConfig()
	}

	// Initialize Nexus SDK, by pointing it to the K8s API endpoint where CRDs are to be stored.
	cfg, err := getConfig(kubeconfig, *useServiceAccount)
	if err != nil {
		log.Fatal().Msgf("unable to fetch kubeconfig: %v", err)
		// panic(err)
	}
	nexusClient, err := nexus_client.NewForConfig(cfg)
	if err != nil {
		log.Fatal().Msgf("unable to initialize nexusClient: %v", err)
		// panic(err)
	}
	reconciler := tenancy.NewReconciler(nexusClient, config)

	subscribeToTenancyEvents(nexusClient, reconciler)

	// Main wait loop for the App.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		<-sigs
		done <- true
	}()
	<-done
	log.Debug().Msg("Exiting")
}

// subscribeToTenancyEvents handles Tenancy subscriptions and callback registrations.

func subscribeToTenancyEvents(nexusClient *nexus_client.Clientset, reconciler *tenancy.Reconciler) {
	// Subscribe to Multi-Tenancy graph.
	// Subscribe() api empowers subscription to objects from datamodel.
	// What subscription does is to keep the local cache in sync with datamodel changes.
	// This sync is done in the background.
	nexusClient.SubscribeAll()

	// API to subscribe and register a callback function that is invoked when a Org is added in the datamodel.
	// Register*Callback() has the effect of subscription and also invoking a callback to the application code
	// when there are datamodel changes to the objects of interest.
	err := subscribeToConfigEvents(nexusClient, reconciler)
	if err != nil {
		log.Fatal().Msgf("Failed to register call backs, error: %v", err)
	}
	err = subscribeToRuntimeEvents(nexusClient, reconciler)
	if err != nil {
		log.Fatal().Msgf("Failed to register call backs, error: %v", err)
	}
}

func subscribeToConfigEvents(nexusClient *nexus_client.Clientset, reconciler *tenancy.Reconciler) error {
	tenant := nexusClient.TenancyMultiTenancy()

	_, err := tenant.Config().Orgs("*").RegisterAddCallback(reconciler.ProcessOrgsAdd)
	if err != nil {
		return fmt.Errorf("failed to register 'Add' call back to process config Org add, error: %w", err)
	}
	_, err = tenant.Config().Orgs("*").RegisterUpdateCallback(reconciler.ProcessOrgsUpdate)
	if err != nil {
		return fmt.Errorf("failed to register 'Update' call back to process config Org update, error: %w", err)
	}

	_, err = tenant.Config().Orgs("*").Folders("*").Projects("*").RegisterAddCallback(reconciler.ProcessProjectsAdd)
	if err != nil {
		return fmt.Errorf("failed to register 'Add' call back to process config Project add, error: %w", err)
	}
	_, err = tenant.Config().Orgs("*").Folders("*").Projects("*").RegisterUpdateCallback(reconciler.ProcessProjectsUpdate)
	if err != nil {
		return fmt.Errorf("failed to register 'Update' call back for config Project update, error: %w", err)
	}
	return nil
}

func subscribeToRuntimeEvents(nexusClient *nexus_client.Clientset, reconciler *tenancy.Reconciler) error {
	tenant := nexusClient.TenancyMultiTenancy()

	// Invoke OrgActiveWatchers Add, Update and Delete register callbacks.
	_, err := tenant.Runtime().Orgs("*").ActiveWatchers("*").RegisterAddCallback(reconciler.ProcessOrgActiveWatcherAdd)
	if err != nil {
		return fmt.Errorf("failed to register 'Add' call back to process OrgActiveWatcher add, error: %w", err)
	}
	_, err = tenant.Runtime().Orgs("*").ActiveWatchers("*").RegisterUpdateCallback(reconciler.ProcessOrgActiveWatcherUpdate)
	if err != nil {
		return fmt.Errorf("failed to register 'Update' call back to process OrgActiveWatcher update, error: %w", err)
	}
	_, err = tenant.Runtime().Orgs("*").ActiveWatchers("*").RegisterDeleteCallback(reconciler.ProcessOrgActiveWatcherDelete)
	if err != nil {
		return fmt.Errorf("failed to register 'Delete' call back to process OrgActiveWatcher delete, error: %w", err)
	}

	// Invoke ProjectActiveWatchers Add, Update and Delete register callbacks.
	_, err = tenant.Runtime().Orgs("*").Folders("*").Projects("*").ActiveWatchers("*").
		RegisterAddCallback(reconciler.ProcessProjectActiveWatcherAdd)
	if err != nil {
		return fmt.Errorf("failed to register 'Add' call back to process ProjectActiveWatcher add, error: %w", err)
	}
	_, err = tenant.Runtime().Orgs("*").Folders("*").Projects("*").ActiveWatchers("*").
		RegisterUpdateCallback(reconciler.ProcessProjectActiveWatcherUpdate)
	if err != nil {
		return fmt.Errorf("failed to register 'Update' call back to process ProjectActiveWatcher update, error: %w", err)
	}
	_, err = tenant.Runtime().Orgs("*").Folders("*").Projects("*").ActiveWatchers("*").
		RegisterDeleteCallback(reconciler.ProcessProjectActiveWatcherDelete)
	if err != nil {
		return fmt.Errorf("failed to register 'Delete' call back to process ProjectActiveWatcher delete, error: %w", err)
	}
	return nil
}

// getConfig initializes the Kubernetes client configuration.
func getConfig(kubeconfig string, useServiceAccount bool) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else if useServiceAccount {
		return rest.InClusterConfig()
	}
	return &rest.Config{Host: "localhost:9000"}, nil
}
