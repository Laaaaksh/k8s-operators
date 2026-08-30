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
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
	"github.com/Laaaaksh/k8s-operators/code/webapp-operator/internal/registry"
)

// WebAppReconciler reconciles a WebApp object
type WebAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Recorder emits Kubernetes Events - the entries `kubectl describe
	// webapp` shows and `kubectl get events` lists. Metrics answer "is the
	// fleet healthy"; Events answer "what happened to this one object,
	// and when" - see curriculum/09-observability.md for why an operator
	// needs both, not one or the other.
	//
	// events.EventRecorder, not the older record.EventRecorder most
	// existing operator code and blog posts still show: mgr.GetEventRecorderFor
	// (record.EventRecorder, the events.k8s.io v1beta1-era API) is
	// deprecated as of controller-runtime v0.24 in favor of
	// mgr.GetEventRecorder (this type, backed by the events.k8s.io/v1
	// API) - see curriculum/09-observability.md.
	Recorder events.EventRecorder
}

// The RBAC markers below are audited against actual client calls, not
// copy-pasted from a scaffold and left broad - see
// curriculum/08-least-privilege-rbac.md for the audit method. Two things
// worth noticing:
//
//   - No "delete" on webapps, deployments, or services, even though this
//     controller creates and updates all three. Deployments and Services
//     are deleted by Kubernetes' own garbage collector via ownerReferences
//     when a WebApp goes - that's a different actor with its own
//     permissions, not this ServiceAccount - and this controller never
//     calls Delete on a WebApp itself. kubebuilder's default scaffold
//     grants "delete" on every resource a controller touches; this repo
//     deliberately removes it here as the worked example of the audit,
//     because it's the one a reader is least likely to predict.
//   - "namespaces" only needs get/create, never delete or update: this
//     controller ensures the registry namespace exists and otherwise
//     leaves it alone.
//
// +kubebuilder:rbac:groups=apps.example.com,resources=webapps,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=webapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=webapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;create
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile drives one WebApp toward the state its Spec describes: a
// Deployment and a Service exist, match Spec, and Status reports the
// truth about them. Every reconcile does the same three things, in order -
// ensure the Deployment, ensure the Service, then recompute Status from
// what's actually there - because idempotency (running this twice with no
// change in cluster state produces no change in output) is the property
// that makes level-triggered reconciliation safe to retry, and it's only
// true if every step is phrased as "make it look like this", never "do
// this action".
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var webapp appsv1beta1.WebApp
	if err := r.Get(ctx, req.NamespacedName, &webapp); err != nil {
		// A NotFound here means the WebApp was deleted between the event
		// that queued this request and now. There is nothing left to
		// reconcile - Kubernetes already garbage-collected the Deployment
		// and Service via their owner references, and this is not an
		// error worth retrying, so don't return it (returning an error
		// requeues immediately, and here that would only queue the exact
		// same no-op again).
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// A non-zero DeletionTimestamp means `kubectl delete` already ran, but
	// the object still exists because our finalizer (added below, on every
	// non-deleting reconcile) is blocking removal. This is the one branch
	// where Reconcile's job isn't "converge Spec to Status" - it's "finish
	// the cleanup this object is waiting on, then let it go." See
	// curriculum/05-finalizers.md.
	if !webapp.DeletionTimestamp.IsZero() {
		return r.finalize(ctx, &webapp)
	}

	if !controllerutil.ContainsFinalizer(&webapp, appsv1beta1.FinalizerName) {
		controllerutil.AddFinalizer(&webapp, appsv1beta1.FinalizerName)
		if err := r.Update(ctx, &webapp); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		// The Update above changed .metadata, which is itself a watch
		// event that re-queues this WebApp immediately - return here
		// rather than falling through, so the rest of Reconcile always
		// runs against a webapp value we know has the finalizer, not a
		// stale in-memory copy from before the Update.
		return ctrl.Result{}, nil
	}

	var regErr error
	if webapp.Spec.HostName != "" {
		regErr = registry.Register(ctx, r.Client, webapp.Namespace, webapp.Name, webapp.Spec.HostName)
	}

	var dep *appsv1.Deployment
	reconcileErr := regErr
	if reconcileErr != nil {
		log.Error(reconcileErr, "registering hostname")
		r.Recorder.Eventf(&webapp, nil, corev1.EventTypeWarning, "HostNameConflict", "Register", "%s", reconcileErr)
	} else {
		var depErr error
		dep, depErr = r.reconcileDeployment(ctx, &webapp)
		if depErr != nil {
			log.Error(depErr, "reconciling Deployment")
			reconcileErr = depErr
		}
		if svcErr := r.reconcileService(ctx, &webapp); svcErr != nil && reconcileErr == nil {
			log.Error(svcErr, "reconciling Service")
			reconcileErr = svcErr
		}
	}

	applyDeploymentStatus(&webapp, dep, reconcileErr)
	recordPhaseMetric(webapp.Namespace, webapp.Name, webapp.Status.Phase)
	if err := r.Status().Update(ctx, &webapp); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating WebApp status: %w", err)
	}

	// No RequeueAfter: a Deployment or Service update - including the
	// status update that lands once its pods become Ready - is itself a
	// watch event that re-triggers this Reconcile (see SetupWithManager's
	// Owns() calls below). Polling with RequeueAfter here would just
	// duplicate what the watch already gives you for free, on a delay.
	return ctrl.Result{}, reconcileErr
}

