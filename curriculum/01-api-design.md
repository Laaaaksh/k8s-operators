# Stage 1 — API design

**You'll be able to:** design a Custom Resource Definition's schema so the
API server itself rejects invalid input, before any of your Go code runs -
and explain why that split (schema validation vs. code) matters.

**Time:** 2–3 hours.

**Build:** `api/v1alpha1/webapp_types.go` and the CRD it generates.

## The mental model first

A Kubernetes controller is a control loop: something watches the difference
between the state you asked for (Spec) and the state that actually exists,
and keeps nudging reality toward what you asked for. That's true whether
"something" is the built-in Deployment controller or code you write. Read
[Kubernetes' own "Controller" concept page](https://kubernetes.io/docs/concepts/architecture/controller/)
first if that loop isn't already second nature - everything else in this
curriculum assumes it.

A Custom Resource Definition (CRD) is how you teach the API server a new
Kind - `WebApp`, in this repo's case - complete with an OpenAPI schema the
API server validates against on every `create`/`update`, before your
controller ever sees the object. [Kubernetes' CRD concept
page](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
covers the mechanism; kubebuilder's own [API design
walkthrough](https://book.kubebuilder.io/cronjob-tutorial/api-design.html)
and [marker reference](https://book.kubebuilder.io/reference/markers.html)
cover the Go-struct-to-schema mapping this repo uses throughout.

## Read, in this order

1. [Kubernetes: "Controllers" concept](https://kubernetes.io/docs/concepts/architecture/controller/) - the reconcile-loop mental model.
2. [Kubernetes: "Custom Resources" concept](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) - what a CRD actually is.
3. [Kubebuilder book: API design](https://book.kubebuilder.io/cronjob-tutorial/api-design.html) - designing a CRD's Go types.
4. [Kubebuilder book: marker reference](https://book.kubebuilder.io/reference/markers.html) - the full `+kubebuilder:validation:*` vocabulary; you'll use this as a reference for the rest of the curriculum, not read start to end.

## In this repo

`api/v1alpha1/webapp_types.go` defines `WebApp`: a stateless HTTP
application the operator runs as a Deployment + Service. Every field is
validated by an OpenAPI schema marker, not checked in Go later:

```go
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
```

Run `make manifests` (from `code/webapp-operator`) and look at
`config/crd/bases/apps.example.com_webapps.yaml` - `controller-gen` turned
those two field comments plus their markers into real OpenAPI v3 schema:
`minLength: 1`, and `minimum: 1` / `maximum: 65535` with a `default: 8080`.
Try applying a WebApp with `port: 99999` or an empty `image` against a real
cluster (or via `envtest`, [Stage 4](04-envtest.md)'s tool) - the API server
rejects it before your reconciler is ever invoked. That's the split worth
internalizing: **anything expressible as "is this value shaped correctly"
belongs in the schema, not in code** - it's enforced earlier, for every
client (`kubectl`, a CI pipeline, another controller), not just the one you
remember to write a check in.

One field this stage's schema genuinely cannot handle, and one it could in
principle but this repo implements elsewhere for teaching reasons - both
are the seam [Stage 6](06-webhooks.md) exists to cover in full:

- `hostName` needs a value *derived from* the object (its name and
  namespace) when left empty. A schema default can only ever be a fixed
  literal - `+kubebuilder:default=8080` works because 8080 never changes;
  there is no marker for "compute this from `.metadata.name`," and CEL
  validation rules (see [Stage 6](06-webhooks.md)) can *validate*, not
  *set*, a field. This one is genuinely webhook-only.
- Whether `hostName` is *immutable after creation* - a schema on its own
  can't compare an object's new value to its old one, but Kubernetes' CEL
  transition rules (`oldSelf`, GA since v1.29) now can. This repo still
  implements the check as webhook code, to teach the mechanism - [Stage 6](06-webhooks.md)
  covers both the webhook version and the current, schema-only
  alternative, and is explicit about which one production code should
  actually prefer.

`WebAppStatus` also gets its shape here (`Phase`, `ReadyReplicas`,
`ObservedGeneration`, `Conditions`) even though nothing populates it until
[Stage 2](02-reconciler-and-status.md) - designing the full Spec/Status
split up front, even before the reconciler exists to fill Status in, is
deliberate: it's the schema every later stage's data has to fit into.

## Checkpoint

Answer these without looking anything up:

- What's the difference between `+kubebuilder:validation:Minimum` on
  `Port` and a check inside `Reconcile` that returns an error if
  `webapp.Spec.Port < 1`? Which one runs first, and which one a `kubectl
  apply` from a totally different tool (not this operator) still benefits
  from?
- Why can't `hostName`'s "derive a default from the object's own name" live
  in `+kubebuilder:default=`? What's fundamentally different about that
  default versus `Port`'s `8080`?
- `WebAppStatus` has an `ObservedGeneration` field and Kubernetes objects
  separately track `.metadata.generation`. What's the difference between
  the two, and why does a status need to say "which generation did I last
  actually look at"?

Next: [Stage 2 — Reconciler & status](02-reconciler-and-status.md).
