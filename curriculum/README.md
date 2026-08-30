# The curriculum

Twelve stages, in order. Each one names what you'll be able to do at the end,
what to read or watch, roughly how long it takes, and what to build to prove
it stuck. Do them in order the first time through - later stages assume
earlier ones, and the operator you build is genuinely cumulative, not a set
of disconnected samples.

| Stage | You'll be able to... | Time | Build |
|---|---|---|---|
| [0 — Toolchain setup](00-toolchain-setup.md) | Get Go, kubebuilder, kind, envtest, and Docker working together | 30–60 min | `make test` runs green with no cluster |
| [1 — API design](01-api-design.md) | Design a CRD schema whose validation/defaulting rejects bad input before any of your code runs | 2–3 hr | `api/v1alpha1/webapp_types.go` |
| [2 — Reconciler & status](02-reconciler-and-status.md) | Write a reconciler that converges Spec to a Deployment, and report truthful status via Conditions | 3–4 hr | `internal/controller` (Deployment + status) |
| [3 — Watches & ownership](03-watches-and-ownership.md) | Explain owner references and predicates; make deleting a child self-heal with no polling | 2–3 hr | Add the Service + `Owns()` |
| [4 — envtest](04-envtest.md) | Write and run integration tests against a real API server, no cluster required | 2–3 hr | The full envtest suite, including a live self-heal test |
| [5 — Finalizers](05-finalizers.md) | Block deletion safely to clean up state Kubernetes' garbage collector can't reach | 2–3 hr | `internal/registry` + the finalizer path |
| [6 — Webhooks](06-webhooks.md) | Write defaulting/validating webhooks and wire cert-manager for their TLS | 3–4 hr | `internal/webhook/v1beta1` |
| [7 — Leader election](07-leader-election.md) | Explain what `--leader-elect` actually does, and prove failover with a test, not a guess | 1–2 hr | `internal/leaderelection` |
| [8 — Least-privilege RBAC](08-least-privilege-rbac.md) | Audit RBAC markers against real client calls instead of copy-pasting a scaffold | 1–2 hr | Prune the RBAC markers; verify with `kubectl auth can-i` |
| [9 — Observability](09-observability.md) | Add a domain-specific Prometheus metric, Kubernetes Events, and pprof - and know which of the three answers which question | 2–3 hr | `internal/controller/metrics.go` + Events |
| [10 — Versioning](10-versioning.md) | Ship a breaking CRD schema change with a conversion webhook, and state exactly what data it loses | 3–5 hr | `api/v1beta1` + hub/spoke conversion |
| [11 — Packaging](11-packaging.md) | Ship Kustomize manifests and a Helm chart, and know where OLM fits (and doesn't, yet) | 2–3 hr | `dist/install.yaml`, `dist/chart/` |

**Total: roughly 25–35 hours** of focused work, spread over however long
that takes you. There's no clock running.

## Prerequisites

You should be comfortable writing and debugging Go - reading a struct,
following an interface, understanding what a pointer receiver is for. You do
**not** need prior Kubernetes API, controller, or client-go experience.
Comfort running `kubectl` against any cluster helps but isn't required going
in - [Stage 0](00-toolchain-setup.md) covers what you need to get one.

## How each stage is structured

- **What to read or watch** - a short, sequenced list, not an unordered
  pile. Full annotated details (what's covered well, how current it is, and
  what's stale but still worth it) live in
  [`resources/curated-resources.md`](../resources/curated-resources.md) -
  each stage links the specific entries relevant to it.
- **In this repo** - which files in [`code/webapp-operator`](../code/webapp-operator)
  this stage added or changed, with the reasoning that doesn't fit in a code
  comment. The operator is one project built up across all twelve stages -
  there's no separate sample directory per stage, so this section is your
  map of what to look at.
- **Checkpoint** - a specific command to run and a specific thing to
  observe, or a question to answer without looking anything up. Not a quiz
  to pass - a way to notice what didn't actually land.

## If something stops you

[`resources/common-pitfalls.md`](../resources/common-pitfalls.md) collects
the specific things that stop most people starting out - toolchain and
version mismatches, a real cached-client race this repo's own registry
package shipped with (and how it was found and fixed), and the
misconceptions almost everyone starts with: reconcile vs. watch,
owned-resource garbage collection vs. finalizers, and why "it compiled" says
nothing about whether the RBAC is right. Check there before assuming you've
found a new problem.
