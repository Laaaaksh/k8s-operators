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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
)

// labelsFor returns the labels every resource this controller owns for a
// given WebApp carries. Used both to build new objects and, via
// client.MatchingLabels, to find existing ones - the same map, one source
// of truth, so the two never drift apart.
func labelsFor(webapp *appsv1beta1.WebApp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "webapp",
		"app.kubernetes.io/instance":   webapp.Name,
		"app.kubernetes.io/managed-by": "webapp-operator",
	}
}

// desiredDeployment builds the Deployment this WebApp wants to exist. It
// does not talk to the API server - mutateDeployment (called from inside
// controllerutil.CreateOrUpdate in the reconciler) copies these fields onto
// whatever object CreateOrUpdate fetched or is about to create. Keeping
// "what we want" as a pure function separate from "how we apply it" is
// what makes this easy to unit test without envtest.
func desiredDeployment(webapp *appsv1beta1.WebApp) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name,
			Namespace: webapp.Namespace,
		},
	}
}

// mutateDeployment sets every field this controller manages on dep to match
// webapp. It's written to be safe to call whether dep is a brand-new,
// empty object or one CreateOrUpdate just fetched from the API server -
// it never reads dep's existing values, only overwrites.
//
// It deliberately does NOT touch fields it doesn't own (e.g. it never sets
// dep.Spec.Strategy), so a cluster operator's manual tuning of unrelated
// fields survives reconciliation. Owning exactly the fields you set, and no
// more, is what keeps a controller from fighting other things that touch
// the same object - see curriculum/03-watches-and-ownership.md.
func mutateDeployment(dep *appsv1.Deployment, webapp *appsv1beta1.WebApp) {
	labels := labelsFor(webapp)
	// This controller doesn't implement autoscaling - see ScalingSpec's
	// comment in api/v1beta1/webapp_types.go - so it always runs exactly
	// MinReplicas, ignoring MaxReplicas until something else (a real HPA)
	// reads it.
	replicas := webapp.Spec.Scaling.MinReplicas

	dep.Labels = labels
	dep.Spec.Replicas = &replicas
	dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dep.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: labels},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "webapp",
					Image: webapp.Spec.Image,
					Ports: []corev1.ContainerPort{
						{ContainerPort: webapp.Spec.Port, Name: "http"},
					},
				},
			},
		},
	}
}

// desiredService builds the ClusterIP Service that fronts webapp's pods.
func desiredService(webapp *appsv1beta1.WebApp) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name,
			Namespace: webapp.Namespace,
		},
	}
}

// mutateService is Service's counterpart to mutateDeployment - see that
// function's comment for why the "pure builder + mutate-in-place" split
// exists.
func mutateService(svc *corev1.Service, webapp *appsv1beta1.WebApp) {
	svc.Labels = labelsFor(webapp)
	svc.Spec.Selector = labelsFor(webapp)
	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "http",
			Port:       webapp.Spec.Port,
			TargetPort: intstr.FromString("http"),
			Protocol:   corev1.ProtocolTCP,
		},
	}
}
