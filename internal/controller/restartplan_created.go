package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	restartv1alpha1 "example.com/restartPlan/api/v1alpha1"
)

func (r *RestartPlanReconciler) reconcileCreated(
	ctx context.Context,
	plan *restartv1alpha1.RestartPlan,
) error {

	changed, err := r.configurationChanged(ctx, plan)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	plan.Status.Phase = "RollingOut"
	plan.Status.Message = "Configuration changed, rollout required"

	// Start a new rollout cycle.
	plan.Status.Deployments = nil

	return nil
}

func (r *RestartPlanReconciler) configurationChanged(
	ctx context.Context,
	plan *restartv1alpha1.RestartPlan,
) (bool, error) {

	var configMap corev1.ConfigMap

	if err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      plan.Spec.ConfigMap,
			Namespace: plan.Namespace,
		},
		&configMap,
	); err != nil {
		return false, err
	}

	currentVersion := configMap.ResourceVersion

	if currentVersion == plan.Status.ConfigMapResourceVersion {
		return false, nil
	}

	// Store the version that triggered this rollout.
	plan.Status.ConfigMapResourceVersion = currentVersion

	return true, nil
}
