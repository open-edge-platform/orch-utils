// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/model"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// CustomResourceDefinitionReconciler reconciles a CustomResourceDefinition object.
type CustomResourceDefinitionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	StopCh chan struct{}
}

//+kubebuilder:rbac:groups=apiextensions.k8s.io.api-gw.com,resources=customresourcedefinitions,
//+kubebuilder:rbac:verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apiextensions.k8s.io.api-gw.com,resources=customresourcedefinitions/status,
//+kubebuilder:rbac:verbs=get;update;patch
//+kubebuilder:rbac:groups=apiextensions.k8s.io.api-gw.com,resources=customresourcedefinitions/finalizers,
//+kubebuilder:rbac:verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

func (r *CustomResourceDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = ctrllog.FromContext(ctx)

	var crd apiextensionsv1.CustomResourceDefinition
	eventType := model.Upsert
	if err := r.Get(ctx, req.NamespacedName, &crd); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		eventType = model.Delete
	}

	log.Info().Msgf("Received CRD notification for Name %s Type %s\n", crd.Name, eventType)
	if err := r.ProcessAnnotation(req.NamespacedName.Name, crd.Annotations, eventType); err != nil {
		log.Error().Msgf("Error Processing CRD Annotation %v\n", err)
	}

	// Get correct version
	if err := r.ProcessCrdSpec(req.NamespacedName.Name, crd.Spec, eventType); err != nil {
		log.Error().Msgf("Error Processing CRD spec %v\n", err)
	}

	// Recreate openapi specification
	// api . Recreate()

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomResourceDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiextensionsv1.CustomResourceDefinition{}).
		Complete(r)
}
