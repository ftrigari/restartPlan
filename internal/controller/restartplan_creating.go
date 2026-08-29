package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	restartv1alpha1 "example.com/restartPlan/api/v1alpha1"
)

func (r *RestartPlanReconciler) reconcileCreating(
	ctx context.Context,
	plan *restartv1alpha1.RestartPlan,
) error {

	// Check all Deployments.
	for _, deploymentName := range plan.Spec.Deployments {

		var deployment appsv1.Deployment

		if err := r.Get(
			ctx,
			types.NamespacedName{
				Name:      deploymentName,
				Namespace: plan.Namespace,
			},
			&deployment,
		); err != nil {

			return fmt.Errorf(
				"deployment %q not found: %w",
				deploymentName,
				err,
			)
		}
	}

	// Check ConfigMap.
	var configMap corev1.ConfigMap

	if err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      plan.Spec.ConfigMap,
			Namespace: plan.Namespace,
		},
		&configMap,
	); err != nil {

		return fmt.Errorf(
			"configmap %q not found: %w",
			plan.Spec.ConfigMap,
			err,
		)
	}

	// IMPORTANT:
	// The current ConfigMap becomes the baseline.
	//
	// Therefore creating the RestartPlan does NOT trigger a rollout.
	plan.Status.ConfigMapResourceVersion = configMap.ResourceVersion

	plan.Status.Phase = "Created"
	plan.Status.Message = "RestartPlan is ready"

	return nil
}
