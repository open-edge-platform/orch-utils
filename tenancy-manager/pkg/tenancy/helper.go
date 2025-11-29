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

package tenancy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	orgsv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/org.edge-orchestrator.intel.com/v1"
	orgactivewatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/orgactivewatcher.edge-orchestrator.intel.com/v1"
	projectv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/project.edge-orchestrator.intel.com/v1"
	projectactivewatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

var ErrNotFound = errors.New("not found")

type Event string

const (
	Create Event = "CREATE"
	Delete Event = "DELETE"

	pollInterval = 5 * time.Second
	
	// Retry configuration for handling optimistic lock conflicts
	// Uses exponential backoff with jitter to handle K8s resource version conflicts
	maxStatusUpdateRetries     = 7                      // Increased from 3 for better concurrency handling
	statusUpdateRetryBaseDelay = 50 * time.Millisecond  // Base delay for exponential backoff
	maxRetryJitter             = 25 * time.Millisecond  // Maximum random jitter to prevent thundering herd
)

var (
	appName = "tenancy-manager"
	log     = logging.GetLogger(appName)
	rng     = rand.New(rand.NewSource(time.Now().UnixNano()))
	rngLock = &sync.Mutex{} // Protects concurrent access to rng
)

// calculateBackoffDelay computes exponential backoff with jitter for handling K8s optimistic lock conflicts.
// Formula: baseDelay * 2^attempt + random jitter
// This prevents thundering herd and gives k8s time to reconcile concurrent writes.
func calculateBackoffDelay(attempt int) time.Duration {
	// Exponential backoff: 50ms, 100ms, 200ms, 400ms, 800ms, 1.6s, 3.2s
	exponentialDelay := statusUpdateRetryBaseDelay * time.Duration(math.Pow(2, float64(attempt)))
	
	// Cap at 3.2 seconds to avoid excessive delays
	if exponentialDelay > 3200*time.Millisecond {
		exponentialDelay = 3200 * time.Millisecond
	}
	
	// Add random jitter (0-25ms) to prevent thundering herd
	// Multiple goroutines won't retry at the exact same time
	// Use mutex to protect concurrent access to RNG (math/rand is not thread-safe)
	rngLock.Lock()
	jitter := time.Duration(rng.Intn(int(maxRetryJitter)))
	rngLock.Unlock()
	
	return exponentialDelay + jitter
}

// hasOrgActiveWatcherSpecChanged compares the old and new specifications and returns true if they are different.
func hasOrgActiveWatcherSpecChanged(oldSpec, newSpec orgactivewatcherv1.OrgActiveWatcherSpec) bool {
	return oldSpec.StatusIndicator != newSpec.StatusIndicator ||
		oldSpec.Message != newSpec.Message
}

// hasProjectActiveWatcherSpecChanged compares the old and new specifications and returns true if they are different.
func hasProjectActiveWatcherSpecChanged(oldSpec, newSpec projectactivewatcherv1.ProjectActiveWatcherSpec) bool {
	return oldSpec.StatusIndicator != newSpec.StatusIndicator ||
		oldSpec.Message != newSpec.Message
}

// getRuntimeOrgUID returns the uID of the runtime org, corresponding to the input org.
// The determination is best effort and only if all relevant objects are found.
func getRuntimeOrgUID(c *nexus_client.Clientset, org *nexus_client.OrgOrg) string {
	uid := org.Status.OrgStatus.UID
	if uid == "" {
		runtimeOrg, err := c.TenancyMultiTenancy().Runtime().GetOrgs(context.Background(), org.DisplayName())
		if err == nil && runtimeOrg != nil {
			uid = string(runtimeOrg.UID)
		}
	}
	return uid
}

// getRuntimeProjectUID returns the uID of the runtime project, corresponding to the input project.
// The determination is best effort and only if all relevant objects are found.
func getRuntimeProjectUID(c *nexus_client.Clientset, project *nexus_client.ProjectProject) string {
	uid := project.Status.ProjectStatus.UID
	if uid == "" {
		folder, err := project.GetParent(context.Background())
		if err == nil && folder != nil {
			org, err := folder.GetParent(context.Background())
			if err == nil && org != nil {
				runtimeProject, err := c.TenancyMultiTenancy().Runtime().Orgs(org.DisplayName()).
					Folders(folder.DisplayName()).GetProjects(context.Background(), project.DisplayName())
				if err == nil {
					uid = string(runtimeProject.UID)
				}
			}
		}
	}
	return uid
}

