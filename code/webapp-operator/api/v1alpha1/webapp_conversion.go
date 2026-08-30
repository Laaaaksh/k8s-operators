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
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
)

// ConvertTo and ConvertFrom make WebApp a conversion.Convertible spoke of
// v1beta1's Hub. Whenever the API server needs a v1alpha1 representation
// of an object stored as v1beta1 (or vice versa) - e.g. `kubectl get
// webapp.v1alpha1` after every WebApp in the cluster is already stored as
// v1beta1 - it calls these, not any code in the reconciler.
//
// The lossy half of this file is ConvertFrom: v1beta1.Scaling has both a
// MinReplicas and a MaxReplicas, v1alpha1.Spec.Replicas has room for only
// one number. This repo's rule, stated once so both directions agree with
// each other, is: v1alpha1's single Replicas field maps to v1beta1's
// MinReplicas going up, and reading MinReplicas back going down.
// MaxReplicas has no v1alpha1 equivalent at all - converting v1beta1 to
// v1alpha1 silently drops it. That's real, observable data loss (round-trip
// a WebApp through v1alpha1 and MaxReplicas resets to MinReplicas next
// time anything reads it back as v1beta1) - the exact failure mode
// curriculum/10-versioning.md's "why this needs a decision, not just a
// schema" section is about. It's also why a validating webhook can't fix
// this: there is no wrong answer to reject, only an inherently smaller
// old shape being asked to hold a newer one.

// ConvertTo converts this v1alpha1 WebApp to the v1beta1 Hub type.
func (src *WebApp) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*appsv1beta1.WebApp)

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Port = src.Spec.Port
	dst.Spec.HostName = src.Spec.HostName
	dst.Spec.Scaling = appsv1beta1.ScalingSpec{
		MinReplicas: src.Spec.Replicas,
		MaxReplicas: src.Spec.Replicas,
	}

	dst.Status.Phase = appsv1beta1.WebAppPhase(src.Status.Phase)
	dst.Status.ReadyReplicas = src.Status.ReadyReplicas
	dst.Status.ObservedGeneration = src.Status.ObservedGeneration
	dst.Status.Conditions = src.Status.Conditions

	return nil
}

// ConvertFrom populates this v1alpha1 WebApp from the v1beta1 Hub type.
// See this file's top comment for the MaxReplicas data loss this
// direction incurs.
func (dst *WebApp) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*appsv1beta1.WebApp)

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Port = src.Spec.Port
	dst.Spec.HostName = src.Spec.HostName
	dst.Spec.Replicas = src.Spec.Scaling.MinReplicas

	dst.Status.Phase = WebAppPhase(src.Status.Phase)
	dst.Status.ReadyReplicas = src.Status.ReadyReplicas
	dst.Status.ObservedGeneration = src.Status.ObservedGeneration
	dst.Status.Conditions = src.Status.Conditions

	return nil
}
