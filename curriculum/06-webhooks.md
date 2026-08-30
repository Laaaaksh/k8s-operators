# Stage 6 — Webhooks

**You'll be able to:** write defaulting and validating admission webhooks
for rules a CRD schema can't express, and wire up the TLS certificates they
need via cert-manager.

**Time:** 3–4 hours.

**Build:** `internal/webhook/v1beta1/webapp_webhook.go`, plus
`config/certmanager/` and the cert-manager kustomize wiring in
`config/default/`.

## A note on where this stage's code actually lives

This repo is one project, built up across all twelve stages, not a
snapshot per stage - see the root `README.md`. When this stage was first
built, its webhooks lived on `v1alpha1`, the only version that existed yet.
[Stage 10](10-versioning.md) later introduces `v1beta1` as the API's hub
version, and at that point the defaulting/validating webhooks moved there
too - because once a hub version exists, that's the one version every
admission request eventually resolves to (`matchPolicy: Equivalent`, more
below), so it's the only one that needs its own `CustomDefaulter`/
`CustomValidator`. `v1alpha1` today has no webhook registration of its own.

That migration is itself a real lesson worth having up front: **the
concepts in this stage - what a schema can't express, `CustomDefaulter`,
`CustomValidator`, cert-manager wiring - don't change** when the webhooks
move to a different version's Go package. Read this stage using
`internal/webhook/v1beta1/webapp_webhook.go` as the reference implementation
regardless of which version you're picturing; come back to this note after
[Stage 10](10-versioning.md) if the "why here and not `v1alpha1`" is still
unclear.

## Read, in this order

