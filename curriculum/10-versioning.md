# Stage 10 — Versioning

**You'll be able to:** ship a genuinely breaking CRD schema change behind a
conversion webhook, keep old clients working, and state exactly what data
that conversion loses instead of discovering it in production.

**Time:** 3–5 hours.

**Build:** `api/v1beta1/` (the new hub version), `api/v1alpha1/webapp_conversion.go`
and `api/v1beta1/webapp_conversion.go` (the ConvertTo/ConvertFrom/Hub
implementations), and `internal/webhook/v1beta1/`'s conversion wiring.

## Read, in this order

1. [Kubernetes: CRD versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/) - storage versions, served versions, conversion strategies.
2. [Kubebuilder book: multi-version API tutorial](https://book.kubebuilder.io/multiversion-tutorial/tutorial.html) - the hub-and-spoke pattern this stage's code follows exactly.
3. [Kubebuilder book: conversion](https://book.kubebuilder.io/multiversion-tutorial/conversion.html) - the `conversion.Hub`/`conversion.Convertible` interfaces.

This is genuinely a second, separate tutorial in the Kubebuilder book from
the one [Stage 6](06-webhooks.md) draws on - it's never folded into the
guided CronJob build there. That gap (leader election, RBAC design, and
multi-version conversion each living in their own disconnected corner,
never assembled into one build) is exactly what this repo's `README`
describes as the reason it exists.

## The breaking change, and why it's real

`v1alpha1.WebAppSpec.Replicas` is a single `int32` - "run exactly N
replicas." `v1beta1` replaces it with:

```go
type ScalingSpec struct {
    MinReplicas int32 `json:"minReplicas"`
    MaxReplicas int32 `json:"maxReplicas"`
}
```

That's not a rename, it's a genuine shape change - the kind every
production API eventually needs (here: making room for a future
autoscaler this repo doesn't itself implement, see the root
[README](../README.md)'s "what this repo doesn't cover"). There is no
schema migration that turns one `int32` into a `{min, max}` pair without
someone deciding what the missing half should be. That decision is the
actual content of this stage - everything else is mechanism.

## Hub and spoke: exactly one version is the source of truth

`v1beta1.WebApp` is the **Hub** - the version every other served version
converts through, and (via `+kubebuilder:storageversion`) what's actually
persisted in etcd:

```go
// api/v1beta1/webapp_conversion.go
func (*WebApp) Hub() {}
```

`v1alpha1.WebApp` is a **spoke**, implementing `conversion.Convertible`:

```go
func (src *WebApp) ConvertTo(dstRaw conversion.Hub) error {
    dst := dstRaw.(*appsv1beta1.WebApp)
    ...
    dst.Spec.Scaling = appsv1beta1.ScalingSpec{
        MinReplicas: src.Spec.Replicas,
        MaxReplicas: src.Spec.Replicas,
    }
    return nil
}
```

Going up (`v1alpha1` → `v1beta1`), the single `Replicas` value becomes
both `MinReplicas` and `MaxReplicas` - a reasonable, lossless-in-this-direction
choice. Going down is where the real decision lives:

```go
func (dst *WebApp) ConvertFrom(srcRaw conversion.Hub) error {
    src := srcRaw.(*appsv1beta1.WebApp)
    ...
    dst.Spec.Replicas = src.Spec.Scaling.MinReplicas
    return nil
}
```

**`MaxReplicas` has nowhere to go in `v1alpha1` - converting down silently
drops it.** Round-trip a WebApp with `{min: 2, max: 10}` through
`v1alpha1` and back, and `max` comes back as `2`, not `10`. That's real,
observable data loss, and `internal/webhook/v1beta1/webapp_webhook_test.go`
asserts it directly rather than only describing it in a comment:

```go
It("documents the lossy direction: a widened v1beta1 range collapses to MinReplicas on the way down", func() {
    ...
    Expect(spoke.Spec.Replicas).To(Equal(int32(2)), "MaxReplicas has nowhere to go in v1alpha1")
    ...
    Expect(back.Spec.Scaling.MaxReplicas).To(Equal(int32(2)), "not the original 10")
})
```

There is no webhook rule that fixes this - a validating webhook rejects
*wrong* input; there's nothing wrong here, just an old, smaller shape
being asked to hold a newer, larger one. The only real answers are "state
the rule plainly, once, so both conversion directions agree with each
other" (done, in this file's own top comment) and "migrate clients off the
old version" (the actual long-term fix, same as any breaking API change).

## Verified live: the real HTTPS conversion, not just the Go functions

Everything above is exercised as a plain Go unit test with no HTTP
involved - `ConvertTo`/`ConvertFrom` called directly. That proves the
*logic*. `internal/webhook/v1beta1/webapp_webhook_test.go` also has one
test that proves the *wiring*: it writes a WebApp as `v1alpha1` through a
real envtest-backed client and reads the same object back as `v1beta1` -
if that succeeds, the API server genuinely called this operator's live
`/convert` endpoint over HTTPS mid-request.

This was also confirmed by hand, on a real kind cluster (v1.37.0, verified
2026-08-30) - the same object, fetched two ways:

```
$ kubectl apply -f - <<'EOF'
apiVersion: apps.example.com/v1alpha1
kind: WebApp
metadata: {name: demo}
spec: {image: nginx:1.27, port: 80, replicas: 2}
EOF

$ kubectl get webapp.v1alpha1.apps.example.com demo -o yaml | ...
spec:
  hostName: demo.default.apps.example.com
  image: nginx:1.27
  port: 80
  replicas: 2

$ kubectl get webapp.v1beta1.apps.example.com demo -o yaml | ...
spec:
  hostName: demo.default.apps.example.com
  image: nginx:1.27
  port: 80
  scaling:
    maxReplicas: 2
    minReplicas: 2
```

`kubectl get webapp.v1beta1.apps.example.com` (not
`kubectl get webapp.v1beta1` - kubectl's `TYPE.VERSION.GROUP` syntax needs
the full group) is the fastest way to see this for yourself.

## `matchPolicy: Equivalent`, and why webhooks only need to target the Hub

[Stage 6](06-webhooks.md)'s defaulting/validating webhooks are registered
only against `v1beta1`. A write submitted as `v1alpha1` still gets
defaulted and validated correctly, because the generated
`ValidatingWebhookConfiguration`/`MutatingWebhookConfiguration` use
`matchPolicy: Equivalent` (controller-runtime's default): the API server
converts the incoming object to whichever version a matching webhook rule
declares *before* invoking it. One webhook, on the Hub, covers every
served version - which is also why `v1alpha1` needed no
`internal/webhook/v1alpha1` package of its own once `v1beta1` existed;
see [Stage 6](06-webhooks.md)'s opening note.

## Checkpoint

- Run the live-conversion test yourself:
  `go test ./internal/webhook/... -run TestAPIs -v` (from
  `code/webapp-operator`) and find the spec in the output that creates a
  WebApp as `v1alpha1` and reads it back as `v1beta1`. What specifically
  would have to break for that test to fail - list two different failure
  modes (one in the conversion Go code, one in the webhook *registration*)
  that would each cause it to fail differently.
- If a third version, `v2alpha1`, needed to exist alongside `v1alpha1` and
  `v1beta1`, would it also implement `ConvertTo`/`ConvertFrom` against
  `v1alpha1`, or against the Hub? Why does hub-and-spoke scale to N
  versions without needing N² conversion functions?
- `+kubebuilder:storageversion` is on `v1beta1`, not `v1alpha1`. If you
  had *not* moved the storage version - kept `v1alpha1` as storage and
  added `v1beta1` as just another served, converted version - would the
  `MaxReplicas` data-loss problem above still exist? Would it be worse,
  better, or the same?

Next: [Stage 11 — Packaging](11-packaging.md).
