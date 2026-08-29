package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	restartv1alpha1 "example.com/restartPlan/api/v1alpha1"
)

// reconcileRollingOut manages the sequential rollout of Deployments.
//
// Only one Deployment is processed at a time.
// The next Deployment starts only after the current one
// has completely rolled out.
func (r *RestartPlanReconciler) reconcileRollingOut(
	ctx context.Context,
	plan *restartv1alpha1.RestartPlan,
) error {

	// Build a map to quickly access the status
	// of each Deployment.
	statusMap := make(
		map[string]restartv1alpha1.DeploymentStatus,
	)

	for _, deploymentStatus := range plan.Status.Deployments {
		statusMap[deploymentStatus.Name] = deploymentStatus
	}

	// Process Deployments in the order specified in the CR.
	for _, deploymentName := range plan.Spec.Deployments {

		deploymentStatus, exists :=
			statusMap[deploymentName]

		// Already completed.
		if exists && deploymentStatus.Phase == "Completed" {
			continue
		}

		// Get Deployment.
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
				"failed to get deployment %q: %w",
				deploymentName,
				err,
			)
		}

		// Annotation identifying the ConfigMap version
		// that triggered this rollout.
		//
		// Example:
		//
		// restart.example.com/restartplan-sample: "90976025"
		//
		annotationKey :=
			"restart.example.com/" + plan.Name

		annotationValue :=
			plan.Status.ConfigMapResourceVersion

		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations =
				make(map[string]string)
		}

		currentAnnotationValue :=
			deployment.Spec.Template.Annotations[annotationKey]

		// ---------------------------------------------------------
		// Deployment has not been started yet.
		// ---------------------------------------------------------

		if !exists ||
			deploymentStatus.Phase == "" ||
			deploymentStatus.Phase == "Pending" {

			// Idempotency check.
			//
			// The Deployment already contains the ConfigMap
			// resourceVersion we want to apply.
			//
			// This can happen if the controller crashes after
			// updating the Deployment but before updating the CR.
			if currentAnnotationValue == annotationValue {

				r.updateDeploymentStatus(
					plan,
					deploymentName,
					"RollingOut",
					annotationValue,
				)

				plan.Status.Message = fmt.Sprintf(
					"Waiting for deployment %q to complete",
					deploymentName,
				)

				return nil
			}

			// Apply the ConfigMap resourceVersion to the
			// Deployment PodTemplate.
			deployment.Spec.Template.Annotations[annotationKey] = annotationValue

			// Update Deployment.
			if err := r.Update(
				ctx,
				&deployment,
			); err != nil {
				return fmt.Errorf(
					"failed to update deployment %q: %w",
					deploymentName,
					err,
				)
			}

			// Store the ConfigMap ResourceVersion in the
			// RestartPlan status.
			//
			// We intentionally do NOT store Deployment.Generation
			// here. Generation is only used internally to determine
			// whether the rollout has completed.
			r.updateDeploymentStatus(
				plan,
				deploymentName,
				"RollingOut",
				annotationValue,
			)

			plan.Status.Message = fmt.Sprintf(
				"Rolling out deployment %q",
				deploymentName,
			)

			// Only one Deployment at a time.
			return nil
		}

		// ---------------------------------------------------------
		// Deployment is currently rolling out.
		// ---------------------------------------------------------

		if deploymentStatus.Phase == "RollingOut" {

			// The ConfigMap version that triggered this rollout
			// is stored in the CR status.
			//
			// The Deployment generation is NOT stored in the CR.
			//
			// Kubernetes tells us whether the current generation
			// has been observed and whether all replicas are ready.
			if !isDeploymentRolloutComplete(&deployment) {

				plan.Status.Message = fmt.Sprintf(
					"Waiting for deployment %q to complete",
					deploymentName,
				)

				return nil
			}

			// Rollout completed.
			r.updateDeploymentStatus(
				plan,
				deploymentName,
				"Completed",
				deploymentStatus.ConfigMapResourceVersion,
			)

			plan.Status.Message = fmt.Sprintf(
				"Deployment %q rollout completed",
				deploymentName,
			)

			// Stop here.
			//
			// The next reconciliation will start the next
			// Deployment.
			return nil
		}
	}

	// -------------------------------------------------------------
	// All Deployments completed.
	// -------------------------------------------------------------

	plan.Status.Phase = "Created"

	plan.Status.Message =
		"All deployments rolled out successfully, watching for changes"

	// Keep the Deployment statuses in the CR.
	return nil
}

// isDeploymentRolloutComplete checks whether the Deployment
// has completely rolled out its current generation.
func isDeploymentRolloutComplete(
	deployment *appsv1.Deployment,
) bool {

	// The Deployment controller has not observed the current
	// generation yet.
	if deployment.Status.ObservedGeneration <
		deployment.Generation {

		return false
	}

	replicas := int32(1)

	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	// All desired replicas must be updated.
	if deployment.Status.UpdatedReplicas != replicas {
		return false
	}

	// All desired replicas must exist.
	if deployment.Status.Replicas != replicas {
		return false
	}

	// All desired replicas must be available.
	if deployment.Status.AvailableReplicas != replicas {
		return false
	}

	return true
}

// updateDeploymentStatus updates or creates the status entry
// for a Deployment inside the RestartPlan.
func (r *RestartPlanReconciler) updateDeploymentStatus(
	plan *restartv1alpha1.RestartPlan,
	name string,
	phase string,
	configMapResourceVersion string,
) {

	for i := range plan.Status.Deployments {

		if plan.Status.Deployments[i].Name != name {
			continue
		}

		plan.Status.Deployments[i].Phase = phase
		plan.Status.Deployments[i].ConfigMapResourceVersion =
			configMapResourceVersion

		return
	}

	plan.Status.Deployments = append(
		plan.Status.Deployments,
		restartv1alpha1.DeploymentStatus{
			Name:                     name,
			Phase:                    phase,
			ConfigMapResourceVersion: configMapResourceVersion,
		},
	)
}