1. [Kubernetes: admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) - the concept: `ValidatingWebhookConfiguration`/`MutatingWebhookConfiguration`, `matchPolicy`, `failurePolicy`.
2. [Kubebuilder book: webhook implementation](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html) - `CustomDefaulter`/`CustomValidator`, the interfaces this repo's webhook implements. Requires controller-runtime v0.21+; this repo is on v0.24.1.
3. [Kubebuilder book: cert-manager](https://book.kubebuilder.io/cronjob-tutorial/cert-manager.html) - how kubebuilder wires a webhook server's TLS cert via cert-manager.
4. [cert-manager: CA Injector](https://cert-manager.io/docs/concepts/ca-injector/) - the piece that patches a webhook's `caBundle` automatically so the API server trusts the cert cert-manager issued. (The older `cert-manager.io/docs/usage/webhook/` page some tutorials still link is dead - use this one.)

## What a schema genuinely cannot express

[Stage 1](01-api-design.md) already drew this line for `hostName`'s
defaulting. The validating side has its own two examples in this repo,
one genuinely schema-inexpressible rule, and two more this repo builds as
webhook code to teach the mechanism - but which a modern CRD schema can, in
fact, now express without a webhook at all. Both are worth seeing, and the
line between them is worth being precise about.

**Genuinely schema-inexpressible: cross-object.** A WebApp can't be
created in the `webapp-system` namespace, because that namespace is
reserved for the registry ConfigMap ([Stage 5](05-finalizers.md)). A CRD
schema - including the CEL-based rules described below - validates one
object in isolation; it has no way to know about a *different* resource (a
reserved namespace name) at all. This one needs a webhook, full stop:

```go
if obj.Namespace == registry.Namespace {
    return nil, apierrors.NewInvalid(webAppGK, obj.Name, field.ErrorList{
        field.Forbidden(field.NewPath("metadata", "namespace"),
            fmt.Sprintf("%q is reserved for the WebApp registry...", registry.Namespace)),
    })
}
```

**Built as a webhook here, but schema-expressible today: immutability.**
`hostName` is immutable once set - changing it would silently repoint
whatever external system relied on that registry entry being stable. This
repo's `ValidateUpdate` checks it in Go, comparing the old and new object:

```go
if oldObj.Spec.HostName != "" && newObj.Spec.HostName != oldObj.Spec.HostName {
    return nil, apierrors.NewInvalid(webAppGK, newObj.Name, field.ErrorList{
        field.Forbidden(field.NewPath("spec", "hostName"),
            fmt.Sprintf("is immutable once set (was %q) - delete and recreate the WebApp instead", oldObj.Spec.HostName)),
    })
}
```

**Built as a webhook here, but schema-expressible today: cross-field.**
`v1beta1`'s `Scaling{MinReplicas, MaxReplicas}` ([Stage 10](10-versioning.md))
needs `minReplicas <= maxReplicas` - each field's own `Minimum`/`Maximum`
markers pass independently even when the relationship is violated (5 and 2
are each individually valid int32s), so `internal/webhook/v1beta1`'s
`validateScaling` checks the relationship in Go.

### Where CEL now covers this - and where it still doesn't

As of Kubernetes v1.29 (GA - stable well before this repo's v1.37.0),
CRDs support **CEL validation rules** embedded directly in the OpenAPI
schema, via `x-kubernetes-validations` -
kubebuilder's `+kubebuilder:validation:XValidation:rule="...",message="..."`
marker generates them. Both webhook-built rules above could be expressed
this way instead, with no webhook involved:

```go
// +kubebuilder:validation:XValidation:rule="self.scaling.minReplicas <= self.scaling.maxReplicas",message="minReplicas must be <= maxReplicas"
```

and, for immutability, a *transition rule* - one whose expression
references `oldSelf`, which only evaluates on update:

```go
// +kubebuilder:validation:XValidation:rule="self.hostName == oldSelf.hostName",message="hostName is immutable once set",optionalOldSelf=true
```

(`optionalOldSelf=true` is what makes this evaluate correctly the first
time `hostName` transitions from unset to set - see the [CRD validation
rules docs](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules)
for the exact semantics of `oldSelf` when a field was previously absent.)

**This repo deliberately implements both as webhook Go code anyway**, to
teach the webhook mechanism itself - real production code still uses
webhooks constantly, including for the one case above that CEL genuinely
cannot reach. But the current, narrower, correct guidance for anything
CEL *can* express: **prefer CEL.** It's enforced by the API server
directly with no extra network hop, no separate service to keep available
and TLS-provisioned, and no `failurePolicy` decision to make (see below).
Reserve a webhook for what CEL cannot do - cross-object checks like the
reserved-namespace rule, or logic too dynamic for CEL's expression
language to express cleanly. The Kubernetes docs are explicit that CEL
rules are "scoped to the current object: no cross-object or stateful
validation rules are supported" - that boundary is exactly where this
repo's one genuinely-necessary webhook rule lives.

## Verified live, verbatim

On a real kind cluster (v1.37.0, verified 2026-08-30), each of these was
rejected by the live webhook - not asserted, actually run:

```
$ kubectl apply -f bad-scaling.yaml
The WebApp "bad-scaling" is invalid: spec.scaling.minReplicas: Invalid value: 5: must be <= maxReplicas (2)

$ kubectl apply -f sneaky-namespace.yaml
The WebApp "sneaky" is invalid: metadata.namespace: Forbidden: "webapp-system" is reserved for the WebApp registry and cannot contain WebApp resources

$ kubectl patch webapp immuttest --type=merge -p '{"spec":{"hostName":"changed.example.com"}}'
The WebApp "immuttest" is invalid: spec.hostName: Forbidden: is immutable once set (was "original.example.com") - delete and recreate the WebApp instead
```

## cert-manager: why a webhook needs it at all

A webhook is an HTTPS server the API server calls into - which means the
API server needs to trust its certificate, and that cert needs rotating
before it expires. `config/certmanager/` defines a self-signed `Issuer`
and a `Certificate` for the webhook server; cert-manager's CA Injector
watches for that Certificate's Secret and patches the
`caBundle` field on every `MutatingWebhookConfiguration`/
`ValidatingWebhookConfiguration`/CRD-conversion-webhook that has the
`cert-manager.io/inject-ca-from` annotation - which is why
`config/default/kustomization.yaml`'s `replacements` block wires that
annotation onto each of them. You don't hand-manage a cert anywhere in
this flow; that's the point of using cert-manager instead of a
self-rolled cert script.

## Checkpoint

- `WebAppCustomValidator.ValidateDelete` exists (required by the
  `CustomValidator` interface) but always returns `nil, nil`. The webhook
  marker's `verbs=create;update` doesn't include `delete`. What would
  change if you added `delete` there and put real logic in
  `ValidateDelete`? What's a rule that would make sense to enforce there
  that this repo doesn't need (hint: think about what a validating-delete
  webhook is uniquely positioned to check versus a finalizer)?
- `failurePolicy=fail` is the default kubebuilder scaffolds for both
  webhooks in this repo. What happens to every `kubectl apply` of a WebApp
  if the webhook server pod is down and `failurePolicy` is `fail`? What
  would change with `failurePolicy=ignore`, and why is `fail` the safer
  default for a validating webhook specifically?

Next: [Stage 7 — Leader election](07-leader-election.md).
