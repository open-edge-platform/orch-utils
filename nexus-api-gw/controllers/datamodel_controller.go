// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"errors"

	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// DatamodelReconciler reconciles a Datamodels object.
type DatamodelReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	StopCh  chan struct{}
	Dynamic dynamic.Interface
}

//+kubebuilder:rbac:groups=apiextensions.k8s.io.api-gw.com,resources=datamodels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apiextensions.k8s.io.api-gw.com,resources=datamodels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apiextensions.k8s.io.api-gw.com,resources=datamodels/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

func (r *DatamodelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = ctrllog.FromContext(ctx)

	eventType := model.Upsert

	if r.Dynamic == nil {
		return ctrl.Result{}, errors.New("dynamic client is not initialized")
	}

	obj, err := r.Dynamic.Resource(schema.GroupVersionResource{
		Group:    "nexus.com",
		Version:  "v1",
		Resource: "datamodels",
	}).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		eventType = model.Delete
	}

	log.Info().Msgf("Received Datamodel notification for Name %s Type %s", req.Name, eventType)
	if obj != nil {
		log.Info().Msgf("Datamodel Object: %s", obj)
	} else {
		log.Info().Msgf("Datamodel Object is nil")
	}
	model.ConstructDatamodel(eventType, req.Name, obj)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatamodelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "Datamodel",
		Group:   "nexus.com",
		Version: "v1",
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(u).
		Complete(r)
}
