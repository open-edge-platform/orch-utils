// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/cache"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/common"
	baseconfigtenancycomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/config.edge-orchestrator.intel.com/v1"
	baseruntimetenancycomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtime.edge-orchestrator.intel.com/v1"
	basetenancytenancycomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/tenancy.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const smallCaseStringDefault = "default"

var (
	appName = "nexus-api-gw-reconciler"
	log     = logging.GetLogger(appName)
)

type TenancyDM struct {
	nexusClient   *nexus_client.Clientset
	statusMutex   *sync.Mutex
	reconcileTime time.Duration
}

func NewTenancyManager(tenancyNC *nexus_client.Clientset, reconcileTime time.Duration) *TenancyDM {
	e := &TenancyDM{
		tenancyNC,
		&sync.Mutex{},
		reconcileTime,
	}
	return e
}

func (tdm *TenancyDM) TenancyDmInit() {
	ctx := context.Background()

	mObj, err := tdm.ensureMultiTenancyObject(ctx)
	if err != nil {
		log.Fatal().Msgf("Failed to ensure MultiTenancy object: %v", err)
	}

	if err := tdm.ensureConfigObject(ctx, mObj); err != nil {
		log.Fatal().Msgf("Failed to ensure Config object: %v", err)
	}

	if err := tdm.ensureRuntimeObject(ctx, mObj); err != nil {
		log.Fatal().Msgf("Failed to ensure Runtime object: %v", err)
	}
}

func (tdm *TenancyDM) ensureMultiTenancyObject(ctx context.Context) (*nexus_client.TenancyMultiTenancy, error) {
	mObj, err := tdm.nexusClient.GetTenancyMultiTenancy(ctx)
	if apierrors.IsNotFound(err) {
		log.Debug().Msgf("Adding MultiTenancy Object: %v", err)
		obj := &basetenancytenancycomv1.MultiTenancy{}
		obj.Name = smallCaseStringDefault
		if testing.Testing() {
			obj.ResourceVersion = "123"
		}

		mObj, err = tdm.nexusClient.AddTenancyMultiTenancy(ctx, obj)
		if err != nil && !nexus_client.IsAlreadyExists(err) {
			log.Error().Msgf("failed to add MultiTenancy: %v", err)
			return nil, fmt.Errorf("failed to add MultiTenancy: %w", err)
		}
	}
	return mObj, nil
}

func (tdm *TenancyDM) ensureConfigObject(ctx context.Context, mObj *nexus_client.TenancyMultiTenancy) error {
	var cnfErr nexus_client.ChildNotFoundError
	_, err := mObj.GetConfig(ctx)
	if errors.As(err, &cnfErr) {
		cObj := &baseconfigtenancycomv1.Config{}
		cObj.Name = "default"
		if testing.Testing() {
			cObj.ResourceVersion = "234"
		}
		cfgObj, err := mObj.AddConfig(ctx, cObj)
		if err != nil && cfgObj == nil && !nexus_client.IsAlreadyExists(err) {
			log.Error().Msgf("failed to add Config: %v", err)
			return fmt.Errorf("failed to add Config: %w", err)
		}
	}
	return nil
}

func (tdm *TenancyDM) ensureRuntimeObject(ctx context.Context, mObj *nexus_client.TenancyMultiTenancy) error {
	var cnfErr nexus_client.ChildNotFoundError
	_, err := mObj.GetRuntime(ctx)
	if errors.As(err, &cnfErr) {
		rObj := &baseruntimetenancycomv1.Runtime{}
		rObj.Name = "default"
		if testing.Testing() {
			rObj.ResourceVersion = "456"
		}
		runtimeObj, err := mObj.AddRuntime(ctx, rObj)
		if err != nil && runtimeObj == nil && !nexus_client.IsAlreadyExists(err) {
			log.Error().Msgf("failed to add Runtime: %v", err)
			return fmt.Errorf("failed to add Runtime: %w", err)
		}
	}
	return nil
}

