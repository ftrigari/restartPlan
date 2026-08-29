/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	restartv1alpha1 "example.com/restartPlan/api/v1alpha1"
)

// RestartPlanReconciler reconciles a RestartPlan object.
type RestartPlanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=restart.example.com,resources=restartplans,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=restart.example.com,resources=restartplans/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=restart.example.com,resources=restartplans/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *RestartPlanReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	var plan restartv1alpha1.RestartPlan

	// ---------------------------------------------------------
	// Get RestartPlan
	// ---------------------------------------------------------

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&plan,
	); err != nil {

		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	log.Info(
		"Reconciling RestartPlan",
		"name", plan.Name,
		"namespace", plan.Namespace,
		"phase", plan.Status.Phase,
	)

	// ---------------------------------------------------------
	// Paused
	// ---------------------------------------------------------

	if plan.Spec.Paused {

		plan.Status.Message = "RestartPlan is paused"

		if err := r.Status().Update(
			ctx,
			&plan,
		); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// ---------------------------------------------------------
	// State machine
	// ---------------------------------------------------------

	var err error

	switch plan.Status.Phase {

	case "":
		plan.Status.Phase = "Creating"
		plan.Status.Message = "Creating RestartPlan"

	case "Creating":
		err = r.reconcileCreating(
			ctx,
			&plan,
		)

	case "Created":
		err = r.reconcileCreated(
			ctx,
			&plan,
		)

	case "RollingOut":
		err = r.reconcileRollingOut(
			ctx,
			&plan,
		)

	case "Error":
		err = r.reconcileError(
			ctx,
			&plan,
		)

	default:
		plan.Status.Phase = "Error"
		plan.Status.Message = "Unknown RestartPlan phase"
	}

	// ---------------------------------------------------------
	// Handle reconciliation error
	// ---------------------------------------------------------

	if err != nil {

		log.Error(
			err,
			"Reconciliation failed",
		)

		// Conflict is a transient error.
		//
		// Do not change the RestartPlan status to Error.
		// Returning the error makes controller-runtime retry
		// using the workqueue rate limiter/backoff.
		if apierrors.IsConflict(err) {
			return ctrl.Result{}, err
		}

		// Real reconciliation error.
		plan.Status.Phase = "Error"
		plan.Status.Message = err.Error()
	}

	// ---------------------------------------------------------
	// Update status
	// ---------------------------------------------------------

	if statusErr := r.Status().Update(
		ctx,
		&plan,
	); statusErr != nil {

		// A conflict while updating status is also transient.
		// Return the error so controller-runtime retries it.
		return ctrl.Result{}, statusErr
	}

	// Return real reconciliation errors after recording them
	// in the RestartPlan status.
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// mapConfigMapToRestartPlan maps a ConfigMap event to every
// RestartPlan that references that ConfigMap in the same namespace.
func (r *RestartPlanReconciler) mapConfigMapToRestartPlan(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {

	configMap, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}

	var plans restartv1alpha1.RestartPlanList

	if err := r.List(
		ctx,
		&plans,
		client.InNamespace(configMap.Namespace),
	); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0)

	for _, plan := range plans.Items {

		if plan.Spec.ConfigMap != configMap.Name {
			continue
		}

		requests = append(
			requests,
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      plan.Name,
					Namespace: plan.Namespace,
				},
			},
		)
	}

	return requests
}

// mapDeploymentToRestartPlan maps a Deployment event to every
// RestartPlan that references that Deployment.
func (r *RestartPlanReconciler) mapDeploymentToRestartPlan(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {

	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil
	}

	var plans restartv1alpha1.RestartPlanList

	if err := r.List(
		ctx,
		&plans,
		client.InNamespace(deployment.Namespace),
	); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0)

	for _, plan := range plans.Items {

		for _, deploymentName := range plan.Spec.Deployments {

			if deploymentName != deployment.Name {
				continue
			}

			requests = append(
				requests,
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      plan.Name,
						Namespace: plan.Namespace,
					},
				},
			)

			break
		}
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestartPlanReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&restartv1alpha1.RestartPlan{}).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(
				r.mapConfigMapToRestartPlan,
			),
		).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(
				r.mapDeploymentToRestartPlan,
			),
		).
		Named("restartplan").
		Complete(r)
}
