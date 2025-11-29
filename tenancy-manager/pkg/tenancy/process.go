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
	"fmt"
	"sync"
	"time"

	foldersv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/folder.edge-orchestrator.intel.com/v1"
	orgsv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/org.edge-orchestrator.intel.com/v1"
	orgactivewatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/orgactivewatcher.edge-orchestrator.intel.com/v1"
	projectv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/project.edge-orchestrator.intel.com/v1"
	projectactivewatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	runtimefoldersv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimefolder.edge-orchestrator.intel.com/v1"
	runtimeorgsv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimeorg.edge-orchestrator.intel.com/v1"
	runtimeprojectsv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimeproject.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	Testing bool

	// Per-org and per-project mutexes to prevent blocking all projects/orgs
	// when one is being updated. Maps hash names to their mutexes.
	orgMutexMap     = make(map[string]*sync.Mutex)
	projectMutexMap = make(map[string]*sync.Mutex)
	orgMutexLock    sync.Mutex
	projectMutexLock sync.Mutex
)

// getOrgMutex returns a mutex for a specific org, creating one if needed.
func getOrgMutex(hashName string) *sync.Mutex {
	orgMutexLock.Lock()
	defer orgMutexLock.Unlock()
	if _, exists := orgMutexMap[hashName]; !exists {
		orgMutexMap[hashName] = &sync.Mutex{}
	}
	return orgMutexMap[hashName]
}

// getProjectMutex returns a mutex for a specific project, creating one if needed.
func getProjectMutex(hashName string) *sync.Mutex {
	projectMutexLock.Lock()
	defer projectMutexLock.Unlock()
	if _, exists := projectMutexMap[hashName]; !exists {
		projectMutexMap[hashName] = &sync.Mutex{}
	}
	return projectMutexMap[hashName]
}

// Reconciler handles the reconciliation logic for the tenancy-datamodel API.
type Reconciler struct {
	Client *nexus_client.Clientset
	Config *config.Config
}

// NewReconciler creates a new instance of Reconciler to manage the tenancy-datamodel API reconciliation.
func NewReconciler(client *nexus_client.Clientset, cfg *config.Config) *Reconciler {
	return &Reconciler{
		Client: client,
		Config: cfg,
	}
}

// GetExpectedOrgWatchers gets the list of Org watchers that need to be notified.
func GetExpectedOrgWatchers(client *nexus_client.Clientset) (map[string]struct{}, error) {
	cfg, err := client.TenancyMultiTenancy().GetConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("fetching expectedOrgWatchers: failed to get config object with an error: %w", err)
	}

	// Build a map of watchers that need to be notified.
	orgWatchersIter := cfg.GetAllOrgWatchersIter(context.Background())
	c := context.Background()
	expectedWatchers := make(map[string]struct{})

	for {
		watcher, err := orgWatchersIter.Next(c)
		if err != nil {
			fmt.Printf("Error retrieving next watcher: %v", err)
			break
		}
		if watcher == nil {
			break
		}
		expectedWatchers[watcher.DisplayName()] = struct{}{}
	}
	return expectedWatchers, nil
}

// GetExpectedProjectWatchers gets the list of Project watchers that need to be notified.
func GetExpectedProjectWatchers(client *nexus_client.Clientset) (map[string]struct{}, error) {
	cfg, err := client.TenancyMultiTenancy().GetConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("fetching expectedProjectWatchers: failed to get config object with an error: %w", err)
	}

	// Build a map of watchers that need to be notified.
	projectWatchersIter := cfg.GetAllProjectWatchersIter(context.Background())
	c := context.Background()
	expectedWatchers := make(map[string]struct{})

	for {
		watcher, err := projectWatchersIter.Next(c)
		if err != nil {
			fmt.Printf("Error retrieving next watcher: %v", err)
			break
		}
		if watcher == nil {
			break
		}
		expectedWatchers[watcher.DisplayName()] = struct{}{}
	}
	return expectedWatchers, nil
}

