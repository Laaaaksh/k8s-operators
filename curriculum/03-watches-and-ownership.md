# Stage 3 — Watches & ownership

**You'll be able to:** explain owner references and controller-runtime's
`Owns()`, and make `kubectl delete` on a child resource self-heal within
milliseconds - with no polling loop anywhere.

**Time:** 2–3 hours.

**Build:** the Service in `internal/controller/deployment.go`, and
`SetupWithManager`'s `Owns()` calls in `internal/controller/webapp_controller.go`.

## Read, in this order

1. [controller-runtime: `pkg/builder`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/builder) - `For()`, `Owns()`, `Watches()`. Read the doc comments on `Owns` specifically before the code below.
2. [controller-runtime: `pkg/predicate`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/predicate) - the event-filtering layer `Owns`/`Watches` can be given, including `GenerationChangedPredicate`, which this stage deliberately does *not* use (see below).
3. [Kubebuilder book: watching resources](https://book.kubebuilder.io/reference/watching-resources.html) - owned vs. externally-managed ("secondary") resources, and requeue behavior.
4. If you want the mechanics underneath `Owns`/`Watches` - how controller-runtime's client cache and informers actually work, not just the builder API - [the Kubernetes blog's "How the controller-runtime Cache Actually Works"](https://kubernetes.io/blog/2026/07/29/controller-runtime-cache-explained/) fills a real gap the reference docs above under-explain.

## Owner references: why the Deployment and Service disappear on their own

Every child object this controller creates gets an owner reference back to
the WebApp:

```go
return ctrl.SetControllerReference(webapp, dep, r.Scheme)
```

That's a `metadata.ownerReferences` entry Kubernetes' own garbage collector
watches - delete the WebApp, and the API server deletes the Deployment and
Service for you. No code in this repo ever calls `Delete` on either of them
directly (see [Stage 8](08-least-privilege-rbac.md) for what that costs the
RBAC role - nothing, since it never needs the verb).

## `Owns()`: the other half of the same reference

`ctrl.SetControllerReference` is what makes garbage collection work.
`Owns()` in `SetupWithManager` is what makes **this controller** react to
those objects changing:

```go
Owns(&appsv1.Deployment{}).
Owns(&corev1.Service{}).
```

This tells the manager: watch every Deployment/Service whose owner
reference points at a WebApp, and when one changes, queue a `Reconcile` for
the *owning WebApp* - not for the Deployment or Service, there's no
reconciler for those. `kubectl delete deploy <name>` on this operator's
managed Deployment doesn't wait for a poll interval to notice; the delete
event itself re-triggers `Reconcile`, `controllerutil.CreateOrUpdate` sees
the object is gone, and recreates it - typically inside a second.

## The predicate this stage deliberately skips

Most `Owns()` calls you'll find in other operators are filtered with
`builder.WithPredicates(predicate.GenerationChangedPredicate{})`, which
drops any event that doesn't change the child's `.spec`. This repo's
`Owns()` calls are **not** filtered that way, on purpose - the comment in
`webapp_controller.go` explains why:

> That predicate drops events that don't change `.spec` - exactly the
> Deployment status updates (readyReplicas ticking up as pods start) that
> `applyDeploymentStatus` needs to see to keep WebApp's own Status current.

If you filtered those out, `WebApp.status.readyReplicas` would lag behind
reality until something *else* happened to trigger a reconcile. The
general rule: filter out events you don't care about when you're only
tracking a child's spec drifting; don't filter when your own status needs
to track the child's status too.

## Checkpoint - do this by hand, not just read about it

If you have `kind` and Docker (see [Stage 0](00-toolchain-setup.md)), the
fastest way to feel this is live:

```bash
cd code/webapp-operator
kind create cluster
kubectl apply -k config/crd
# ... deploy the operator (see Stage 11), or run it locally with
#     `make run` against the kind cluster's kubeconfig ...
kubectl apply -f - <<'EOF'
apiVersion: apps.example.com/v1alpha1
kind: WebApp
metadata: {name: demo}
spec: {image: nginx:1.27, port: 80, replicas: 1}
EOF
kubectl get deploy,svc -l app.kubernetes.io/instance=demo
kubectl delete svc demo
kubectl get svc demo -w   # watch it come back with a new UID, unprompted
```

This repo's own [Stage 4](04-envtest.md) automates exactly this check as a
real test (`internal/controller/webapp_controller_test.go`'s "recreates a
deleted owned Service" case) - reading that test after doing this by hand
is the fastest way to see how the two connect.

Answer, without looking anything up:

- If `Owns()` weren't filtered, and this stage instead used
  `GenerationChangedPredicate`, would `kubectl delete svc demo` above still
  self-heal? (Careful - deleting an object *does* remove it from the
  cache, which is a different kind of event than a spec change. Which
  predicate would actually break, and which wouldn't?)
- `mutateDeployment` and `mutateService` never set an owner reference
  themselves - `ctrl.SetControllerReference` is called separately, inside
  `controllerutil.CreateOrUpdate`'s mutate closure. What would happen if
  you called `SetControllerReference` once, outside the closure, before
  the first `CreateOrUpdate`?

Next: [Stage 4 — envtest](04-envtest.md).