// finalize runs the cleanup a deleting WebApp is blocked on: deregistering
// its hostname from the shared registry ConfigMap. Once that succeeds, it
// removes our finalizer and updates the object - which is what actually
// lets Kubernetes delete it. Until that Update happens, the WebApp stays
// visible (with a DeletionTimestamp) forever, no matter how many times
// `kubectl delete` is re-run.
func (r *WebAppReconciler) finalize(ctx context.Context, webapp *appsv1beta1.WebApp) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(webapp, appsv1beta1.FinalizerName) {
		// Already cleaned up (or never got the finalizer, e.g. a WebApp
		// created and deleted before this controller ever ran) - nothing
		// to do, let deletion proceed.
		return ctrl.Result{}, nil
	}

	if err := registry.Deregister(ctx, r.Client, webapp.Namespace, webapp.Name); err != nil {
		return ctrl.Result{}, fmt.Errorf("deregistering hostname during deletion: %w", err)
	}
	deletePhaseMetric(webapp.Namespace, webapp.Name)

	controllerutil.RemoveFinalizer(webapp, appsv1beta1.FinalizerName)
	if err := r.Update(ctx, webapp); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *WebAppReconciler) reconcileDeployment(ctx context.Context, webapp *appsv1beta1.WebApp) (*appsv1.Deployment, error) {
	dep := desiredDeployment(webapp)
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		mutateDeployment(dep, webapp)
		// SetControllerReference makes Kubernetes' own garbage collector
		// delete dep automatically when webapp is deleted (as long as no
		// finalizer blocks it - see curriculum/05-finalizers.md for the
		// case where that isn't enough), and is what makes Owns() below
		// route this Deployment's events back to this WebApp.
		return ctrl.SetControllerReference(webapp, dep, r.Scheme)
	})
	if err != nil {
		return dep, fmt.Errorf("create/update Deployment %s/%s: %w", webapp.Namespace, webapp.Name, err)
	}
	if op != controllerutil.OperationResultNone {
		logf.FromContext(ctx).Info("reconciled Deployment", "operation", op)
		r.Recorder.Eventf(webapp, dep, corev1.EventTypeNormal, "DeploymentReconciled", "Reconcile", "Deployment %s %s", dep.Name, op)
	}
	return dep, nil
}

func (r *WebAppReconciler) reconcileService(ctx context.Context, webapp *appsv1beta1.WebApp) error {
	svc := desiredService(webapp)
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		mutateService(svc, webapp)
		return ctrl.SetControllerReference(webapp, svc, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("create/update Service %s/%s: %w", webapp.Namespace, webapp.Name, err)
	}
	if op != controllerutil.OperationResultNone {
		logf.FromContext(ctx).Info("reconciled Service", "operation", op)
		r.Recorder.Eventf(webapp, svc, corev1.EventTypeNormal, "ServiceReconciled", "Reconcile", "Service %s %s", svc.Name, op)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta1.WebApp{}).
		// Owns tells the manager: "also watch every Deployment/Service
		// that has a WebApp as its controller owner reference, and when
		// one changes, queue a Reconcile for that owning WebApp" - not
		// for the Deployment/Service itself, there is no reconciler for
		// those. This is what makes `kubectl delete deploy <name>`
		// self-heal within milliseconds instead of waiting for the next
		// poll: no polling happens at all, the delete event itself
		// re-triggers Reconcile.
		//
		// Deliberately NOT filtered with predicate.GenerationChangedPredicate
		// here, unlike many Owns() calls you'll see in other operators.
		// That predicate drops events that don't change .spec - exactly
		// the Deployment status updates (readyReplicas ticking up as pods
		// start) that applyDeploymentStatus needs to see to keep WebApp's
		// own Status current. Filtering those out would make Status lag
		// behind reality until something else happened to trigger a
		// reconcile. Owning a status that needs to track a child's status
		// is the specific case where you want every event, not just spec
		// changes - see curriculum/03-watches-and-ownership.md.
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("webapp").
		Complete(r)
}