// setOrgStatusWithRetry attempts to set org status with retry logic for handling conflict errors
func setOrgStatusWithRetry(client *nexus_client.Clientset, displayName, hashName string,
	status orgsv1.TenancyRequestStatus, msg string, eventType Event,
) error {
	var lastErr error
	
	for attempt := 0; attempt < maxStatusUpdateRetries; attempt++ {
		configOrg, err := getConfigOrg(client, displayName)
		if err != nil {
			return err
		}
		
		// Optimization: If status is already set to desired value, avoid unnecessary update
		if configOrg.Status.OrgStatus.StatusIndicator == status {
			log.Debug().Msgf("OrgStatus of %s (hashName: %s) already set to %v, no update needed",
				displayName, hashName, status)
			return nil
		}
		
		err = configOrg.SetOrgStatus(context.Background(), &orgsv1.OrgStatus{
			StatusIndicator: status,
			Message:         msg,
			TimeStamp:       safeUnixTime(),
			UID:             getRuntimeOrgUID(client, configOrg),
		})
		
		if err == nil {
			log.Debug().Msgf("Successfully set OrgStatus of %s (hashName: %s) to %v after %d attempt(s)",
				displayName, hashName, status, attempt+1)
			return nil
		}
		
		// Check if this is a conflict error that can be retried
		if nexus_client.IsConflict(err) || nexus_client.IsAlreadyExists(err) {
			lastErr = err
			if attempt < maxStatusUpdateRetries-1 {
				backoffDelay := calculateBackoffDelay(attempt)
				log.Debug().Msgf("Conflict updating OrgStatus of %s (hashName: %s), retrying with exponential backoff (attempt %d/%d, delay: %v)",
					displayName, hashName, attempt+1, maxStatusUpdateRetries, backoffDelay)
				time.Sleep(backoffDelay)
				// Retry by continuing loop - will refetch fresh object on next iteration
				continue
			}
			// On last attempt, exhausted all retries
			log.Error().Msgf("Exhausted retries (%d) for OrgStatus conflict on %s (hashName: %s), last error: %v",
				maxStatusUpdateRetries, displayName, hashName, err)
			return err
		}
		
		// For non-retryable errors, return immediately
		log.Error().Msgf("Failed to set OrgStatus of %s (hashName: %s) with non-retryable error: %v",
			displayName, hashName, err)
		return err
	}
	
	log.Error().Msgf("Exhausted retries (%d) setting OrgStatus of %s (hashName: %s), last error: %v",
		maxStatusUpdateRetries, displayName, hashName, lastErr)
	return lastErr
}

// setProjectStatusWithRetry attempts to set project status with retry logic for handling conflict errors
func setProjectStatusWithRetry(client *nexus_client.Clientset, displayName, hashName string,
	parentOrgName, parentFolderName string,
	status projectv1.TenancyRequestStatus, msg string, eventType Event,
) error {
	var lastErr error
	
	for attempt := 0; attempt < maxStatusUpdateRetries; attempt++ {
		configProject, err := getConfigProject(client, parentOrgName, parentFolderName, displayName)
		if err != nil {
			return err
		}
		
		// Optimization: If status is already set to desired value, avoid unnecessary update
		if configProject.Status.ProjectStatus.StatusIndicator == status {
			log.Debug().Msgf("ProjectStatus of %s (hashName: %s) already set to %v, no update needed",
				displayName, hashName, status)
			return nil
		}
		
		err = configProject.SetProjectStatus(context.Background(), &projectv1.ProjectStatus{
			StatusIndicator: status,
			Message:         msg,
			TimeStamp:       safeUnixTime(),
			UID:             getRuntimeProjectUID(client, configProject),
		})
		
		if err == nil {
			log.Debug().Msgf("Successfully set ProjectStatus of %s (hashName: %s) to %v after %d attempt(s)",
				displayName, hashName, status, attempt+1)
			return nil
		}
		
		// Check if this is a conflict error that can be retried
		if nexus_client.IsConflict(err) || nexus_client.IsAlreadyExists(err) {
			lastErr = err
			if attempt < maxStatusUpdateRetries-1 {
				backoffDelay := calculateBackoffDelay(attempt)
				log.Debug().Msgf("Conflict updating ProjectStatus of %s (hashName: %s), retrying with exponential backoff (attempt %d/%d, delay: %v)",
					displayName, hashName, attempt+1, maxStatusUpdateRetries, backoffDelay)
				time.Sleep(backoffDelay)
				// Retry by continuing loop - will refetch fresh object on next iteration
				continue
			}
			// On last attempt, exhausted all retries
			log.Error().Msgf("Exhausted retries (%d) for ProjectStatus conflict on %s (hashName: %s), last error: %v",
				maxStatusUpdateRetries, displayName, hashName, err)
			return err
		}
		
		// For non-retryable errors, return immediately
		log.Error().Msgf("Failed to set ProjectStatus of %s (hashName: %s) with non-retryable error: %v",
			displayName, hashName, err)
		return err
	}
	
	log.Error().Msgf("Exhausted retries (%d) setting ProjectStatus of %s (hashName: %s), last error: %v",
		maxStatusUpdateRetries, displayName, hashName, lastErr)
	return lastErr
}