// ProcessOrgsAdd is the callback function to be invoked when Org is added.
func (r *Reconciler) ProcessOrgsAdd(org *nexus_client.OrgOrg) {
	log.Debug().Msgf("Org %s (hashName: %s) created", org.DisplayName(), org.Name)

	/* The 'Testing' boolean is a temporary workaround to bypass the check below in unit tests.
	In unit tests, the object is created with a deletion timestamp.*/
	if !Testing {
		// If the process restarts, the org might be marked for deletion.
		// In that scenario, continue with the deletion of the org.
		if !org.DeletionTimestamp.IsZero() {
			log.Debug().Msgf("Org %v (hashName: %s) is marked for delete, processing delete",
				org.DisplayName(), org.Name)
			r.ProcessOrgsDelete(org)
			return
		}
	}

	if org.Status.OrgStatus.StatusIndicator == orgsv1.StatusIndicationIdle {
		// Org create already processed. Return.
		log.Debug().Msgf("Processing OrgAdd: Skip org creation of %v (hashName: %s) as it is already created",
			org.DisplayName(), org.Name)
		return
	}

	displayName, hashName := org.DisplayName(), org.Name

	// Create default Folder in the config tree of the datamodel.
	_, err := org.AddFolders(context.Background(), &foldersv1.Folder{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: foldersv1.FolderSpec{},
	})
	if err != nil && !nexus_client.IsAlreadyExists(err) {
		log.InfraErr(err).Msgf(`Creation of org %s (hashName: %s) failed,unable to add config default Folder`,
			org.DisplayName(), org.Name)
		setOrgStatus(r.Client, org.DisplayName(),
			org.Name,
			orgsv1.StatusIndicationError,
			fmt.Sprintf("Org creation failed: unable to add config default Folder, error: %v", err),
			Create)
		return
	}

	// Create an Org in the Runtime tree of the datamodel.
	runtimeOrg, err := r.Client.TenancyMultiTenancy().Runtime().
		AddOrgs(context.Background(), &runtimeorgsv1.RuntimeOrg{
			ObjectMeta: metav1.ObjectMeta{
				Name: org.DisplayName(),
			},
			Spec: runtimeorgsv1.RuntimeOrgSpec{},
		})
	if err != nil && !nexus_client.IsAlreadyExists(err) {
		log.InfraErr(err).Msgf(`Creation of org %s (hashName: %s) failed, unable to add runtime Org`, org.DisplayName(), org.Name)
		setOrgStatus(r.Client, org.DisplayName(),
			org.Name,
			orgsv1.StatusIndicationError,
			fmt.Sprintf("Org creation failed: unable to add runtime Org, error: %v", err),
			Create)
		return
	}

	if org.Status.OrgStatus.StatusIndicator == "" {
		// Set the Org status to InProgress and continue creation.
		setOrgStatus(r.Client, org.DisplayName(), org.Name,
			orgsv1.StatusIndicationInProgress,
			fmt.Sprintf("Org %v CREATE initiated", org.DisplayName()),
			Create,
		)
	}

	// Create a default Folder in the Runtime tree of the datamodel.
	_, err = runtimeOrg.AddFolders(context.Background(), &runtimefoldersv1.RuntimeFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: runtimefoldersv1.RuntimeFolderSpec{},
	})
	if err != nil && !nexus_client.IsAlreadyExists(err) {
		log.InfraErr(err).Msgf(`Creation of org %s (hashName: %s) failed, unable to add runtime default Folder`,
			org.DisplayName(), org.Name)
		setOrgStatus(r.Client, org.DisplayName(),
			org.Name,
			orgsv1.StatusIndicationError,
			fmt.Sprintf("Org creation failed: unable to add runtime default Folder, error: %v", err),
			Create)
		return
	}

	expectedOrgWatchers, err := GetExpectedOrgWatchers(r.Client)
	if err != nil {
		log.InfraErr(err).Msgf(`Creation of org %s (hashName: %s) failed, unable to fetch expectedOrgWatchers`,
			org.DisplayName(), org.Name)
		setOrgStatus(r.Client, org.DisplayName(),
			org.Name,
			orgsv1.StatusIndicationError,
			fmt.Sprintf("Org creation failed: unable to fetch expectedOrgWatchers, error: %v", err),
			Create)
		return
	}
	log.Debug().Msgf("Processing OrgAdd: Expected OrgWatchers: %#v", expectedOrgWatchers)

	// If no watchers are registered, then mark it to idle.
	if len(expectedOrgWatchers) == 0 {
		log.Debug().Msgf("Processing OrgAdd: Creation of org %s (hashName: %s) is successful, marking it as 'IDLE'",
			org.DisplayName(), org.Name)
		setOrgStatus(r.Client, org.DisplayName(),
			org.Name,
			orgsv1.StatusIndicationIdle,
			fmt.Sprintf("Org %v CREATE is complete", org.DisplayName()),
			Create)
		return
	}

	// Otherwise, set it to Inprogress and wait for watchers to acknowledge this org.
	log.Debug().Msgf("Processing OrgAdd: Waiting for watchers %v to acknowledge the org %s (hashName: %s)",
		getMapKeys(expectedOrgWatchers), org.DisplayName(), org.Name)
	setOrgStatus(r.Client, org.DisplayName(), org.Name,
		orgsv1.StatusIndicationInProgress,
		fmt.Sprintf("Waiting for watchers %v to acknowledge this org",
			getMapKeys(expectedOrgWatchers)),
		Create)

	go r.StartOrgCreateAcknowledgementTimer(time.Duration(r.Config.OrgCreateTimeoutInSecs)*time.Second,
		expectedOrgWatchers, displayName, hashName, runtimeOrg)
}

// ProcessOrgsUpdate is the callback function to be invoked when Org is updated.
func (r *Reconciler) ProcessOrgsUpdate(old, updated *nexus_client.OrgOrg) {
	// Skip those events that aren't delete events.
	if updated.DeletionTimestamp.IsZero() {
		return
	}

	/* The 'Testing' boolean is a temporary workaround to bypass the check below in unit tests.
	In unit tests, the object is created with a deletion timestamp.
	This is necessary because Nexus update handlers do not update the deletion timestamp;
	they only patch annotations, labels, and finalizers.
	The deletion timestamp is managed in real time by the kube-api-server.
	Hence, this boolean serves solely to bypass the check and allow for testing the code flow. */
	if !Testing {
		// Skip processing duplicate delete events.
		if !old.DeletionTimestamp.IsZero() {
			log.Debug().Msgf("Skipping processing of duplicate delete events of Org %s (hash name: %s)",
				updated.DisplayName(), updated.Name)
			return
		}
	}

	log.Debug().Msgf("Org %s (hashName: %s) deleted", updated.DisplayName(), updated.Name)

	r.ProcessOrgsDelete(updated)
}

