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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ScalingSpec replaces v1alpha1's single Replicas field. This is the
// breaking change this version exists for: v1alpha1 could only say "run
// exactly N replicas", which had no room to express "run between N and M,
// let something else decide" - the shape every autoscaling story needs.
// There's no schema migration that turns an int into this without a
// decision about what MinReplicas and MaxReplicas should be - see this
// package's webapp_conversion.go for the one this repo makes, and
// curriculum/10-versioning.md for why that decision is inherently lossy.
type ScalingSpec struct {
	// minReplicas is the floor: the operator never runs fewer than this
	// many, even under an autoscaler.
	// +required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	MinReplicas int32 `json:"minReplicas"`

	// maxReplicas is the ceiling. This repo's controller does not
	// implement autoscaling itself - it always runs exactly MinReplicas -
	// but the field exists so a real HorizontalPodAutoscaler (out of
	// scope here; see the root README's "what this repo doesn't cover")
	// has somewhere to read a ceiling from.
	// +required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	MaxReplicas int32 `json:"maxReplicas"`
}

// WebAppSpec defines the desired state of a WebApp. Identical to
// v1alpha1.WebAppSpec except Replicas -> Scaling - see ScalingSpec's
// comment for why.
type WebAppSpec struct {
	// image is the container image to run, e.g. "nginx:1.27".
	// +required
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// port is the container port the application listens on, and the port
	// the generated Service exposes.
	// +optional
	// +kubebuilder:default=8080
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`

	// scaling replaces v1alpha1's flat "replicas" field. See ScalingSpec.
	// +optional
	// +kubebuilder:default={minReplicas: 1, maxReplicas: 1}
	Scaling ScalingSpec `json:"scaling,omitzero"`

	// hostName is the externally-visible hostname this WebApp registers
	// under in the cluster's WebApp registry. Unchanged from v1alpha1.
	// +optional
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	HostName string `json:"hostName,omitempty"`
}

// WebAppPhase mirrors v1alpha1.WebAppPhase - see that type's comment.
// +kubebuilder:validation:Enum=Pending;Progressing;Available;Degraded
type WebAppPhase string

const (
	WebAppPhasePending     WebAppPhase = "Pending"
	WebAppPhaseProgressing WebAppPhase = "Progressing"
	WebAppPhaseAvailable   WebAppPhase = "Available"
	WebAppPhaseDegraded    WebAppPhase = "Degraded"
)

// Condition types this controller sets - identical vocabulary to
// v1alpha1, unaffected by the Scaling change.
const (
	ConditionAvailable   = "Available"
	ConditionProgressing = "Progressing"
	ConditionDegraded    = "Degraded"
)

// WebAppStatus defines the observed state of WebApp. Identical to
// v1alpha1.WebAppStatus - only Spec changed in this version.
type WebAppStatus struct {
	// +optional
	Phase WebAppPhase `json:"phase,omitempty"`
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// conditions represent the current state of the WebApp resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// FinalizerName mirrors v1alpha1.FinalizerName.
const FinalizerName = "apps.example.com/webapp-registry-cleanup"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Min",type=integer,JSONPath=`.spec.scaling.minReplicas`
// +kubebuilder:printcolumn:name="Max",type=integer,JSONPath=`.spec.scaling.maxReplicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// WebApp is the Schema for the webapps API. This is the storage version
// (+kubebuilder:storageversion) and the conversion Hub - see
// webapp_conversion.go and curriculum/10-versioning.md. v1alpha1 remains
// served, converting through this type, for clients that haven't moved
// yet.
type WebApp struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of WebApp
	// +required
	Spec WebAppSpec `json:"spec"`

	// status defines the observed state of WebApp
	// +optional
	Status WebAppStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// WebAppList contains a list of WebApp
type WebAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []WebApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &WebApp{}, &WebAppList{})
		return nil
	})
}