func setOrgStatus(client *nexus_client.Clientset, displayName, hashName string,
	status orgsv1.TenancyRequestStatus, msg string, eventType Event,
) {
	orgMutex.Lock()
	defer orgMutex.Unlock()

	configOrg, defaultErr := getConfigOrg(client, displayName)
	if defaultErr != nil {
		if !errors.Is(defaultErr, ErrNotFound) {
			log.Panic().Msgf("Failed to get config org %s (hashName: %s) object to add status: %v",
				displayName, hashName, defaultErr)
		}
		return
	}
	if eventType != Delete && !configOrg.DeletionTimestamp.IsZero() && !Testing {
		log.Debug().Msgf("Org of %s (hashName: %s) is marked for delete, skip processing Create",
			displayName, hashName)
		return
	}
	if eventType == Create &&
		configOrg.Status.OrgStatus.StatusIndicator == orgsv1.StatusIndicationIdle {
		log.Debug().Msgf("OrgStatus of %s (hashName: %s) is already set to %v, skip processing",
			displayName, hashName, orgsv1.StatusIndicationIdle)
		return
	}
	log.Debug().Msgf("Setting OrgStatus of %s (hashName: %s) to %v", displayName, hashName, status)
	err := setOrgStatusWithRetry(client, displayName, hashName, status, msg, eventType)
	if err != nil {
		log.Error().Msgf("Failed to set OrgStatus of %s (hashName: %s) to %v after retries: %v. Object may be stuck in current state, watchers will attempt recovery.", displayName, hashName, status, err)
		// Don't panic - let watchers handle recovery via event reconciliation
		return
	}
	// Verify if the status is set as expected.
	verifyOrgStatus(client, displayName, hashName, status)
}

