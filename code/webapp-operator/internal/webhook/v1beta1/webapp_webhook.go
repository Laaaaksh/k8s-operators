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
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
	"github.com/Laaaaksh/k8s-operators/code/webapp-operator/internal/registry"
)

// nolint:unused
// log is for logging in this package.
var webapplog = logf.Log.WithName("webapp-resource")

// SetupWebAppWebhookWithManager registers every webhook for the Hub
// version: defaulting and validation (this file's business logic, a
// straight port of v1alpha1's - see api/v1alpha1's webhook for why each
// rule exists), plus - because WebApp implements conversion.Hub - a
// generic "/convert" handler the builder wires in automatically. v1alpha1
// keeps no defaulting/validating webhook of its own past this stage: every
// v1alpha1 write gets converted to this type first, so validating and
// defaulting it here is enough to cover both served versions. See
// curriculum/10-versioning.md.
func SetupWebAppWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &appsv1beta1.WebApp{}).
		WithValidator(&WebAppCustomValidator{}).
		WithDefaulter(&WebAppCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-apps-example-com-v1beta1-webapp,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.example.com,resources=webapps,verbs=create;update,versions=v1beta1,name=mwebapp-v1beta1.kb.io,admissionReviewVersions=v1

// WebAppCustomDefaulter sets defaults for the Hub version. See
// api/v1alpha1's WebAppCustomDefaulter for why this can't just be a CRD
// schema default.
type WebAppCustomDefaulter struct{}

// Default implements webhook.CustomDefaulter.
func (d *WebAppCustomDefaulter) Default(_ context.Context, obj *appsv1beta1.WebApp) error {
	webapplog.Info("Defaulting for WebApp", "name", obj.GetName())

	if obj.Spec.HostName == "" {
		obj.Spec.HostName = fmt.Sprintf("%s.%s.apps.example.com", obj.Name, obj.Namespace)
	}

	return nil
}

var webAppGK = schema.GroupKind{Group: appsv1beta1.GroupVersion.Group, Kind: "WebApp"}

// +kubebuilder:webhook:path=/validate-apps-example-com-v1beta1-webapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.example.com,resources=webapps,verbs=create;update,versions=v1beta1,name=vwebapp-v1beta1.kb.io,admissionReviewVersions=v1

// WebAppCustomValidator validates the Hub version. Same two rules as
// v1alpha1's validator - see that package's comments for the reasoning -
// plus one Scaling-specific rule that has no v1alpha1 equivalent.
type WebAppCustomValidator struct{}

// ValidateCreate implements webhook.CustomValidator.
func (v *WebAppCustomValidator) ValidateCreate(_ context.Context, obj *appsv1beta1.WebApp) (admission.Warnings, error) {
	webapplog.Info("Validation for WebApp upon creation", "name", obj.GetName())

	if errs := validateScaling(obj); len(errs) > 0 {
		return nil, apierrors.NewInvalid(webAppGK, obj.Name, errs)
	}

	if obj.Namespace == registry.Namespace {
		return nil, apierrors.NewInvalid(webAppGK, obj.Name, field.ErrorList{
			field.Forbidden(field.NewPath("metadata", "namespace"),
				fmt.Sprintf("%q is reserved for the WebApp registry and cannot contain WebApp resources", registry.Namespace)),
		})
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator.
func (v *WebAppCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *appsv1beta1.WebApp) (admission.Warnings, error) {
	webapplog.Info("Validation for WebApp upon update", "name", newObj.GetName())

	if errs := validateScaling(newObj); len(errs) > 0 {
		return nil, apierrors.NewInvalid(webAppGK, newObj.Name, errs)
	}

	if oldObj.Spec.HostName != "" && newObj.Spec.HostName != oldObj.Spec.HostName {
		return nil, apierrors.NewInvalid(webAppGK, newObj.Name, field.ErrorList{
			field.Forbidden(field.NewPath("spec", "hostName"),
				fmt.Sprintf("is immutable once set (was %q) - delete and recreate the WebApp instead", oldObj.Spec.HostName)),
		})
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator.
func (v *WebAppCustomValidator) ValidateDelete(_ context.Context, obj *appsv1beta1.WebApp) (admission.Warnings, error) {
	return nil, nil
}

// validateScaling is the one rule this version's CRD schema genuinely
// cannot express on its own without CEL: minReplicas and maxReplicas are
// each independently valid (both pass Minimum=0/Maximum=100), but their
// *relationship* - min must not exceed max - is a cross-field constraint.
func validateScaling(obj *appsv1beta1.WebApp) field.ErrorList {
	if obj.Spec.Scaling.MinReplicas > obj.Spec.Scaling.MaxReplicas {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "scaling", "minReplicas"), obj.Spec.Scaling.MinReplicas,
				fmt.Sprintf("must be <= maxReplicas (%d)", obj.Spec.Scaling.MaxReplicas)),
		}
	}
	return nil
}
