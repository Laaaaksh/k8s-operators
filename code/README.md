# Code

One real, runnable operator: [`webapp-operator`](webapp-operator). Unlike a
series of small independent samples, this is a single Go module built up
incrementally across all twelve [curriculum](../curriculum) stages - there's
no per-stage snapshot directory. Each `curriculum/NN-*.md` file names the
specific files and functions it added or changed; that's your map of what
belongs to which concept.

## What it is

A `WebApp` custom resource: given an image, port, and replica count (or, as
of [Stage 10](../curriculum/10-versioning.md)'s `v1beta1`, a min/max
scaling range), the operator manages a Deployment and Service, tracks a
shared hostname registry with finalizer-backed cleanup, validates and
defaults input via admission webhooks, supports leader-election HA, runs
under an audited least-privilege RBAC role, exposes a custom Prometheus
metric alongside controller-runtime's built-in ones, and ships as both
Kustomize manifests and a Helm chart.

It is not a toy that only compiles - see "Verified" below.

## Layout

```
api/v1alpha1/        the original CRD version - a spoke, converting through v1beta1
api/v1beta1/          the current CRD version (storage + conversion hub)
internal/controller/  the reconciler: Deployment/Service, status, metrics
internal/registry/     the shared hostname registry, and why it needs a finalizer
internal/webhook/v1beta1/  defaulting, validating, and conversion webhooks
internal/leaderelection/   a test proving --leader-elect actually fails over
cmd/main.go            wiring: manager options, flags, webhook/controller registration
config/                Kustomize source (CRDs, RBAC, manager, webhook, cert-manager)
dist/install.yaml       config/default, flattened - `kubectl apply -f` this
dist/chart/             a Helm chart generated from the same Kustomize source
test/manifests/         proves dist/install.yaml is accepted by a real API server
test/e2e/                a full kind-cluster lifecycle test, including live conversion
```

## Building and testing

```bash
cd code/webapp-operator
make test        # unit + envtest integration tests - no cluster, ~20s
make lint         # golangci-lint
make test-e2e     # creates its own kind cluster, deploys, tests, tears down
```

See [Stage 0](../curriculum/00-toolchain-setup.md) for what each of these
needs installed.

## Verified

Every claim this curriculum makes about this operator's behavior was
checked, not assumed:

- **`make test`**: 81.0% / 76.4% / 85.7% statement coverage across
  `internal/controller`, `internal/registry`, `internal/webhook/v1beta1`,
  all passing, run 2026-08-30 against `envtest` (Kubernetes 1.36.2
  binaries, `controller-runtime` v0.24.1).
- **`make test-e2e`**: a full `kind` v0.33.0 cluster (Kubernetes v1.37.0),
  cert-manager v1.21.1 installed, the operator image built and deployed via
  Kustomize, leader election contested for real, and a WebApp created as
  `v1alpha1`, read back as `v1beta1` through the live HTTPS conversion
  webhook, reconciled into a real Deployment and Service, then deleted and
  confirmed fully cleaned up - 7 of 7 specs passing, run 2026-08-30.
- **Manual verification on the same real cluster**: `kubectl auth can-i`
  against the operator's actual deployed ServiceAccount (confirming the
  [least-privilege RBAC](../curriculum/08-least-privilege-rbac.md) claims),
  and the exact validating-webhook rejection messages quoted in
  [Stage 6](../curriculum/06-webhooks.md) and
  [Stage 10](../curriculum/10-versioning.md).
- **`helm lint` / `helm template`**: 0 issues, against Helm v4.2.4.
- **A real bug found this way**: `internal/registry`'s original
  cached-client race ([`resources/common-pitfalls.md`](../resources/common-pitfalls.md))
  passed hundreds of envtest runs and failed on the first real-cluster
  reconcile - found, fixed, and left documented rather than quietly
  corrected, because the failure mode is the actual lesson.

None of this is asserted from a description of what *should* happen if the
code were correct - every number above came from a command that was
actually run. If you re-run any of it and get a different result, that's
worth a bug report (see [`CONTRIBUTING.md`](../CONTRIBUTING.md)) - the
version table in [`resources/curated-resources.md`](../resources/curated-resources.md)
is where to start checking why.