func (tdm *TenancyDM) PeriodicReconciler(ctx context.Context) {
	iam, err := tdm.nexusClient.GetTenancyMultiTenancy(context.Background())
	if err != nil {
		log.InfraErr(err).Msg("Error while creating looking up Iam object")
		return
	}

	runtime, err := iam.GetRuntime(context.Background())
	if err != nil {
		log.InfraErr(err).Msg("Error while creating looking up Iam runtime object")
		return
	}

	rOrgItr := runtime.GetAllOrgsIter(ctx)

	for rOrg, e := rOrgItr.Next(ctx); e == nil && rOrg != nil; rOrg, e = rOrgItr.Next(ctx) {
		var org common.Org
		org.Name = rOrg.DisplayName()
		org.UID = string(rOrg.UID)
		org.Deleted = rOrg.Spec.Deleted
		cache.GlobalOrgCache.Set(string(rOrg.UID), org)

		folderOrgs := rOrg.GetAllFoldersIter(ctx)
		for fOrg, e := folderOrgs.Next(ctx); e == nil && fOrg != nil; fOrg, e = folderOrgs.Next(ctx) {
			pOrgItr := fOrg.GetAllProjectsIter(ctx)

			for pObj, e := pOrgItr.Next(ctx); e == nil && pObj != nil; pObj, e = pOrgItr.Next(ctx) {
				var proj common.Project
				proj.Name = pObj.DisplayName()
				proj.UID = string(pObj.UID)
				proj.Deleted = pObj.Spec.Deleted
				proj.Org.Name = rOrg.DisplayName()
				proj.Org.UID = string(rOrg.UID)
				proj.Org.Deleted = rOrg.Spec.Deleted
				projKey := fmt.Sprintf("%s_%s", rOrg.UID, pObj.DisplayName())
				cache.GlobalProjectCache.Set(projKey, proj)
			}
		}
	}
}

func (tdm *TenancyDM) createProjectFromRuntimeProject(project *nexus_client.RuntimeprojectRuntimeProject,
) (*common.Project, error) {
	log.Info().Msgf("Processing project with UID: %s", project.UID)

	folderOrgs, err := project.GetParent(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error while looking up Iam runtime object: %w", err)
	}

	runtimeOrg, err := folderOrgs.GetParent(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error while looking up Iam runtime org object: %w", err)
	}

	proj := &common.Project{
		Name:    project.DisplayName(),
		UID:     string(project.UID),
		Deleted: project.Spec.Deleted,
		Org: common.Org{
			Name:    runtimeOrg.DisplayName(),
			UID:     string(runtimeOrg.UID),
			Deleted: runtimeOrg.Spec.Deleted,
		},
	}

	return proj, nil
}

func (tdm *TenancyDM) AddProjectToCache(project *nexus_client.RuntimeprojectRuntimeProject) {
	log.Info().Msgf("AddProjectToCache Called from for %s", project.UID)

	proj, err := tdm.createProjectFromRuntimeProject(project)
	if err != nil {
		log.InfraErr(err).Msg("Error while creating createProjectFromRuntimeProject")
		return
	}

	projKey := fmt.Sprintf("%s_%s", proj.Org.UID, proj.Name)
	cache.GlobalProjectCache.Set(projKey, *proj)
}

func (tdm *TenancyDM) UpdateProjectFromCache(_, newObj *nexus_client.RuntimeprojectRuntimeProject) {
	log.Info().Msgf("UpdateProjectFromCache triggerd for %s", newObj.DisplayName())

	proj, err := tdm.createProjectFromRuntimeProject(newObj)
	if err != nil {
		log.InfraErr(err).Msg("Error while creating createProjectFromRuntimeProject")
		return
	}

	if value, ok := cache.GlobalProjectCache.Get(string(newObj.UID)); ok {
		value.Deleted = proj.Deleted
		value.Org.Deleted = proj.Org.Deleted
		cache.GlobalProjectCache.Set(value.UID, value)
	}
}

