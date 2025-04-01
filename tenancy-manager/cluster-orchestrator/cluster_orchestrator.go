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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	orgActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/orgactivewatcher.edge-orchestrator.intel.com/v1"
	orgwatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/orgwatcher.edge-orchestrator.intel.com/v1"
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	projectwatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectwatcher.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Handle to initialized nexus sdk client.
var (
	nexusClient *nexus_client.Clientset
	appName     = "cluster-orchestrator"
	log         = logging.GetLogger(appName)
)

func safeUnixTime() uint64 {
	t := time.Now().Unix()
	if t < 0 {
		return 0
	}
	return uint64(t)
}

// Callback function to be invoked when Org is added.
func processRuntimeOrgsAdd(org *nexus_client.RuntimeorgRuntimeOrg) {
	log.Info().Msgf("Runtime Orgs: %+v created\n", *org)

	//  **********************************************************************
	//   BUSINESS LOGIC: Implement ORG Creation Handling
	//  **********************************************************************

	watcherObj, err := org.GetActiveWatchers(context.Background(), appName)
	if err != nil {
		if nexus_client.IsChildNotFound(err) {
			// Register this app as an active watcher for this org.
			watcherObj, err = org.AddActiveWatchers(context.Background(), &orgActiveWatcherv1.OrgActiveWatcher{
				ObjectMeta: metav1.ObjectMeta{
					Name: appName,
				},
				Spec: orgActiveWatcherv1.OrgActiveWatcherSpec{
					StatusIndicator: orgActiveWatcherv1.StatusIndicationInProgress,
					Message:         "Creating",
					TimeStamp:       safeUnixTime(),
				},
			})
			if err != nil {
				log.Error().Msgf("Error %+v while creating watch %s for org %s\n", err, appName, org.DisplayName())
				return
			}
		}
	} else if watcherObj.Spec.StatusIndicator == orgActiveWatcherv1.StatusIndicationIdle {
		log.Info().Msgf("Skipping processing of orgactivewatcher %v as it is already created and set to IDLE", appName)
		return
	}

	// After processing, set the status to IDLE state.
	watcherObj.Spec = orgActiveWatcherv1.OrgActiveWatcherSpec{
		StatusIndicator: orgActiveWatcherv1.StatusIndicationIdle,
		Message:         "Created",
		TimeStamp:       safeUnixTime(),
	}

	err = watcherObj.Update(context.Background())
	if err != nil {
		log.Error().Msgf("Failed to update OrgActiveWatcher object with an error: %v", err)
		return
	}

	log.Info().Msgf("Active watcher %s created for Org %s\n", watcherObj.DisplayName(), org.DisplayName())
}

// Callback function to be invoked when Org is deleted.
func processRuntimeOrgsUpdate(_, newObj *nexus_client.RuntimeorgRuntimeOrg) {
	if newObj.Spec.Deleted {
		log.Info().Msgf("Orgs: %+v marked for deletion\n", newObj.DisplayName())

		//  **********************************************************************
		//   BUSINESS LOGIC: Implement ORG Deletion Handling
		//  **********************************************************************

		// Stop watching the org as it is marked for deletion.
		err := newObj.DeleteActiveWatchers(context.Background(), appName)
		if nexus_client.IsChildNotFound(err) {
			// This app has already stopped watching the org.
			log.Info().Msgf("App %s DOES NOT watch org %s\n", appName, newObj.DisplayName())
			return
		} else if err != nil {
			log.Error().Msgf("Error %+v while deleting watch %s for org %s\n", err, appName, newObj.DisplayName())
			return
		}
		log.Info().Msgf("Active watcher %s deleted for Org %s\n", appName, newObj.DisplayName())
	}
}

// Callback function to be invoked when Project is added.
func processRuntimeProjectsAdd(proj *nexus_client.RuntimeprojectRuntimeProject) {
	log.Info().Msgf("Runtime Project: %+v created\n", *proj)

	//  **********************************************************************
	//   BUSINESS LOGIC: Implement Project Creation Handling
	//  **********************************************************************

	// Register this app as an active watcher for this project.
	watcherObj, err := proj.GetActiveWatchers(context.Background(), appName)
	if err != nil {
		if nexus_client.IsChildNotFound(err) {
			// Register this app as an active watcher for this project.
			watcherObj, err = proj.AddActiveWatchers(context.Background(), &projectActiveWatcherv1.ProjectActiveWatcher{
				ObjectMeta: metav1.ObjectMeta{
					Name: appName,
				},
				Spec: projectActiveWatcherv1.ProjectActiveWatcherSpec{
					StatusIndicator: projectActiveWatcherv1.StatusIndicationInProgress,
					Message:         "Creating",
					TimeStamp:       safeUnixTime(),
				},
			})
			if err != nil {
				log.Error().Msgf("Error %+v while creating watch %s for project %s\n", err, appName, proj.DisplayName())
				return
			}
		}
	} else if watcherObj.Spec.StatusIndicator == projectActiveWatcherv1.StatusIndicationIdle {
		log.Info().Msgf("Skipping processing of projectactivewatcher %v as it is already created and set to IDLE", appName)
		return
	}

	// After processing, set the status to IDLE state.
	watcherObj.Spec = projectActiveWatcherv1.ProjectActiveWatcherSpec{
		StatusIndicator: projectActiveWatcherv1.StatusIndicationIdle,
		Message:         "Created",
		TimeStamp:       safeUnixTime(),
	}
	err = watcherObj.Update(context.Background())
	if err != nil {
		log.Error().Msgf("Failed to update ProjectActiveWatcher object with an error: %v", err)
		return
	}

	log.Info().Msgf("Active watcher %s created for Project %s\n", watcherObj.DisplayName(), proj.DisplayName())
}

