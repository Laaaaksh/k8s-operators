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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WebAppSpec defines the desired state of a WebApp: a stateless HTTP
// application the operator runs as a Deployment fronted by a Service.
//
// Every field here is validated by the OpenAPI schema markers below - the
// API server rejects an invalid WebApp before it ever reaches the
// controller. See curriculum/01-api-design.md for why that split matters.
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

	// replicas is the desired number of pods. Zero is valid and means
	// "scaled to zero, but still registered" - see status.phase.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Replicas int32 `json:"replicas,omitempty"`

	// hostName is the externally-visible hostname this WebApp registers
	// under in the cluster's WebApp registry (see internal/registry). It
	// must be unique across every WebApp in the cluster - the registry is
	// how that's enforced. If empty, the mutating webhook introduced in
	// Stage 6 derives one from the WebApp's name and namespace.
	// +optional
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	HostName string `json:"hostName,omitempty"`
}

// WebAppPhase is a coarse, human-readable summary of a WebApp's state.
// It's a convenience for `kubectl get` output - the Conditions below are
// the machine-readable source of truth. See
// curriculum/02-reconciler-and-status.md for why both exist.
// +kubebuilder:validation:Enum=Pending;Progressing;Available;Degraded
type WebAppPhase string

const (
	WebAppPhasePending     WebAppPhase = "Pending"
	WebAppPhaseProgressing WebAppPhase = "Progressing"
	WebAppPhaseAvailable   WebAppPhase = "Available"
	WebAppPhaseDegraded    WebAppPhase = "Degraded"
)

// Condition types this controller sets. Mirrors the Kubernetes API
// conventions' recommended condition types
// (https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties),
// not an invented vocabulary.
const (
	// ConditionAvailable is True when the Deployment has at least one ready
	// replica (or Spec.Replicas is 0 and none are expected).
	ConditionAvailable = "Available"
	// ConditionProgressing is True while the Deployment/Service are being
	// created or updated to match Spec.
	ConditionProgressing = "Progressing"
	// ConditionDegraded is True when reconciliation is failing - a bad
	// image, a registry conflict, or an API error.
	ConditionDegraded = "Degraded"
)

// WebAppStatus defines the observed state of WebApp.
type WebAppStatus struct {
	// phase is a coarse summary of the WebApp's state. Always derived from
	// Conditions, never set independently - see the reconciler's
	// computePhase.
	// +optional
	Phase WebAppPhase `json:"phase,omitempty"`

	// readyReplicas mirrors the managed Deployment's status.readyReplicas.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// observedGeneration is the .metadata.generation the controller last
	// acted on. Compare it to .metadata.generation to tell whether Status
	// reflects the current Spec or a stale one the controller hasn't
	// gotten to yet.
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

// FinalizerName is set on every WebApp so the controller can deregister it
// from the cluster registry before Kubernetes garbage-collects it. See
// curriculum/05-finalizers.md.
const FinalizerName = "apps.example.com/webapp-registry-cleanup"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// WebApp is the Schema for the webapps API
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
