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
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	configv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/config.edge-orchestrator.intel.com/v1"
	tenancyv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/tenancy.edge-orchestrator.intel.com/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	config_helper "github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/tenancy"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	
	// Ensure default MultiTenancy and Config objects exist before subscriptions
	// This prevents cache miss issues when operations try to reference these core objects
	err = initializeDefaultTenancyObjects(nexusClient)
	if err != nil {
		log.Error().Msgf("Failed to initialize default tenancy objects: %v. Continuing anyway, objects may be created on first use.", err)
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

// initializeDefaultTenancyObjects ensures the default MultiTenancy and Config objects exist.
// This is critical for preventing cache miss issues and 30+ second retry loops when operations
// try to reference these core objects via labels.
func initializeDefaultTenancyObjects(nexusClient *nexus_client.Clientset) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to get the default MultiTenancy object
	tenant, err := nexusClient.GetTenancyMultiTenancy(ctx)
	if err == nil && tenant != nil {
		// MultiTenancy exists, proceed to check Config
		log.Debug().Msgf("Default MultiTenancy object already exists")
	} else if err != nil {
		// Object not found or other error - try to create it
		log.Debug().Msgf("Default MultiTenancy not found, attempting to create: %v", err)
		tenancyObj := &tenancyv1.MultiTenancy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		}
		tenant, createErr := nexusClient.AddTenancyMultiTenancy(ctx, tenancyObj)
		if createErr != nil {
			// Creation failed, might already exist due to race condition
			log.Debug().Msgf("Failed to create MultiTenancy: %v, attempting to retrieve existing object...", createErr)
			tenant, getErr := nexusClient.GetTenancyMultiTenancy(ctx)
			if getErr != nil || tenant == nil {
				return fmt.Errorf("failed to get or create default MultiTenancy: create_err=%w get_err=%w tenant_nil=%v",
					createErr, getErr, tenant == nil)
			}
		} else if tenant == nil {
			return fmt.Errorf("failed to create default MultiTenancy: returned nil with no error")
		}
		log.Debug().Msgf("Successfully created default MultiTenancy object")
	} else {
		// err == nil but tenant == nil - this shouldn't happen but handle it
		return fmt.Errorf("failed to initialize MultiTenancy: returned nil with no error")
	}
	
	// Now check if Config exists
	_, err = tenant.GetConfig(ctx)
	if err != nil {
		log.Debug().Msgf("Default Config not found in MultiTenancy, attempting to create: %v", err)
		configObj := &configv1.Config{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		}
		_, err = tenant.AddConfig(ctx, configObj)
		if err != nil {
			log.Error().Msgf("Failed to create default Config: %v", err)
			return fmt.Errorf("failed to create default Config: %w", err)
		}
		log.Debug().Msgf("Successfully created default Config object")
	} else {
		log.Debug().Msgf("Default MultiTenancy and Config objects already exist")
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