func (tdm *TenancyDM) DeleteProjectFromCache(project *nexus_client.RuntimeprojectRuntimeProject) {
	log.Info().Msgf("DeleteProjectFromCache triggerd for %s", project.UID)
	folderOrgs, err := project.GetParent(context.Background())
	if err != nil {
		log.InfraErr(err).Msg("Error while creating looking up Iam runtime object")
		return
	}

	runtimeOrg, err := folderOrgs.GetParent(context.Background())
	if err != nil {
		log.InfraErr(err).Msg("Error while looking up Iam runtime org object")
		return
	}
	projKey := fmt.Sprintf("%s_%s", runtimeOrg.UID, project.DisplayName())

	cache.GlobalProjectCache.Delete(projKey)
}

func (tdm *TenancyDM) ADDOrgToCache(orgObj *nexus_client.RuntimeorgRuntimeOrg) {
	log.Info().Msgf("ADDOrgToCache triggerd for %s", orgObj.Name)

	var org common.Org
	org.Name = orgObj.DisplayName()
	org.UID = string(orgObj.UID)
	org.Deleted = orgObj.Spec.Deleted
	cache.GlobalOrgCache.Set(string(orgObj.UID), org)
}

func (tdm *TenancyDM) UpdateOrgFromCache(_, newObj *nexus_client.RuntimeorgRuntimeOrg) {
	log.Info().Msgf("UpdateOrgFromCache triggered for %s", newObj.DisplayName())

	if value, ok := cache.GlobalOrgCache.Get(string(newObj.UID)); ok {
		value.Deleted = newObj.Spec.Deleted
		cache.GlobalOrgCache.Set(value.UID, value)
	}
}

func (tdm *TenancyDM) DeleteOrgFromCache(orgObj *nexus_client.RuntimeorgRuntimeOrg) {
	log.Info().Msgf("DeleteOrgFromCache triggered for %s", orgObj.DisplayName())
	cache.GlobalOrgCache.Delete(string(orgObj.UID))
}

func (tdm *TenancyDM) registerProjectCallbacks() error {
	if _, err := tdm.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").
		RegisterAddCallback(tdm.AddProjectToCache); err != nil {
		log.InfraErr(err).Msg("Failed to RegisterAddCallback for Project")
		return err
	}

	if _, err := tdm.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").
		RegisterUpdateCallback(tdm.UpdateProjectFromCache); err != nil {
		log.InfraErr(err).Msg("Failed to RegisterUpdateCallback for Project")
		return err
	}

	if _, err := tdm.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").
		RegisterDeleteCallback(tdm.DeleteProjectFromCache); err != nil {
		log.InfraErr(err).Msg("Failed to RegisterDeleteCallback for Project")
		return err
	}
	return nil
}

func (tdm *TenancyDM) registerOrgCallbacks() error {
	if _, err := tdm.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").
		RegisterAddCallback(tdm.ADDOrgToCache); err != nil {
		log.InfraErr(err).Msg("Failed to RegisterAddCallback for Project")
		return err
	}
	if _, err := tdm.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").
		RegisterUpdateCallback(tdm.UpdateOrgFromCache); err != nil {
		log.InfraErr(err).Msg("Failed to RegisterUpdateCallback for Project")
		return err
	}
	if _, err := tdm.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").
		RegisterDeleteCallback(tdm.DeleteOrgFromCache); err != nil {
		log.InfraErr(err).Msg("Failed to RegisterDeleteCallback for Project")
		return err
	}
	return nil
}

func (tdm *TenancyDM) startPeriodicReconciler(gctx context.Context) {
	log.Debug().Msgf("Starting PeriodicReconciler: %f sec", tdm.reconcileTime.Seconds())
	ticker := time.NewTicker(tdm.reconcileTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tdm.PeriodicReconciler(gctx)
		case <-gctx.Done():
			log.Debug().Msg("Closing ticker")
			return
		}
	}
}

func (tdm *TenancyDM) Start(gctx context.Context) error {
	if err := tdm.registerProjectCallbacks(); err != nil {
		log.InfraErr(err).Msg("error registering project callbacks")
		return err
	}

	if err := tdm.registerOrgCallbacks(); err != nil {
		log.InfraErr(err).Msg("error registering org callbacks")
		return err
	}

	tdm.startPeriodicReconciler(gctx)
	return nil
}
