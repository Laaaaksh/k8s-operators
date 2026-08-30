# Stage 2 — Reconciler & status

**You'll be able to:** write a reconciler that converges a WebApp's Spec
into a real Deployment, and report truthful status via Conditions - not just
"reconciled without error," but "here's specifically what's true right now."

**Time:** 3–4 hours.

**Build:** `internal/controller/webapp_controller.go`,
`internal/controller/deployment.go`, and `internal/controller/status.go`.

## Read, in this order

1. [Kubebuilder book: controller implementation](https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html) - the shape of a `Reconcile` function, `Status().Update()`. Doesn't cover finalizers (that's [Stage 5](05-finalizers.md)) or the watch mechanics behind why `Reconcile` gets called at all (that's [Stage 3](03-watches-and-ownership.md)) - it's scoped to the loop body, which is this stage's scope too.
2. [Kubernetes API conventions: typical status properties](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties) - the `Conditions` pattern every controller-adjacent Kubernetes API uses, including the built-in ones. This repo's `Available`/`Progressing`/`Degraded` vocabulary is taken directly from here, not invented.
3. [`metav1.Condition` reference](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition) - the actual Go type and its `SetStatusCondition` helper.
4. [kstatus conventions](https://github.com/kubernetes-sigs/cli-utils/blob/master/pkg/kstatus/README.md) - a useful second read on how tools like `kubectl wait` and `kstatus` itself infer "is this healthy" generically from Conditions. Not required to build this stage, but explains why the *polarity* of condition types (`Available: True` is good, `Degraded: True` is bad - never invert that) is a convention worth following exactly, not a style preference.

## Idempotency is the whole game

Every `Reconcile` call has to be safe to run twice in a row with no change
in cluster state and produce no change in output - because it *will* run
more times than you'd naively expect (see [Stage 3](03-watches-and-ownership.md)
for exactly why), and because retries after a transient error are how this
loop recovers, not an edge case. The way to get that property is to never
phrase a reconciler step as "do this action" - phrase it as "make this look
like that":

```go
op, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
    mutateDeployment(dep, webapp)
    return ctrl.SetControllerReference(webapp, dep, r.Scheme)
})
```

`controllerutil.CreateOrUpdate` fetches the object if it exists, calls your
mutate function, and only issues an `Update` if something actually changed -
running it against an already-correct Deployment is a no-op, not a
no-op-that-still-writes. `internal/controller/deployment.go` splits this
further: `desiredDeployment` builds a bare object, `mutateDeployment` sets
every field this controller owns, and never touches a field it doesn't -
see that file's comment on why "own exactly the fields you set, and no
more" is what keeps this controller from fighting anything else that
touches the same Deployment.

## Status: a phase for humans, Conditions for machines

`internal/controller/status.go`'s `applyDeploymentStatus` computes both a
coarse `Phase` (`Pending`/`Progressing`/`Available`/`Degraded` - what
`kubectl get webapp` shows in one column) and the `Conditions` slice that's
the actual source of truth Phase is derived from, never set independently.
Two details worth noticing:

- It uses `meta.SetStatusCondition`, never `webapp.Status.Conditions =
  []metav1.Condition{...}` from scratch. `SetStatusCondition` updates a
  condition in place by `Type` and only bumps `LastTransitionTime` when the
  `Status` actually changes - rebuilding the slice every reconcile would
  reset that timestamp on every single pass, which breaks anything
  computing "how long has this been true" (alerting rules, `kubectl get`'s
  age column, a human reading the object).
- `wantReplicas == 0` is its own branch, reporting `Available` (not
  `Degraded`) with zero ready replicas. A WebApp scaled to zero on purpose
  isn't broken - conflating "off" with "broken" is exactly the kind of
  status bug that trains people to ignore alerts.

## Checkpoint

Run `go test ./internal/controller/...` (from `code/webapp-operator` - the
Stage 4 envtest suite already exists and exercises this stage's code, since
this repo is one project, not per-stage snapshots; you're reading its
tests, not writing your own yet). Then answer:

- `applyDeploymentStatus` takes `dep *appsv1.Deployment` and can be called
  with a `nil` Deployment. When does that happen, and why is that a real
  code path rather than a bug?
- Why does `Reconcile` return `ctrl.Result{}, reconcileErr` at the end
  instead of `ctrl.Result{}, nil` even when a later step (the status
  update) succeeded? What would silently swallowing that error cost you?
- `mutateDeployment` never sets `dep.Spec.Strategy`. If a cluster operator
  manually edits that field on the running Deployment, what happens to
  their edit the next time this controller reconciles? Why is that the
  right behavior for a field the controller doesn't own?

Next: [Stage 3 — Watches & ownership](03-watches-and-ownership.md).
