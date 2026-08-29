/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RestartPlanSpec defines the desired state of RestartPlan.
type RestartPlanSpec struct {
	// Deployments contains the Deployments that will be rolled out
	// sequentially when the ConfigMap changes.
	Deployments []string `json:"deployments"`

	// ConfigMap is the name of the ConfigMap watched by this RestartPlan.
	ConfigMap string `json:"configMap"`

	// Paused prevents the RestartPlan from performing rollouts.
	Paused bool `json:"paused,omitempty"`
}

// DeploymentStatus represents the observed state of a Deployment
// managed by this RestartPlan.
type DeploymentStatus struct {
	// Name is the name of the Deployment.
	Name string `json:"name"`

	// Phase represents the current rollout phase.
	// +kubebuilder:validation:Enum=Pending;RollingOut;Completed;Error;Paused
	Phase string `json:"phase,omitempty"`

	// ConfigMapResourceVersion identifies the ConfigMap version
	// that triggered this Deployment rollout.
	ConfigMapResourceVersion string `json:"configMapResourceVersion,omitempty"`
}

// RestartPlanStatus defines the observed state of RestartPlan.
type RestartPlanStatus struct {
	// Phase represents the current state of the RestartPlan.
	// +kubebuilder:validation:Enum=Creating;Created;RollingOut;Paused;Error
	Phase string `json:"phase,omitempty"`

	// Message contains a human-readable description of the current state.
	Message string `json:"message,omitempty"`

	// ConfigMapResourceVersion is the resourceVersion of the ConfigMap
	// that triggered the current or last rollout.
	ConfigMapResourceVersion string `json:"configMapResourceVersion,omitempty"`

	// Deployments contains the observed rollout state of every
	// Deployment managed by this RestartPlan.
	Deployments []DeploymentStatus `json:"deployments,omitempty"`

	// Conditions represent detailed observations about the RestartPlan.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ConfigMap",type="string",JSONPath=".spec.configMap"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// RestartPlan is the Schema for the restartplans API.
type RestartPlan struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of the RestartPlan.
	// +required
	Spec RestartPlanSpec `json:"spec"`

	// Status defines the observed state of the RestartPlan.
	// +optional
	Status RestartPlanStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
// RestartPlanList contains a list of RestartPlan.
type RestartPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`

	Items []RestartPlan `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		func(s *runtime.Scheme) error {
			s.AddKnownTypes(
				SchemeGroupVersion,
				&RestartPlan{},
				&RestartPlanList{},
			)
			return nil
		},
	)
}