func verifyOrgStatus(client *nexus_client.Clientset, displayName, hashName string,
	status orgsv1.TenancyRequestStatus,
) {
	// Try to verify status with a small retry in case of cache lag
	for attempt := 0; attempt < 2; attempt++ {
		updatedOrg, defaultErr := getConfigOrg(client, displayName)
		if defaultErr != nil {
			if !errors.Is(defaultErr, ErrNotFound) {
				log.Error().Msgf("Failed to get config org %s (hashName: %s) during verification: %v",
					displayName, hashName, defaultErr)
			}
			return
		}
		
		if status == updatedOrg.Status.OrgStatus.StatusIndicator {
			log.Debug().Msgf("OrgStatus verification passed for %s: status is %v",
				displayName, status)
			return
		}
		
		// Status mismatch - log it and retry once in case of cache lag
		if attempt < 1 {
			log.Debug().Msgf("OrgStatus mismatch for %s (attempt %d/2): expected %v but found %v, retrying...",
				displayName, attempt+1, status, updatedOrg.Status.OrgStatus.StatusIndicator)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		
		// After retry still mismatched - log error with full context
		log.Error().Msgf("OrgStatus verification FAILED for %s (hashName: %s): Expected %v but found %v (TS: %d). This indicates a cache/visibility issue or concurrent modification.",
			displayName, hashName, status, updatedOrg.Status.OrgStatus.StatusIndicator,
			updatedOrg.Status.OrgStatus.TimeStamp)
		return
	}
}

func setProjectStatus(client *nexus_client.Clientset, displayName, hashName string,
	parentOrgName, parentFolderName string,
	status projectv1.TenancyRequestStatus, msg string, eventType Event,
) {
	projectMutex.Lock()
	defer projectMutex.Unlock()

	configProject, err := getConfigProject(client, parentOrgName, parentFolderName, displayName)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			log.Panic().Msgf("Failed to get config project %s object to add status: %v", displayName, err)
		}
		return
	}
	if eventType != Delete && !configProject.DeletionTimestamp.IsZero() && !Testing {
		log.Debug().Msgf("Proeject of %s (hashName: %s) is marked for delete, skip processing Create",
			displayName, hashName)
		return
	}
	if eventType == Create &&
		configProject.Status.ProjectStatus.StatusIndicator == projectv1.StatusIndicationIdle {
		log.Debug().Msgf("ProjectStatus of %s (hashName: %s) is already set to %v, skip processing",
			displayName, hashName, projectv1.StatusIndicationIdle)
		return
	}
	log.Debug().Msgf("Setting ProjectStatus of %s (hashName: %s) to %v", displayName, hashName, status)
	err = setProjectStatusWithRetry(client, displayName, hashName, parentOrgName, parentFolderName, status, msg, eventType)
	if err != nil {
		log.Error().Msgf("Failed to set ProjectStatus of %s (hashName: %s) to %v after retries: %v. Object may be stuck in current state, watchers will attempt recovery.", displayName, hashName, status, err)
		// Don't panic - let watchers handle recovery via event reconciliation
		return
	}
	// Verify if the status is set as expected.
	verifyProjectStatus(client, displayName, hashName, parentOrgName, parentFolderName, status)
}

func verifyProjectStatus(client *nexus_client.Clientset, displayName, hashName string,
	parentOrgName, parentFolderName string,
	status projectv1.TenancyRequestStatus,
) {
	// Try to verify status with a small retry in case of cache lag
	for attempt := 0; attempt < 2; attempt++ {
		updatedProject, err := getConfigProject(client, parentOrgName, parentFolderName, displayName)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				log.Error().Msgf("Failed to get config project %s during verification: %v",
					displayName, err)
			}
			return
		}
		
		if status == updatedProject.Status.ProjectStatus.StatusIndicator {
			log.Debug().Msgf("ProjectStatus verification passed for %s: status is %v",
				displayName, status)
			return
		}
		
		// Status mismatch - log it and retry once in case of cache lag
		if attempt < 1 {
			log.Debug().Msgf("ProjectStatus mismatch for %s (attempt %d/2): expected %v but found %v, retrying...",
				displayName, attempt+1, status, updatedProject.Status.ProjectStatus.StatusIndicator)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		
		// After retry still mismatched - log error with full context
		log.Error().Msgf("ProjectStatus verification FAILED for %s (hashName: %s): Expected %v but found %v (TS: %d). This indicates a cache/visibility issue or concurrent modification.",
			displayName, hashName, status, updatedProject.Status.ProjectStatus.StatusIndicator,
			updatedProject.Status.ProjectStatus.TimeStamp)
		return
	}
}

