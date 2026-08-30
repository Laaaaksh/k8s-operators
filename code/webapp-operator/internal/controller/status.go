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

package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
)

// applyDeploymentStatus updates webapp.Status from the current state of
// dep. It sets Conditions using meta.SetStatusCondition, which is the
// controller-runtime helper for the pattern every Kubernetes API object
// uses: update-in-place-by-Type, bump LastTransitionTime only when Status
// actually changes. Never construct a fresh []metav1.Condition slice here -
// that would reset LastTransitionTime on every reconcile, which breaks
// anything watching "how long has this condition been true" (including
// `kubectl get` age columns and alerting rules).
func applyDeploymentStatus(webapp *appsv1beta1.WebApp, dep *appsv1.Deployment, err error) {
	webapp.Status.ObservedGeneration = webapp.Generation

	if err != nil {
		setCondition(webapp, appsv1beta1.ConditionDegraded, metav1.ConditionTrue, "ReconcileError", err.Error())
		setCondition(webapp, appsv1beta1.ConditionAvailable, metav1.ConditionFalse, "ReconcileError", err.Error())
		setCondition(webapp, appsv1beta1.ConditionProgressing, metav1.ConditionFalse, "ReconcileError", err.Error())
		webapp.Status.Phase = appsv1beta1.WebAppPhaseDegraded
		return
	}

	webapp.Status.ReadyReplicas = dep.Status.ReadyReplicas
	setCondition(webapp, appsv1beta1.ConditionDegraded, metav1.ConditionFalse, "AsExpected", "no errors reconciling")

	wantReplicas := webapp.Spec.Scaling.MinReplicas
	haveReady := dep.Status.ReadyReplicas
	upToDate := dep.Status.ObservedGeneration >= dep.Generation

	switch {
	case wantReplicas == 0:
		// Scaled to zero on purpose. Not "unavailable" - available with
		// nothing to serve. Distinguishing "off" from "broken" is the
		// whole reason Phase exists as a field separate from a single
		// boolean "ready".
		setCondition(webapp, appsv1beta1.ConditionAvailable, metav1.ConditionTrue, "ScaledToZero", "replicas is 0")
		setCondition(webapp, appsv1beta1.ConditionProgressing, metav1.ConditionFalse, "ScaledToZero", "replicas is 0")
		webapp.Status.Phase = appsv1beta1.WebAppPhaseAvailable
	case upToDate && haveReady >= wantReplicas:
		setCondition(webapp, appsv1beta1.ConditionAvailable, metav1.ConditionTrue, "MinimumReplicasReady",
			"all desired replicas are ready")
		setCondition(webapp, appsv1beta1.ConditionProgressing, metav1.ConditionFalse, "MinimumReplicasReady",
			"all desired replicas are ready")
		webapp.Status.Phase = appsv1beta1.WebAppPhaseAvailable
	default:
		setCondition(webapp, appsv1beta1.ConditionAvailable, metav1.ConditionFalse, "ReplicasNotReady",
			"waiting for the Deployment to report enough ready replicas")
		setCondition(webapp, appsv1beta1.ConditionProgressing, metav1.ConditionTrue, "ReplicasNotReady",
			"waiting for the Deployment to report enough ready replicas")
		webapp.Status.Phase = appsv1beta1.WebAppPhaseProgressing
	}
}

func setCondition(webapp *appsv1beta1.WebApp, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&webapp.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: webapp.Generation,
	})
}