// ProcessOrgsDelete is the function invoked when Org is deleted.
func (r *Reconciler) ProcessOrgsDelete(obj *nexus_client.OrgOrg) {
	displayName, hashName := obj.DisplayName(), obj.Name

	// Update the runtime org object to deleting.
	runtimeOrg, err := r.Client.TenancyMultiTenancy().Runtime().
		GetOrgs(context.Background(), obj.DisplayName())
	if err != nil {
		if nexus_client.IsNotFound(err) {
			// Runtime object does not exist for this Org. Just delete it.
			err := obj.Delete(context.Background())
			if err != nil && !nexus_client.IsNotFound(err) {
				log.InfraErr(err).Msgf("Failed to delete runtime Org object %s (hashName: %s)",
					obj.DisplayName(), obj.Name)
				return
			}
		} else {
			errMsg := fmt.Sprintf("Org %s (hashName: %s) deletion failed, unable to get runtime Org, error: %v",
				obj.DisplayName(), obj.Name, err)
			setOrgStatus(r.Client, obj.DisplayName(), obj.Name,
				orgsv1.StatusIndicationError,
				errMsg, Delete)
			log.Error().Msg(errMsg)
			return
		}
		return
	}

	runtimeOrg.Spec.Deleted = true
	defaultErr := runtimeOrg.Update(context.Background())
	if defaultErr != nil && !Testing {
		// SAFETY: 'Testing' bool prevents panic in UTs. In tests, handle errors gracefully without crashing.
		// In normal runtime, a panic is raised on update failure as it's critical.
		log.Panic().Msgf("failed update to runtime Org %s (hashName %s), with error: %v",
			obj.DisplayName(), obj.Name, defaultErr)
	}

	expectedOrgWatchers, err := GetExpectedOrgWatchers(r.Client)
	if err != nil {
		errMsg := fmt.Sprintf("Org deletion failed: unable to fetch expectedOrgWatchers, error: %v", err)
		setOrgStatus(r.Client, obj.DisplayName(), obj.Name,
			orgsv1.StatusIndicationError, errMsg, Delete)
		log.InfraErr(err).Msgf("Failed to delete runtime Org object %s (hashName: %s), unable to fetch expectedOrgWatchers",
			obj.DisplayName(), obj.Name)
		return
	}
	log.Debug().Msgf("Processing OrgDelete: Expected OrgWatchers of Org %s (hashName: %s): %#v",
		obj.DisplayName(), obj.Name, expectedOrgWatchers)

	// Get the list of watchers that have acknowledged the Org.
	currentActiveWatchers := make(map[string]struct{})
	activeWatchersIter := runtimeOrg.GetAllActiveWatchersIter(context.Background())
	foundActiveWatcher := false

	for {
		watcher, err := activeWatchersIter.Next(context.Background())
		if err != nil {
			log.InfraErr(err).Msgf("Error retrieving next watcher for runtimeOrg %s", obj.DisplayName())
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := expectedOrgWatchers[watcher.DisplayName()]; exists {
			// If at least one active watcher is found in expectedOrgWatchers, set the boolean to true.
			foundActiveWatcher = true
			currentActiveWatchers[watcher.DisplayName()] = struct{}{}
		}
	}

	if !foundActiveWatcher {
		// There are no active watchers.
		// The org is ready to be deleted.
		err = runtimeOrg.Delete(context.Background())
		if err != nil && !nexus_client.IsNotFound(err) {
			log.InfraErr(err).Msgf("Failed to delete runtime Org %s (hashName: %s)",
				runtimeOrg.DisplayName(), runtimeOrg.Name)
		}
		obj.SetFinalizers([]string{})
		err = obj.Update(context.Background())
		if err != nil && !nexus_client.IsNotFound(err) {
			log.InfraErr(err).Msgf("Failed to remove the finalizers from config Org %s (hashName: %s)",
				obj.DisplayName(), obj.Name)
		}
		return
	}

	// If there is at least one active watcher, don't mark for deletion.
	log.Debug().Msgf("At least one active watcher acknowledged the org %s (hashName: %s). No deletion necessary",
		runtimeOrg.DisplayName(), runtimeOrg.Name)

	// Set the Org status to InProgress and continue deletion.
	msg := fmt.Sprintf("Waiting for watchers %v to be deleted", getMapKeys(currentActiveWatchers))
	setOrgStatus(r.Client, obj.DisplayName(), obj.Name, orgsv1.StatusIndicationInProgress, msg, Delete)

	go r.StartOrgDeleteAcknowledgementTimer(time.Duration(r.Config.OrgCreateTimeoutInSecs)*time.Second,
		expectedOrgWatchers,
		displayName, hashName, runtimeOrg)
}

// ProcessOrgActiveWatcherAdd is the callback function to be invoked when active watcher has acknowledged an org creation request.
func (r *Reconciler) ProcessOrgActiveWatcherAdd(w *nexus_client.OrgactivewatcherOrgActiveWatcher) {
	log.Debug().Msgf("Orgs active watcher %v (hashName: %s) created", w.DisplayName(), w.Name)

	if err := processOrgActiveWatcher(r.Client, w); err != nil {
		log.InfraErr(err).Msgf("Processing OrgActiveWatcherAdd %s (hashName: %s) failed with an error: %v",
			w.DisplayName(), w.Name, err)
	}
}

// ProcessOrgActiveWatcherUpdate is the callback function to be invoked when active watcher is updated.
func (r *Reconciler) ProcessOrgActiveWatcherUpdate(old, updated *nexus_client.OrgactivewatcherOrgActiveWatcher) {
	log.Debug().Msgf("Org active watcher %v (hashName: %s) updated", updated.DisplayName(), updated.Name)

	if !hasOrgActiveWatcherSpecChanged(old.Spec, updated.Spec) {
		log.Debug().Msgf("Org active watcher %s (hashName: %s) spec has not changed, skip processing update event",
			updated.DisplayName(), updated.Name)
		return
	}

	// Process only when the active watcher is marked IDLE.
	if updated.Spec.StatusIndicator != orgactivewatcherv1.StatusIndicationIdle {
		log.Debug().Msgf("Org active watcher %s (hashName: %s) spec is not IDLE, skip processing update event",
			updated.DisplayName(), updated.Name)
		return
	}

	if err := processOrgActiveWatcher(r.Client, updated); err != nil {
		log.InfraErr(err).Msgf("Processing OrgActiveWatcherUpdate %s (hashName: %s) failed with an error: %v",
			updated.DisplayName(), updated.Name, err)
	}
}

// ProcessOrgActiveWatcherDelete is the callback function to be invoked when active watcher has stopped watching an org.
func (r *Reconciler) ProcessOrgActiveWatcherDelete(w *nexus_client.OrgactivewatcherOrgActiveWatcher) {
	log.Debug().Msgf("Orgs active watcher %s (hashName: %s) deleted", w.DisplayName(), w.Name)

	// Get the runtime obj associated with this active org watcher.
	runtimeOrg, err := w.GetParent(context.Background())
	if err != nil {
		log.InfraErr(err).Msgf("Processing OrgActiveWatcherDelete of %s (hashName: %s): failed to get runtime Org",
			w.DisplayName(), w.Name)
		return
	}

	if !runtimeOrg.Spec.Deleted {
		// A watcher got removed but the runtime is not marked for deletion. So dont have to react, as watchers can come and go.
		return
	}

	configOrg, err := r.Client.TenancyMultiTenancy().Config().
		GetOrgs(context.Background(), runtimeOrg.DisplayName())
	if err != nil {
		log.InfraErr(err).Msgf("Processing OrgActiveWatcherDelete of %s (hashName: %s): failed to get config Org",
			w.DisplayName(), w.Name)
		return
	}

	// Get all watchers registered to be notified.
	expectedOrgWatchers, err := GetExpectedOrgWatchers(r.Client)
	if err != nil {
		setOrgStatus(r.Client, configOrg.DisplayName(), configOrg.Name,
			orgsv1.StatusIndicationError,
			fmt.Sprintf("Failed to process OrgActiveWatcher delete, unable to fetch expectedOrgWatchers, error: %v",
				err),
			Delete)
		return
	}

	// Get the list of watchers that have acknowledged the Org.
	currentActiveWatchers := make(map[string]struct{})
	activeWatchersIter := runtimeOrg.GetAllActiveWatchersIter(context.Background())
	foundActiveWatcher := false

	for {
		watcher, err := activeWatchersIter.Next(context.Background())
		if err != nil {
			log.InfraErr(err).Msgf("Error retrieving next watcher for runtimeOrg %s", configOrg.DisplayName())
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := expectedOrgWatchers[watcher.DisplayName()]; exists {
			// If at least one active watcher is found in expectedOrgWatchers, set the boolean to true.
			foundActiveWatcher = true
			currentActiveWatchers[watcher.DisplayName()] = struct{}{}
		}
	}

	// If there is at least one active watcher, the org is not ready to be deleted.
	if foundActiveWatcher {
		msg := fmt.Sprintf("Waiting for watchers %v to be deleted", getMapKeys(currentActiveWatchers))
		setOrgStatus(r.Client, configOrg.DisplayName(), configOrg.Name, orgsv1.StatusIndicationInProgress, msg, Delete)
		log.Debug().Msgf("Processing OrgActiveWatcher delete: %v", msg)
		return
	}

	// There are no active watchers.
	// The org is ready to be deleted.
	configOrg.SetFinalizers([]string{})
	err = configOrg.Update(context.Background())
	if err != nil && !nexus_client.IsNotFound(err) {
		log.InfraErr(err).Msgf("Failed to remove the finalizers of config Org %s (hashName %s)",
			configOrg.DisplayName(), configOrg.Name)
	}
	err = runtimeOrg.Delete(context.Background())
	if err != nil && !nexus_client.IsNotFound(err) {
		log.InfraErr(err).Msgf("Failed to delete runtime Org %s (hashName %s)",
			runtimeOrg.DisplayName(), runtimeOrg.Name)
	}
}

// ProcessProjectsAdd is callback function to be invoked when Project is added.
func (r *Reconciler) ProcessProjectsAdd(project *nexus_client.ProjectProject) {
	log.Debug().Msgf("Project %s (hashName: %s) created", project.DisplayName(), project.Name)

	/* The 'Testing' boolean is a temporary workaround to bypass the check below in unit tests.
	In unit tests, the object is created with a deletion timestamp.*/
	if !Testing {
		// If the process restarts, the org might be marked for deletion.
		// In that scenario, continue with the deletion of the org.
		if !project.DeletionTimestamp.IsZero() {
			log.Debug().Msgf("Project %s (hashName: %s) is marked for delete, processing delete",
				project.DisplayName(), project.Name)
			r.ProcessProjectsDelete(project)
			return
		}
	}

	if project.Status.ProjectStatus.StatusIndicator == projectv1.StatusIndicationIdle {
		// Project create already processed. Return.
		log.Debug().Msgf("Skip project creation of %s (hashName: %s) as it is already created",
			project.DisplayName(), project.Name)
		return
	}

	displayName, hashName := project.DisplayName(), project.Name

	// Derive Org and Folder name from labels.
	parentOrgName := project.GetLabels()["orgs.org.edge-orchestrator.intel.com"]
	parentFolderName := project.GetLabels()["folders.folder.edge-orchestrator.intel.com"]

	runtimeProject, err := r.Client.TenancyMultiTenancy().Runtime().
		Orgs(parentOrgName).Folders(parentFolderName).
		AddProjects(context.Background(), &runtimeprojectsv1.RuntimeProject{
			ObjectMeta: metav1.ObjectMeta{
				Name: project.DisplayName(),
			},
			Spec: runtimeprojectsv1.RuntimeProjectSpec{},
		})
	if err != nil && !nexus_client.IsAlreadyExists(err) {
		log.InfraErr(err).Msgf("Project creation for config Project %s (hashName: %s) failed: "+
			"unable to add runtime Project", project.DisplayName(), project.Name)
		setProjectStatus(r.Client, project.DisplayName(),
			project.Name, parentOrgName, parentFolderName,
			projectv1.StatusIndicationError,
			fmt.Sprintf("Project creation failed: unable to add runtime Project, error: %v", err),
			Create)
		return
	}

	if project.Status.ProjectStatus.StatusIndicator == "" {
		// Set the Project status to InProgress and continue creation.
		setProjectStatus(r.Client, project.DisplayName(),
			project.Name, parentOrgName, parentFolderName,
			projectv1.StatusIndicationInProgress,
			fmt.Sprintf("Project %v CREATE initiated", project.DisplayName()),
			Create)
	}

	expectedProjectWatchers, err := GetExpectedProjectWatchers(r.Client)
	if err != nil {
		log.InfraErr(err).Msgf("Project creation for config Project %s (hashName: %s) failed: "+
			"unable to fetch expectedProjectWatchers", project.DisplayName(), project.Name)

		setProjectStatus(r.Client, project.DisplayName(),
			project.Name, parentOrgName, parentFolderName,
			projectv1.StatusIndicationError,
			fmt.Sprintf("Project creation failed: unable to fetch expectedProjectWatchers, error: %v", err),
			Create)
		return
	}

	// If no watchers are registered, then mark it to idle.
	if len(expectedProjectWatchers) == 0 {
		log.Debug().Msgf("Creation of project %s (hashName: %s) is successful, marking it as 'IDLE'",
			project.DisplayName(), project.Name)
		setProjectStatus(r.Client, project.DisplayName(),
			project.Name, parentOrgName, parentFolderName,
			projectv1.StatusIndicationIdle,
			fmt.Sprintf("Project %v CREATE is complete", project.DisplayName()),
			Create)
		return
	}

	// Otherwise, set it to Inprogress and wait for watchers to acknowledge this project.
	log.Debug().Msgf("Waiting for watchers %v to acknowledge the project %s (hashName: %s)",
		getMapKeys(expectedProjectWatchers), project.DisplayName(), project.Name)
	setProjectStatus(r.Client, project.DisplayName(), project.Name,
		parentOrgName, parentFolderName, projectv1.StatusIndicationInProgress,
		fmt.Sprintf("Waiting for watchers %v to acknowledge this project",
			getMapKeys(expectedProjectWatchers)),
		Create)

	go r.StartProjectCreateAcknowledgementTimer(time.Duration(r.Config.ProjectCreateTimeoutInSecs)*time.Second,
		expectedProjectWatchers,
		displayName, hashName, parentOrgName, parentFolderName, runtimeProject)
}

// ProcessProjectsUpdate is callback function to be invoked when Project is updated.
func (r *Reconciler) ProcessProjectsUpdate(old, updated *nexus_client.ProjectProject) {
	// Skip those events that aren't delete events.
	if updated.DeletionTimestamp.IsZero() {
		return
	}

	/* The 'Testing' boolean is a temporary workaround to bypass the check below in unit tests.
	In unit tests, the object is created with a deletion timestamp.
	This is necessary because Nexus update handlers do not update the deletion timestamp;
	they only patch annotations, labels, and finalizers.
	The deletion timestamp is managed in real time by the kube-api-server.
	Hence, this boolean serves solely to bypass the check and allow for testing the code flow. */
	if !Testing {
		// Skip processing duplicate delete events.
		if !old.DeletionTimestamp.IsZero() {
			log.Debug().Msgf("Skipping processing of duplicate delete events of Project %s (hash name: %s)",
				updated.DisplayName(), updated.Name)
			return
		}
	}

	log.Debug().Msgf("Project %s (hashName: %s) deleted", updated.DisplayName(), updated.Name)

	r.ProcessProjectsDelete(updated)
}

// ProcessProjectsDelete is the function invoked when Project is deleted.
func (r *Reconciler) ProcessProjectsDelete(obj *nexus_client.ProjectProject) {
	displayName, hashName := obj.DisplayName(), obj.Name

	// Derive org and folder name from labels.
	parentOrgName := obj.GetLabels()["orgs.org.edge-orchestrator.intel.com"]
	parentFolderName := obj.GetLabels()["folders.folder.edge-orchestrator.intel.com"]

	// Update the runtime project object to deleting.
	runtimeProject, err := r.Client.TenancyMultiTenancy().Runtime().
		Orgs(parentOrgName).Folders(parentFolderName).
		GetProjects(context.Background(), obj.DisplayName())
	if err != nil {
		if nexus_client.IsNotFound(err) {
			// Runtime object does not exist for this Project. Just delete it.
			err = obj.Delete(context.Background())
			if err != nil {
				log.InfraErr(err).Msgf("Failed to delete config Project %s (hashName: %s)",
					obj.DisplayName(), obj.Name)
				return
			}
		} else {
			errMsg := fmt.Sprintf("Project deletion failed, unable to get runtime Project, error: %v", err)
			setProjectStatus(r.Client, obj.DisplayName(),
				obj.Name, parentOrgName, parentFolderName,
				projectv1.StatusIndicationError,
				errMsg,
				Delete)
			log.Error().Msg(errMsg)
		}
		return
	}

	runtimeProject.Spec.Deleted = true
	err = runtimeProject.Update(context.Background())
	if err != nil && !Testing {
		// SAFETY: 'Testing' bool prevents panic in UTs. In tests, handle errors gracefully without crashing.
		// In normal runtime, a panic is raised on update failure as it's critical.
		log.Panic().Msgf("failed update to runtime Project %s (hashName: %s), error: %v",
			runtimeProject.DisplayName(), runtimeProject.Name, err)
	}

	expectedProjectWatchers, err := GetExpectedProjectWatchers(r.Client)
	if err != nil {
		errMsg := fmt.Sprintf("Project deletion failed: unable to fetch expectedProjectWatchers, error: %v", err)
		setProjectStatus(r.Client, obj.DisplayName(),
			obj.Name, parentOrgName, parentFolderName,
			projectv1.StatusIndicationError, errMsg,
			Delete)
		log.Error().Msg(errMsg)
		return
	}

	// Get the list of watchers that have acknowledged the Project.
	currentActiveWatchers := make(map[string]struct{})
	activeWatchersIter := runtimeProject.GetAllActiveWatchersIter(context.Background())
	foundActiveWatcher := false

	for {
		watcher, err := activeWatchersIter.Next(context.Background())
		if err != nil {
			log.InfraErr(err).Msgf("Error retrieving next watcher for runtimeProject %s",
				runtimeProject.DisplayName())
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := expectedProjectWatchers[watcher.DisplayName()]; exists {
			// If at least one active watcher is found in expectedProjectWatchers, set the boolean to true.
			foundActiveWatcher = true
			currentActiveWatchers[watcher.DisplayName()] = struct{}{}
		}
	}

	if !foundActiveWatcher {
		// There are no active watchers.
		// The project is ready to be deleted.
		err = runtimeProject.Delete(context.Background())
		if err != nil && !nexus_client.IsNotFound(err) {
			log.InfraErr(err).Msgf("Failed to delete runtime Project %s (hashName: %s)",
				runtimeProject.DisplayName(), runtimeProject.Name)
		}
		obj.SetFinalizers([]string{})
		err = obj.Update(context.Background())
		if err != nil && !nexus_client.IsNotFound(err) {
			log.InfraErr(err).Msgf("Failed to remove the finalizers of config Project %s (hashName: %s)",
				obj.DisplayName(), obj.Name)
		}
		return
	}

	// If there is at least one active watcher, don't mark for deletion.
	log.Debug().Msgf("At least one active watcher acknowledged the project %s. No deletion necessary",
		runtimeProject.DisplayName())

	// Set the Project status to InProgress and continue deletion.
	msg := fmt.Sprintf("Waiting for watchers %v to be deleted", getMapKeys(currentActiveWatchers))
	setProjectStatus(r.Client, obj.DisplayName(),
		obj.Name, parentOrgName, parentFolderName,
		projectv1.StatusIndicationInProgress, msg, Delete)

	go r.StartProjectDeleteAcknowledgementTimer(time.Duration(r.Config.ProjectDeleteTimeoutInSecs)*time.Second,
		expectedProjectWatchers,
		displayName, hashName, parentOrgName, parentFolderName, runtimeProject)
}

// ProcessProjectActiveWatcherAdd is the callback function to be invoked,
// when active watcher has acknowledged a project creation request.
func (r *Reconciler) ProcessProjectActiveWatcherAdd(w *nexus_client.ProjectactivewatcherProjectActiveWatcher) {
	log.Debug().Msgf("Projects active watcher %s (hashName: %s) created", w.DisplayName(), w.Name)

	if err := processProjectActiveWatcher(r.Client, w); err != nil {
		log.InfraErr(err).Msgf("Processing ProjectActiveWatcherAdd %s (hashName: %s) failed with an error: %v",
			w.DisplayName(), w.Name, err)
	}
}

// ProcessProjectActiveWatcherUpdate is the callback function to be invoked when active watcher is updated.
func (r *Reconciler) ProcessProjectActiveWatcherUpdate(old, updated *nexus_client.ProjectactivewatcherProjectActiveWatcher) {
	if !hasProjectActiveWatcherSpecChanged(old.Spec, updated.Spec) {
		log.Debug().Msgf("Project active watcher %s (hashName: %s) spec has not changed, skip processing update event",
			updated.DisplayName(), updated.Name)
		return
	}

	// Process only when the active watcher is marked IDLE.
	if updated.Spec.StatusIndicator != projectactivewatcherv1.StatusIndicationIdle {
		log.Debug().Msgf("Project active watcher %s (hashName: %s) spec is not IDLE, skip processing update event",
			updated.DisplayName(), updated.Name)
		return
	}

	if err := processProjectActiveWatcher(r.Client, updated); err != nil {
		log.InfraErr(err).Msgf("Processing ProjectActiveWatcherUpdate %s (hashName: %s) failed with an error: %v",
			updated.DisplayName(), updated.Name, err)
	}
}

// ProcessProjectActiveWatcherDelete is the callback function to be invoked when active watcher has stopped watching a project.
func (r *Reconciler) ProcessProjectActiveWatcherDelete(w *nexus_client.ProjectactivewatcherProjectActiveWatcher) {
	log.Debug().Msgf("Project active watcher %s (hashName: %s) deleted", w.DisplayName(), w.Name)

	// Get the runtime project associated with this active project watcher.
	runtimeProject, err := w.GetParent(context.Background())
	if err != nil {
		log.InfraErr(err).Msgf("Processing ProjectActiveWatcherDelete of %s (hashName: %s): failed to get runtime Project",
			w.DisplayName(), w.Name)
		return
	}

	if !runtimeProject.Spec.Deleted {
		// A watcher got removed but the runtime is not marked for deletion. So dont have to react, as watchers can come and go.
		return
	}

	// Derive Org and Folder name from labels.
	parentOrgName := runtimeProject.GetLabels()["runtimeorgs.runtimeorg.edge-orchestrator.intel.com"]
	parentFolerName := runtimeProject.GetLabels()["runtimefolders.runtimefolder.edge-orchestrator.intel.com"]

	configProject, err := r.Client.TenancyMultiTenancy().Config().
		Orgs(parentOrgName).Folders(parentFolerName).
		GetProjects(context.Background(), runtimeProject.DisplayName())
	if err != nil {
		log.InfraErr(err).Msgf("Processing ProjectActiveWatcherDelete of %s (hashName: %s): failed to get config Project",
			w.DisplayName(), w.Name)
		return
	}

	expectedProjectWatchers, err := GetExpectedProjectWatchers(r.Client)
	if err != nil {
		log.InfraErr(err).Msgf("Processing ProjectActiveWatcherDelete of %s (hashName: %s): "+
			"unable to fetch expectedProjectWatchers", w.DisplayName(), w.Name)
		setProjectStatus(r.Client, configProject.DisplayName(), configProject.Name, parentOrgName, parentFolerName,
			projectv1.StatusIndicationError,
			fmt.Sprintf("Failed to process ProjectActiveWatcher delete, unable to fetch expectedProjectWatchers, "+
				"error: %v", err),
			Delete)
		return
	}

	// Get the list of watchers that have acknowledged the Project.
	currentActiveWatchers := make(map[string]struct{})
	activeWatchersIter := runtimeProject.GetAllActiveWatchersIter(context.Background())
	foundActiveWatcher := false

	for {
		watcher, err := activeWatchersIter.Next(context.Background())
		if err != nil {
			log.InfraErr(err).Msgf("Error retrieving next watcher for runtimeProject %s",
				configProject.DisplayName())
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := expectedProjectWatchers[watcher.DisplayName()]; exists {
			// If at least one active watcher is found in expectedProjectWatchers, set the boolean to true.
			foundActiveWatcher = true
			currentActiveWatchers[watcher.DisplayName()] = struct{}{}
		}
	}

	// If there is at least one active watcher, the project is not ready to be deleted.
	if foundActiveWatcher {
		msg := fmt.Sprintf("Waiting for watchers %v to be deleted", getMapKeys(currentActiveWatchers))
		setProjectStatus(r.Client, configProject.DisplayName(),
			configProject.Name, parentOrgName, parentFolerName,
			projectv1.StatusIndicationInProgress,
			msg, Delete)
		log.Debug().Msg(msg)
		return
	}

	// There are no active watchers.
	// The project is ready to be deleted.
	configProject.SetFinalizers([]string{})
	err = configProject.Update(context.Background())
	if err != nil && !nexus_client.IsNotFound(err) {
		log.InfraErr(err).Msgf("Failed to remove finalizers of config Project %s (hashName: %s)",
			configProject.DisplayName(), configProject.Name)
	}
	err = runtimeProject.Delete(context.Background())
	if err != nil && !nexus_client.IsNotFound(err) {
		log.InfraErr(err).Msgf("Failed to delete runtime Project %s (hashName: %s)",
			runtimeProject.DisplayName(), runtimeProject.Name)
	}
}

// processOrgActiveWatcher processes OrgActiveWatcher's add and update events.
func processOrgActiveWatcher(client *nexus_client.Clientset, obj *nexus_client.OrgactivewatcherOrgActiveWatcher) error {
	// Get the runtime Org associated with this active org watcher.
	runtimeorg, err := obj.GetParent(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get runtime Org with error %w", err)
	}

	configOrg, err := client.TenancyMultiTenancy().Config().
		GetOrgs(context.Background(), runtimeorg.DisplayName())
	if err != nil {
		return fmt.Errorf("failed to get config Org with error %w", err)
	}

	if !Testing {
		if !configOrg.DeletionTimestamp.IsZero() {
			log.Debug().Msgf("Org %v (hashName: %s) is marked for delete, skip processing OrgActiveWatcher Add/Update events",
				configOrg.DisplayName(), configOrg.Name)
			return nil
		}
	}

	// Check if the config object has already been marked as idle.
	// In which case, this is an duplicate update and essentially NO-OP.
	if configOrg.Status.OrgStatus.StatusIndicator == orgsv1.StatusIndicationIdle {
		log.Debug().Msgf("Skipping processing OrgActiveWatcher %s (hashName: %s), as org %s (hashName: %s) is set to IDLE",
			obj.DisplayName(), obj.Name, configOrg.DisplayName(), configOrg.Name)
		return nil
	}

	// Get all watchers registered to be notified.
	expectedOrgWatchers, err := GetExpectedOrgWatchers(client)
	if err != nil {
		return fmt.Errorf("failed to process OrgActiveWatcher, unable to fetch expectedOrgWatchers, %w", err)
	}

	log.Debug().Msgf("Org active watcher %v (hashName: %s), expectedOrgWatchers: %#v",
		obj.DisplayName(), obj.Name, expectedOrgWatchers)

	// Get the list of watchers that have acknowledged the Org.
	activeWatchersIter := runtimeorg.GetAllActiveWatchersIter(context.Background())
	c := context.Background()

	for {
		watcher, err := activeWatchersIter.Next(c)
		if err != nil {
			log.InfraErr(err).Msgf("Error retrieving next watcher for runtimeOrg %s: %v", configOrg.DisplayName(), err)
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := expectedOrgWatchers[watcher.DisplayName()]; !exists {
			// Watcher is not in the expected list, skip it
			continue
		}
		if watcher.Spec.StatusIndicator != orgactivewatcherv1.StatusIndicationIdle {
			log.Debug().Msgf("Org active watcher %v (hashName: %s) is not IDLE",
				watcher.DisplayName(), watcher.Name)
			continue
		}
		// Watcher is in the expected list and is IDLE, so remove it from the map
		delete(expectedOrgWatchers, watcher.DisplayName())
	}

	// If there is delta, then we will need to wait for additional watchers to acknowledge this org.
	if len(expectedOrgWatchers) > 0 {
		msg := fmt.Sprintf("Waiting for watchers %v to acknowledge org %s",
			getMapKeys(expectedOrgWatchers), configOrg.DisplayName())
		setOrgStatus(client, configOrg.DisplayName(), configOrg.Name,
			orgsv1.StatusIndicationInProgress,
			msg,
			Create)
		log.Debug().Msgf("Processing OrgActiveWatcher: %v", msg)
		return nil
	}

	log.Debug().Msgf("Org active watchers are all IDLE, setting Org %s (hashName: %s) to IDLE",
		configOrg.DisplayName(), configOrg.Name)

	// All watchers have acknowledged. Mark the org as created.
	setOrgStatus(client, configOrg.DisplayName(), configOrg.Name,
		orgsv1.StatusIndicationIdle,
		fmt.Sprintf("Org %v CREATE is complete", configOrg.DisplayName()),
		Create)
	return nil
}

// processProjectActiveWatcher processes ProjectActiveWatcher's add and update events.
func processProjectActiveWatcher(client *nexus_client.Clientset,
	watcher *nexus_client.ProjectactivewatcherProjectActiveWatcher,
) error {
	// Get the runtime obj associated with this active project watcher.
	runtimeProject, err := watcher.GetParent(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get runtime Project with error: %w", err)
	}

	// Derive Org and Folder name from labels.
	parentOrgName := runtimeProject.GetLabels()["runtimeorgs.runtimeorg.edge-orchestrator.intel.com"]
	parentFolerName := runtimeProject.GetLabels()["runtimefolders.runtimefolder.edge-orchestrator.intel.com"]

	configProject, err := client.TenancyMultiTenancy().
		Config().Orgs(parentOrgName).Folders(parentFolerName).
		GetProjects(context.Background(), runtimeProject.DisplayName())
	if err != nil {
		return fmt.Errorf("failed to get config Project with error: %w", err)
	}

	if !Testing {
		if !configProject.DeletionTimestamp.IsZero() {
			log.Debug().Msgf(`Project %v (hashName: %s) is marked for delete, skip processing ProjectActiveWatcher`+
				`Add/Update events`, configProject.DisplayName(), configProject.Name)
			return nil
		}
	}

	// Check if the config object has already marked as created.
	// In which case, this is an duplicate update and essentially NO-OP.
	if configProject.Status.ProjectStatus.StatusIndicator == projectv1.StatusIndicationIdle {
		log.Debug().Msgf("Skipping processing ProjectActiveWatcher, as project %s is already marked IDLE",
			configProject.DisplayName())
		return nil
	}

	// Get all watchers registered to be notified.
	expectedProjectWatchers, err := GetExpectedProjectWatchers(client)
	if err != nil {
		return fmt.Errorf(`failed to process ProjectActiveWatcher,
			unable to fetch expectedProjectWatchers, %w`, err)
	}

	log.Debug().Msgf("Project active watcher %v (hashName: %s), expectedProjectWatchers: %#v",
		watcher.DisplayName(), watcher.Name, expectedProjectWatchers)

	// Get the list of watchers that have acknowledged the Project.
	activeWatchersIter := runtimeProject.GetAllActiveWatchersIter(context.Background())
	c := context.Background()

	for {
		watcher, err := activeWatchersIter.Next(c)
		if err != nil {
			log.InfraErr(err).Msgf("Error retrieving next watcher for runtimeProject %s: %v",
				configProject.DisplayName(), err)
			break
		}
		if watcher == nil {
			break
		}
		if _, exists := expectedProjectWatchers[watcher.DisplayName()]; !exists {
			// Watcher is not in the expected list, skip it
			continue
		}
		if watcher.Spec.StatusIndicator != projectactivewatcherv1.StatusIndicationIdle {
			log.Debug().Msgf("Project active watcher %v (hashName: %s) is not IDLE",
				watcher.DisplayName(), watcher.Name)
			continue
		}
		// Watcher is in the expected list and is IDLE, so remove it from the map
		delete(expectedProjectWatchers, watcher.DisplayName())
	}

	// If there is delta, then we will need to wait for additional watchers to acknowledge this project.
	if len(expectedProjectWatchers) > 0 {
		msg := fmt.Sprintf("Waiting for watchers %v to acknowledge project %s",
			getMapKeys(expectedProjectWatchers), configProject.DisplayName())
		setProjectStatus(client, configProject.DisplayName(),
			configProject.Name, parentOrgName, parentFolerName,
			projectv1.StatusIndicationInProgress,
			msg, Create)
		log.Debug().Msgf("Processing ProjectActiveWatcher: %v", msg)
		return nil
	}

	log.Debug().Msgf("Project active watchers are all IDLE, setting Project %s (hashName: %s) to IDLE",
		configProject.DisplayName(), configProject.Name)

	// All watchers have acknowledged. Mark the project as created.
	setProjectStatus(client, configProject.DisplayName(),
		configProject.Name, parentOrgName, parentFolerName,
		projectv1.StatusIndicationIdle,
		fmt.Sprintf("Project %v CREATE is complete", configProject.DisplayName()),
		Create)
	return nil
}