func (r *Reconciler) StartOrgCreateAcknowledgementTimer(timeoutInterval time.Duration, expectedOrgWatchers map[string]struct{},
	displayName, hashName string, runtimeOrg *nexus_client.RuntimeorgRuntimeOrg,
) {
	timeout := time.After(timeoutInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Debug().Msgf("Timer started for org %s (hashName: %s) create", displayName, hashName)
	defer log.Debug().Msgf("Timer ended for org %s (hashName: %s) create", displayName, hashName)

	for {
		select {
		case <-ticker.C:
			success, err := isOrgCreationSuccessful(r.Client, displayName, runtimeOrg, expectedOrgWatchers)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					log.InfraErr(err).Msgf("Creation of org %s (hashName: %s) failed", displayName, hashName)
					setOrgStatus(r.Client, displayName, hashName, orgsv1.StatusIndicationError,
						fmt.Sprintf("Org creation failed with an error: %v", err),
						Create)
				}
				return
			}
			if success {
				setOrgStatus(r.Client, displayName, hashName, orgsv1.StatusIndicationIdle,
					fmt.Sprintf("Org %s CREATE is complete", displayName),
					Create)
				log.Debug().Msgf("Creation of org %s (hashName: %s) is successful", displayName, hashName)
				return
			}
		case <-timeout:
			success, err := isOrgCreationSuccessful(r.Client, displayName, runtimeOrg, expectedOrgWatchers)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					log.InfraErr(err).Msgf(`Creation of org %s (hashName: %s) failed`, displayName, hashName)
					setOrgStatus(r.Client, displayName, hashName, orgsv1.StatusIndicationError,
						fmt.Sprintf("Org creation failed with an error: %v", err),
						Create)
				}
				return
			}
			if !success {
				// If there are unacknowledged watchers or any active watcher is not IDLE, set error state.
				log.Debug().Msgf("Timeout, active watchers %v haven't acknowledged org %s (hashName: %s), marking as 'ERROR'",
					getMapKeys(expectedOrgWatchers), displayName, hashName)
				setOrgStatus(r.Client, displayName, hashName, orgsv1.StatusIndicationError,
					fmt.Sprintf("Timeout, active watchers %v haven't acknowledged this org", getMapKeys(expectedOrgWatchers)),
					Create)
				log.Debug().Msgf("Org %s (hashName: %s) status transition to ERROR completed (timeout handling)", displayName, hashName)
			}
			return
		}
	}
}

func (r *Reconciler) StartOrgDeleteAcknowledgementTimer(timeoutInterval time.Duration, expectedOrgWatchers map[string]struct{},
	displayName, hashName string, runtimeOrg *nexus_client.RuntimeorgRuntimeOrg,
) {
	timeout := time.After(timeoutInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Debug().Msgf("Timer started for org %s (hashName: %s) delete", displayName, hashName)
	defer log.Debug().Msgf("Timer ended for org %s (hashName: %s) delete", displayName, hashName)

	for {
		select {
		case <-ticker.C:
			success, _, err := isOrgDeletionSuccessful(r.Client, displayName, runtimeOrg, expectedOrgWatchers)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					log.InfraErr(err).Msgf("Deletion of org %s (hashName: %s) failed", displayName, hashName)
					setOrgStatus(r.Client, displayName, hashName,
						orgsv1.StatusIndicationError,
						fmt.Sprintf("Org deletion failed with an error: %v", err),
						Delete)
				}
				return
			}
			if success {
				log.Debug().Msgf("Deletion of org %s (hashName: %s) is successful", displayName, hashName)
				return
			}
		case <-timeout:
			success, currentActiveWatchers, err := isOrgDeletionSuccessful(r.Client, displayName, runtimeOrg,
				expectedOrgWatchers)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					log.InfraErr(err).Msgf("Deletion of org %s (hashName: %s) failed", displayName, hashName)
					setOrgStatus(r.Client, displayName, hashName,
						orgsv1.StatusIndicationError,
						fmt.Sprintf("Org deletion failed with an error: %v", err),
						Delete)
				}
				return
			}
			if !success {
				// If at least one active watcher is found in expectedOrgWatchers, then mark org as error state.
				log.Debug().Msgf("Timeout, cannot be deleted as active watchers %v have acknowledged org %s (hashName: %s), "+
					"marking as 'ERROR'", getMapKeys(currentActiveWatchers), displayName, hashName)
				setOrgStatus(r.Client, displayName, hashName,
					orgsv1.StatusIndicationError,
					fmt.Sprintf("Timeout, cannot be deleted as active watchers %v have acknowledged this org",
						getMapKeys(currentActiveWatchers)),
					Delete)
			}
			return
		}
	}
}

