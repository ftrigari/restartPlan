package controller

import (
	"context"

	restartv1alpha1 "example.com/restartPlan/api/v1alpha1"
)

// reconcileError handles the Error state.
//
// For now we simply retry the reconciliation by moving back to
// Creating. The next reconciliation will verify the current state
// of the referenced resources again.
func (r *RestartPlanReconciler) reconcileError(
	ctx context.Context,
	plan *restartv1alpha1.RestartPlan,
) error {

	// Error is currently a terminal state.
	//
	// The RestartPlan must be modified/recreated to leave this state.
	return nil
}
