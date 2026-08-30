# Stage 8 — Least-privilege RBAC

**You'll be able to:** audit `+kubebuilder:rbac` markers against what a
controller's code actually calls, and verify the result against a real
cluster instead of trusting the generated role.

**Time:** 1–2 hours.

**Build:** the RBAC marker block at the top of
`internal/controller/webapp_controller.go`, and what `make manifests`
generates from it into `config/rbac/role.yaml`.

## Read, in this order

1. [Kubebuilder book: RBAC markers](https://book.kubebuilder.io/reference/markers/rbac.html) - the marker vocabulary this stage audits.
2. [Kubernetes: RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) - `Role`/`ClusterRole`/`RoleBinding` fundamentals, if you need the concept refresher.

## The gap this stage closes

Kubebuilder's own scaffold defaults every RBAC marker for a resource a
controller touches to the full CRUD-plus-watch set:
`get;list;watch;create;update;patch;delete`. That's a reasonable starting
point and a bad place to stop - most controllers never call `Delete` on
half of what they're granted delete access to, and nothing in the default
build catches that. This is a flat *reference* page in the Kubebuilder
book (the marker syntax), never a tutorial chapter on auditing what you
actually need - exactly the "reference exists, sequencing doesn't" gap
this repo's whole [README](../README.md) is about.

## The audit method

Read every method on `WebAppReconciler`, list every client verb it
actually calls per resource type, and cross out anything the markers
grant that no code path uses. For this repo's reconciler, that produces:

| Resource | Verbs the code calls | Why not more |
|---|---|---|
| `webapps` | get, list, watch, update, patch | `Get` (fetch), watch/list (via the manager's `For()`), `Update` (finalizer add/remove). Never `Delete` - this controller never deletes its own primary resource, only `kubectl`/a client does. Never `Create` either - unusual to grant a controller `create` on its own CRD, and this one doesn't need to. |
| `webapps/status` | get, update, patch | `r.Status().Update()`, once per reconcile. |
| `webapps/finalizers` | update | Finalizer mutations technically go through the main resource's `update` for CRDs without a dedicated finalizers subresource, but kubebuilder scaffolds this defensively; harmless to keep. |
| `deployments` (`apps`) | get, list, watch, create, update, patch | `controllerutil.CreateOrUpdate`. **No `delete`** - Kubernetes' garbage collector deletes owned Deployments via `ownerReferences` ([Stage 3](03-watches-and-ownership.md)); that's a *different* actor (the garbage collector, running as part of `kube-controller-manager`) with its own permissions, not this ServiceAccount. |
| `services` | get, list, watch, create, update, patch | Same reasoning as `deployments`. |
| `configmaps` | get, list, watch, create, update, patch | `internal/registry`'s reads and mutations. No `delete` - the registry ConfigMap itself is never deleted, only its `.data` entries. |
| `namespaces` | get, create | `ensureNamespace` creates the registry namespace if missing, and never touches it again. |
| `events` | create, patch | Event recording ([Stage 9](09-observability.md)). |

The one most people don't predict: **no `delete` on `webapps`,
`deployments`, or `services`**, even though this controller very much
creates and updates all three. Deleting a *child* is Kubernetes' job via
ownership; deleting the *primary* resource is a client's job. A controller
that only ever converges state toward a Spec has no code path that needs
to delete either.

## Verified against a real cluster, not just read from the YAML

RBAC markers generating the *intended* role is one claim; the API server
actually *enforcing* it is a different, stronger one. On a real kind
cluster (v1.37.0, verified 2026-08-30), against the operator's real
deployed ServiceAccount:

```
$ kubectl auth can-i delete deployments \
    --as=system:serviceaccount:webapp-operator-system:webapp-operator-controller-manager -n default
no

$ kubectl auth can-i create deployments \
    --as=system:serviceaccount:webapp-operator-system:webapp-operator-controller-manager -n default
yes

$ kubectl auth can-i delete webapps.apps.example.com \
    --as=system:serviceaccount:webapp-operator-system:webapp-operator-controller-manager -n default
no
```

`kubectl auth can-i --as=<serviceaccount>` is the check to run after any
RBAC change - it answers the real question (what can this identity
actually do against a real API server) rather than the proxy question
(does this YAML look right).

## Checkpoint

- The audit table above says this controller never needs `delete` on
  `deployments`. Under what circumstance - a real one, not hypothetical -
  would you need to add it back? (Think about what happens if a WebApp's
  `spec.image` changes to something that requires recreating the
  Deployment rather than patching it in place - does this operator's
  `mutateDeployment` ever need to do that?)
- This repo's RBAC is a single `ClusterRole`, not a per-namespace `Role`,
  even though every resource it manages (`configmaps`, `namespaces`
  aside) is namespaced. Why? What would have to change about how the
  manager's cache is configured for a namespaced `Role` to be enough
  instead?
- Run `kubectl auth can-i --list --as=system:serviceaccount:webapp-operator-system:webapp-operator-controller-manager`
  against a real deployment of this operator. Is there anything in the
  output you can't immediately explain by pointing at a specific line of
  Go code that needs it?

Next: [Stage 9 — Observability](09-observability.md).