func (r *Reconciler) StartProjectCreateAcknowledgementTimer(timeoutInterval time.Duration,
	expectedProjectWatchers map[string]struct{},
	displayName, hashName, orgName, folderName string,
	runtimeProject *nexus_client.RuntimeprojectRuntimeProject,
) {
	timeout := time.After(timeoutInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Debug().Msgf("Timer started for project %s (hashName: %s) create", displayName, hashName)
	defer log.Debug().Msgf("Timer ended for project %s (hashName: %s) create", displayName, hashName)

	for {
		select {
		case <-ticker.C:
			success, err := isProjectCreationSuccessful(r.Client,
				displayName, orgName, folderName, runtimeProject,
				expectedProjectWatchers)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					log.InfraErr(err).Msgf("Creation of project %s (hashName: %s) failed", displayName, hashName)
					setProjectStatus(r.Client, displayName, hashName, orgName, folderName,
						projectv1.StatusIndicationError,
						fmt.Sprintf("Project creation failed with an error: %v", err),
						Create)
				}
				// If config object not found, do nothing, simply return.
				return
			}
			if success {
				setProjectStatus(r.Client, displayName, hashName, orgName, folderName,
					projectv1.StatusIndicationIdle,
					fmt.Sprintf("Project %s CREATE is complete", displayName),
					Create)
				log.Debug().Msgf("Creation of project %s (hashName: %s) is successful", displayName, hashName)
				return
			}
		case <-timeout:
			success, err := isProjectCreationSuccessful(r.Client,
				displayName, orgName, folderName,
				runtimeProject, expectedProjectWatchers)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					log.InfraErr(err).Msgf(`Creation of project %s (hashName: %s) failed`, displayName, hashName)
					setProjectStatus(r.Client, displayName, hashName, orgName, folderName,
						projectv1.StatusIndicationError,
						fmt.Sprintf("Project creation failed with an error: %v", err),
						Create)
				}
				return
			}
			if !success {
				// If there are unacknowledged watchers or any active watcher is not IDLE, set error state.
				log.Debug().Msgf("Timeout, active watchers %v haven't acknowledged project %s (hashName: %s), marking as 'ERROR'",
					getMapKeys(expectedProjectWatchers), displayName, hashName)
				setProjectStatus(r.Client, displayName, hashName, orgName, folderName,
					projectv1.StatusIndicationError,
					fmt.Sprintf("Timeout, active watchers %v haven't acknowledged this project",
						getMapKeys(expectedProjectWatchers)),
					Create)
				log.Debug().Msgf("Project %s (hashName: %s) status transition to ERROR completed (timeout handling)", displayName, hashName)
			}
			return
		}
	}
}

func (r *Reconciler) StartProjectDeleteAcknowledgementTimer(timeoutInterval time.Duration,
	expectedProjectWatchers map[string]struct{},
	displayName, hashName, orgName,
	folderName string, runtimeProject *nexus_client.RuntimeprojectRuntimeProject,
) {
	timeout := time.After(timeoutInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Debug().Msgf("Timer started for project %s (hashName: %s) delete", displayName, hashName)
	defer log.Debug().Msgf("Timer ended for project %s (hashName: %s) delete", displayName, hashName)

	for {
		select {
		case <-ticker.C:
			success, _, err := isProjectDeletionSuccessful(r.Client,
				displayName, orgName, folderName, runtimeProject, expectedProjectWatchers)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					log.InfraErr(err).Msgf("Deletion of project %s (hashName: %s) failed", displayName, hashName)
					setProjectStatus(r.Client, displayName, hashName, orgName, folderName,
						projectv1.StatusIndicationError,
						fmt.Sprintf("Project deletion failed with an error: %v", err),
						Delete)
				}
				return
			}
			if success {
				log.Debug().Msgf("Deletion of project %s (hashName: %s) is successful", displayName, hashName)
				return
			}
		case <-timeout:
			success, currentActiveWatchers, err := isProjectDeletionSuccessful(r.Client,
				displayName, orgName, folderName, runtimeProject, expectedProjectWatchers)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					log.InfraErr(err).Msgf(`Deletion of project %s (hashName: %s) failed`, displayName, hashName)
					setProjectStatus(r.Client, displayName, hashName, orgName, folderName,
						projectv1.StatusIndicationError,
						fmt.Sprintf("Project deletion failed with an error: %v", err),
						Delete)
				}
				return
			}
			if !success {
				// If at least one active watcher is found in expectedProjectWatchers, then mark project as error state.
				log.Debug().Msgf("Timeout, cannot be deleted as active watchers %v have acknowledged project %s "+
					"(hashName: %s), marking as 'ERROR'", getMapKeys(currentActiveWatchers), displayName, hashName)
				setProjectStatus(r.Client, displayName, hashName, orgName, folderName,
					projectv1.StatusIndicationError,
					fmt.Sprintf("Timeout, cannot be deleted as active watchers %v have acknowledged this project",
						getMapKeys(currentActiveWatchers)),
					Delete)
			}
			return
		}
	}
}

