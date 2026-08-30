# Stage 5 — Finalizers

**You'll be able to:** explain what a finalizer actually blocks, and use one
to safely clean up state that Kubernetes' own garbage collector can't
reach.

**Time:** 2–3 hours.

**Build:** `internal/registry/registry.go` and the deletion-handling branch
in `internal/controller/webapp_controller.go`'s `Reconcile`/`finalize`.

## Read, in this order

1. [Kubernetes: finalizers concept](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/) - what `metadata.finalizers` and `deletionTimestamp` actually do at the API level.
2. [Kubebuilder book: using finalizers](https://book.kubebuilder.io/reference/using-finalizers.html) - the reconciler-side pattern this stage's code follows.

## Why the Deployment and Service didn't need one, and the registry does

[Stage 3](03-watches-and-ownership.md) already gives WebApp's Deployment
and Service free cleanup via owner references - Kubernetes' garbage
collector deletes them when the WebApp goes, no finalizer required. A
finalizer is for the specific case owner references can't cover: **cleanup
that isn't itself owned by the one object being deleted.**

This repo's example is `internal/registry`: a single ConfigMap
(`webapp-system/webapp-registry`) that every WebApp in the cluster writes
one entry into - a hostname-to-owner mapping standing in for a real
external system (DNS, a load balancer, a service mesh) that would need to
be told "this WebApp is gone." A ConfigMap owned by many WebApps can't have
`ownerReferences` pointing at just one of them; nothing deletes a WebApp's
entry automatically. A finalizer is what buys the controller time to do
that deregistration itself before the object is actually allowed to
disappear.

## The mechanism, in this repo's code

Every WebApp gets the finalizer added on its first non-deleting reconcile:

```go
if !controllerutil.ContainsFinalizer(&webapp, appsv1beta1.FinalizerName) {
    controllerutil.AddFinalizer(&webapp, appsv1beta1.FinalizerName)
    if err := r.Update(ctx, &webapp); err != nil { ... }
    return ctrl.Result{}, nil
}
```

`kubectl delete webapp <name>` doesn't remove the object - it sets
`metadata.deletionTimestamp` and otherwise leaves it exactly as it is, as
long as `metadata.finalizers` is non-empty. `Reconcile` sees that
timestamp and switches into cleanup mode instead of its usual
converge-Spec-to-Status work:

```go
if !webapp.DeletionTimestamp.IsZero() {
    return r.finalize(ctx, &webapp)
}
```

`finalize` deregisters the hostname, then removes the finalizer and issues
one more `Update` - and it's *that* `Update`, not the earlier `kubectl
delete`, that actually lets the object go:

```go
if err := registry.Deregister(ctx, r.Client, webapp.Namespace, webapp.Name); err != nil {
    return ctrl.Result{}, fmt.Errorf("deregistering hostname during deletion: %w", err)
}
controllerutil.RemoveFinalizer(webapp, appsv1alpha1.FinalizerName)
if err := r.Update(ctx, webapp); err != nil { ... }
```

If `Deregister` fails, `finalize` returns the error, the finalizer stays,
and the WebApp stays visible (with a `deletionTimestamp`) no matter how
many times `kubectl delete` is re-run - the object is now, correctly,
stuck until deregistration can succeed. That's the whole point: a
finalizer trades "delete always works instantly" for "delete only
completes once cleanup genuinely has."

## Verified live

On a real kind cluster (v1.37.0, verified 2026-08-30): create a WebApp
with a `hostName`, confirm the registry ConfigMap gets an entry, `kubectl
delete` it, and both the registry entry and the WebApp object itself are
gone within seconds - along with the Deployment and Service, garbage
collected via their owner references in parallel with the finalizer doing
its own cleanup:

```
$ kubectl get configmap webapp-registry -n webapp-system -o jsonpath='{.data}'
{"demo.default.apps.example.com":"default/demo"}
$ kubectl delete webapp demo
webapp.apps.example.com "demo" deleted from default namespace
$ kubectl get configmap webapp-registry -n webapp-system -o jsonpath='{.data}'
$ kubectl get deploy,svc -l app.kubernetes.io/instance=demo
No resources found in default namespace.
```

## Checkpoint

- `finalize` checks `controllerutil.ContainsFinalizer` before doing any
  cleanup, and returns success immediately if it's already gone. When does
  that branch actually get hit - what real sequence of events creates a
  WebApp that's being deleted but never had this finalizer attached?
- `registry.Deregister` is written to succeed even if the `webapp-system`
  namespace (and therefore the ConfigMap) doesn't exist at all. Why does
  that matter specifically for a finalizer's cleanup path, more than it
  would for the normal `Register` path?
- What would happen to a WebApp stuck mid-deletion if you manually edited
  it with `kubectl edit` and removed `apps.example.com/webapp-registry-cleanup`
  from `metadata.finalizers` yourself? Would the registry entry ever get
  cleaned up?

Next: [Stage 6 — Webhooks](06-webhooks.md).