// Callback function to be invoked when Project is deleted.
func processRuntimeProjectsUpdate(_, newObj *nexus_client.RuntimeprojectRuntimeProject) {
	if newObj.Spec.Deleted {
		log.Info().Msgf("Project: %+v marked for deletion\n", newObj.DisplayName())

		//  **********************************************************************
		//   BUSINESS LOGIC: Implement PROJECT Deletion Handling
		//  **********************************************************************

		// Stop watching the project as it is marked for deletion.
		err := newObj.DeleteActiveWatchers(context.Background(), appName)
		if nexus_client.IsChildNotFound(err) {
			// This app has already stopped watching the project.
			log.Error().Msgf("App %s DOES NOT watch project %s\n", appName, newObj.DisplayName())
			return
		} else if err != nil {
			log.Error().Msgf("Error %+v while deleting watch %s for project %s\n", err, appName, newObj.DisplayName())
			return
		}
		log.Info().Msgf("Active watcher %s deleted for project %s\n", appName, newObj.DisplayName())
	}
}

func main() {
	var (
		err        error
		kubeconfig string
	)
	flag.StringVar(&kubeconfig, "k", "", "Absolute path to the kubeconfig file. Defaults to ~/.kube/config.")
	flag.Parse()

	config := ctrl.GetConfigOrDie()
	nexusClient, err = nexus_client.NewForConfig(config)
	if err != nil {
		log.Panic().Msgf("Error: %v", err)
	}

	// Subscribe to Multi-Tenancy graph.
	// Subscribe() api empowers subscription to objects from datamodel.
	// What subscription does is to keep the local cache in sync with datamodel changes.
	// This sync is done in the background.
	nexusClient.SubscribeAll()

	cfg, err := nexusClient.TenancyMultiTenancy().GetConfig(context.Background())
	if err != nil {
		log.Panic().Msgf("Error: %v", err)
	}
	// Add orgwatchers that need to be notified.
	_, err = cfg.AddOrgWatchers(context.Background(), &orgwatcherv1.OrgWatcher{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
		},
	})
	if err != nil {
		log.Error().Msgf("Failed to add OrgWatcher %s, error: %v", appName, err)
		return
	}
	// Add projectwatchers that need to be notified.
	_, err = cfg.AddProjectWatchers(context.Background(), &projectwatcherv1.ProjectWatcher{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
		},
	})
	if err != nil {
		log.Error().Msgf("Failed to add ProjectWatcher %s, error: %v", appName, err)
		return
	}

	// API to subscribe and register a callback function that is invoked when a Org is added in the datamodel.
	// Register*Callback() has the effect of subscription and also invoking a callback to the application code
	// when there are datamodel changes to the objects of interest.
	_, err = nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").RegisterAddCallback(processRuntimeOrgsAdd)
	if err != nil {
		log.Error().Msgf("Failed to register 'Add' call back for runtime Org, error: %v", err)
		return
	}
	_, err = nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").RegisterUpdateCallback(processRuntimeOrgsUpdate)
	if err != nil {
		log.Error().Msgf("Failed to register 'Update' call back for runtime Org, error: %v", err)
		return
	}

	// API to subscribe and register a callback function that is invoked when a Project is added in the datamodel.
	// Register*Callback() has the effect of subscription and also invoking a callback to the application code
	// when there are datamodel changes to the objects of interest.
	_, err = nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").
		RegisterAddCallback(processRuntimeProjectsAdd)
	if err != nil {
		log.Error().Msgf("Failed to register 'Add' call back for runtime Project, error: %v", err)
		return
	}
	_, err = nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").
		RegisterUpdateCallback(processRuntimeProjectsUpdate)
	if err != nil {
		log.Error().Msgf("Failed to register 'Update' call back for runtime Project, error: %v", err)
		return
	}

	// Dummy code to make the program block, while it waits for events.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		<-sigs
		done <- true
	}()
	<-done
}