func isOrgCreationSuccessful(client *nexus_client.Clientset,
	displayName string, runtimeOrg *nexus_client.RuntimeorgRuntimeOrg,
	expectedOrgWatchers map[string]struct{},
) (bool, error) {
	configOrg, err := getConfigOrg(client, displayName)
	if err != nil {
		return false, err
	}

	// If org is marked for deletion or success, exit early.
	if !configOrg.DeletionTimestamp.IsZero() ||
		configOrg.Status.OrgStatus.StatusIndicator == orgsv1.StatusIndicationIdle {
		return true, nil
	}

	// Create a local copy to avoid mutating the input map
	// This is critical: we must not modify expectedOrgWatchers as it's used
	// in error messages and subsequent polls to track pending watchers
	pendingWatchers := make(map[string]struct{})
	for k, v := range expectedOrgWatchers {
		pendingWatchers[k] = v
	}

	activeWatchersIter := runtimeOrg.GetAllActiveWatchersIter(context.Background())
	for {
		watcher, err := activeWatchersIter.Next(context.Background())
		if err != nil {
			fmt.Printf("Error retrieving next watcher: %v", err)
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := pendingWatchers[watcher.DisplayName()]; !exists {
			// Watcher is not in the expected list, skip it
			continue
		}
		if watcher.Spec.StatusIndicator != orgactivewatcherv1.StatusIndicationIdle {
			// Watcher is in expected list but not IDLE yet, keep it in the map
			continue
		}
		// Watcher is in the expected list and is IDLE, so remove it from the local copy
		delete(pendingWatchers, watcher.DisplayName())
	}

	// If all watchers acknowledged the org, we can exit.
	return len(pendingWatchers) == 0, nil
}

func isOrgDeletionSuccessful(client *nexus_client.Clientset,
	displayName string, runtimeOrg *nexus_client.RuntimeorgRuntimeOrg,
	expectedOrgWatchers map[string]struct{},
) (bool, map[string]struct{}, error) {
	configOrg, err := getConfigOrg(client, displayName)
	if err != nil {
		return false, nil, err
	}

	// Verify if the object is still marked to be deleted. If not, simply return.
	if configOrg.DeletionTimestamp.IsZero() {
		return true, nil, nil
	}

	// Retrieve all active watchers.
	// If at least one active watcher present, then simply return false.
	currentActiveWatchers := make(map[string]struct{})
	activeWatchersIter := runtimeOrg.GetAllActiveWatchersIter(context.Background())
	for {
		watcher, err := activeWatchersIter.Next(context.Background())
		if err != nil {
			log.InfraError("Error retrieving next watcher: %v", err).Msg("")
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := expectedOrgWatchers[watcher.DisplayName()]; exists {
			currentActiveWatchers[watcher.DisplayName()] = struct{}{}
		}
	}
	if len(currentActiveWatchers) > 0 {
		return false, currentActiveWatchers, nil
	}
	return true, currentActiveWatchers, nil
}

func isProjectCreationSuccessful(client *nexus_client.Clientset, displayName, orgName, folderName string,
	runtimeProject *nexus_client.RuntimeprojectRuntimeProject, expectedProjectWatchers map[string]struct{},
) (bool, error) {
	configProject, err := getConfigProject(client, orgName, folderName, displayName)
	if err != nil {
		return false, err
	}

	// If project is marked for deletion or success, exit early.
	if !configProject.DeletionTimestamp.IsZero() ||
		configProject.Status.ProjectStatus.StatusIndicator == projectv1.StatusIndicationIdle {
		return true, nil
	}

	// Create a local copy to avoid mutating the input map
	// This is critical: we must not modify expectedProjectWatchers as it's used
	// in error messages and subsequent polls to track pending watchers
	pendingWatchers := make(map[string]struct{})
	for k, v := range expectedProjectWatchers {
		pendingWatchers[k] = v
	}

	activeWatchersIter := runtimeProject.GetAllActiveWatchersIter(context.Background())
	for {
		watcher, err := activeWatchersIter.Next(context.Background())
		if err != nil {
			log.InfraError("Error retrieving next watcher: %v", err).Msg("")
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := pendingWatchers[watcher.DisplayName()]; !exists {
			// Watcher is not in the expected list, skip it
			continue
		}
		if watcher.Spec.StatusIndicator != projectactivewatcherv1.StatusIndicationIdle {
			// Watcher is in expected list but not IDLE yet, keep it in the map
			continue
		}
		// Watcher is in the expected list and is IDLE, so remove it from the local copy
		delete(pendingWatchers, watcher.DisplayName())
	}

	// If all watchers acknowledged the project, we can exit.
	return len(pendingWatchers) == 0, nil
}

func isProjectDeletionSuccessful(client *nexus_client.Clientset, displayName, orgName, folderName string,
	runtimeProject *nexus_client.RuntimeprojectRuntimeProject, expectedProjectWatchers map[string]struct{},
) (bool, map[string]struct{}, error) {
	configProject, err := getConfigProject(client, orgName, folderName, displayName)
	if err != nil {
		return false, nil, err
	}

	// Verify if the object is still marked to be deleted. If not, simply return.
	if configProject.DeletionTimestamp.IsZero() {
		return true, nil, nil
	}

	// Retrieve all active watchers.
	// If at least one active watcher present, then mark the project as error state.
	currentActiveWatchers := make(map[string]struct{})
	activeWatchersIter := runtimeProject.GetAllActiveWatchersIter(context.Background())
	for {
		watcher, err := activeWatchersIter.Next(context.Background())
		if err != nil {
			log.InfraError("Error retrieving next watcher: %v", err).Msg("")
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := expectedProjectWatchers[watcher.DisplayName()]; exists {
			currentActiveWatchers[watcher.DisplayName()] = struct{}{}
		}
	}
	if len(currentActiveWatchers) > 0 {
		return false, currentActiveWatchers, nil
	}
	return true, currentActiveWatchers, nil
}

func getConfigOrg(client *nexus_client.Clientset, name string) (*nexus_client.OrgOrg, error) {
	config, err := client.TenancyMultiTenancy().GetConfig(context.Background())
	if err != nil {
		return nil, err
	}
	
	// Retry logic for recently-created orgs that may not be in cache yet.
	// The cache may lag slightly after org creation, so retry once with backoff.
	for attempt := 0; attempt < 2; attempt++ {
		configOrg, err := config.GetOrgs(context.Background(), name)
		if err == nil {
			if attempt > 0 {
				log.Debug().Msgf("Found org %s after %d retry attempt(s)", name, attempt)
			}
			return configOrg, nil
		}
		
		// If it's not a "not found" error, return immediately (other errors shouldn't be retried)
		if !nexus_client.IsChildNotFound(err) {
			return nil, err
		}
		
		if attempt < 1 {
			// Wait before retrying to allow informer to sync the newly created object
			log.Debug().Msgf("Org %s not found in cache, retrying after backoff (attempt %d/2)", name, attempt+1)
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	// Org not found after retries
	log.Debug().Msgf("Org %s not found after cache retry attempts", name)
	return nil, ErrNotFound
}

func getConfigProject(client *nexus_client.Clientset, orgName, folderName, projectName string,
) (*nexus_client.ProjectProject, error) {
	configOrg, err := getConfigOrg(client, orgName)
	if err != nil {
		return nil, err
	}
	
	// Retry logic for recently-created projects that may not be in cache yet.
	// The cache may lag slightly after project creation, so retry once with backoff.
	for attempt := 0; attempt < 2; attempt++ {
		configProject, err := client.TenancyMultiTenancy().Config().Orgs(configOrg.DisplayName()).
			Folders(folderName).GetProjects(context.Background(), projectName)
		if err == nil {
			if attempt > 0 {
				log.Debug().Msgf("Found project %s after %d retry attempt(s)", projectName, attempt)
			}
			return configProject, nil
		}
		
		// If it's not a "not found" error, return immediately (other errors shouldn't be retried)
		if !nexus_client.IsNotFound(err) {
			return nil, err
		}
		
		if attempt < 1 {
			// Wait before retrying to allow informer to sync the newly created object
			log.Debug().Msgf("Project %s not found in cache, retrying after backoff (attempt %d/2)", projectName, attempt+1)
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	// Project not found after retries
	log.Debug().Msgf("Project %s not found after cache retry attempts", projectName)
	return nil, ErrNotFound
}

func getMapKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func safeUnixTime() uint64 {
	t := time.Now().Unix()
	if t < 0 {
		return 0
	}
	return uint64(t)
}