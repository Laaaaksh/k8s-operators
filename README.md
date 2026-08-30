<div align="center">

# k8s-operators

**A sequenced path from "I can write a reconcile loop" to "I can ship a
production-shaped operator" — one real operator, built incrementally,
tested against a real API server and a real cluster at every stage.**

[![CI](https://github.com/Laaaaksh/k8s-operators/actions/workflows/ci.yml/badge.svg)](https://github.com/Laaaaksh/k8s-operators/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.37-326CE5?logo=kubernetes&logoColor=white)](curriculum/00-toolchain-setup.md)

**[Curriculum](curriculum/README.md) • [Code](code/webapp-operator) • [Resources](resources/curated-resources.md) • [Common pitfalls](resources/common-pitfalls.md) • [Contributing](CONTRIBUTING.md) • [License](LICENSE)**

</div>

## What this is

Kubernetes operator material isn't scarce - the Kubebuilder book has the
pieces (webhooks, finalizers, envtest, CRD versioning), NVIDIA-scale
production teams give conference talks about running thousands of them, and
a dozen "awesome-operator" link lists exist. What's missing is the **order**:
the Kubebuilder book's own guided build (the CronJob tutorial) walks
CRD → controller → webhooks → tests, then stops - leader election, RBAC
design, multi-version conversion, and observability each live in their own
disconnected reference page or separate tutorial, never assembled into one
build. A 2026 SEO blog series patching this piecemeal, one post per missing
topic, is the visible symptom. An unordered list of 200 links is the problem
this repo exists to not be.

This repo is twelve sequenced stages, each with:

- **What you'll be able to do** at the end of it, stated concretely.
- **What to read**, in order - a handful of things, not a pile, with the
  reasoning in [`resources/curated-resources.md`](resources/curated-resources.md).
- **One real, cumulative Go operator** in [`code/webapp-operator`](code/webapp-operator) -
  not per-stage toy samples. CRD design, a reconciler, owned-resource
  watches, envtest integration tests, finalizers, admission webhooks with
  cert-manager, leader election, an audited least-privilege RBAC role, a
  custom Prometheus metric, a breaking CRD schema change behind a real
  conversion webhook, and Kustomize/Helm packaging - twelve stages of one
  build, not twelve disconnected demos.
- **Checkpoint questions**, several with a real command to run and a real
  thing to observe against a live cluster - not just prose to nod along
  with.

Start at [`curriculum/README.md`](curriculum/README.md).

## What this doesn't cover

This builds one namespaced CRD with a Deployment/Service-shaped
reconciler. It does not cover: multi-cluster or fleet-scale operator
patterns, building a real autoscaler (the `v1beta1` API's
`scaling.{min,max}Replicas` fields exist to be read by one, but this
operator doesn't implement one), OLM bundle authoring or OperatorHub
publishing (OLM v0 is in maintenance mode and OLM v1 has no settled
migration story as of this writing - see
[Stage 11](curriculum/11-packaging.md)'s reasoning, not a recommendation to
build against either yet), CEL-based `x-kubernetes-validations` as an
alternative to the webhook validation this repo builds by hand (a genuinely
current direction - see [Stage 6](curriculum/06-webhooks.md)'s note on
where CEL now covers what a webhook used to be required for, and where it
still doesn't), or performance/scale testing a controller against thousands
of live objects. [Stage 11](curriculum/11-packaging.md) covers where this
sits in the current ecosystem and what to reach for past this repo's scope.

## Verified, not asserted

**Every claim this curriculum makes about the operator's behavior was
checked against a real system, not written from what should theoretically
happen.** `make test` runs the full unit + [envtest](curriculum/04-envtest.md)
suite against a real (if kubelet-less) `kube-apiserver` - no cluster
required, 81–86% statement coverage across the reconciler, registry, and
webhook packages. Beyond that, this repo was built and re-verified against
a real `kind` v0.33.0 cluster running Kubernetes v1.37.0 with cert-manager
v1.21.1 installed: leader election contested for real between two live
managers, the live HTTPS conversion webhook exercised end-to-end (not just
its Go functions), RBAC claims checked with `kubectl auth can-i` against
the operator's actual deployed ServiceAccount, and every validating-webhook
rejection message quoted in the curriculum copied verbatim from real
`kubectl` output. `make test-e2e` automates that same cluster lifecycle in
CI on every push. One real bug - a cached-client race, found only by
running against a real cluster after passing hundreds of envtest runs - is
documented rather than quietly fixed and forgotten; see
[`resources/common-pitfalls.md`](resources/common-pitfalls.md). Full detail
in [`code/README.md`](code/README.md#verified).

## Repository layout

```
curriculum/   12 sequenced stages - the path itself
code/         one real, cumulative Go operator (code/webapp-operator)
resources/    honest, dated curation of everything external cited above
```

## Contributing

Contributions are welcome - a wrong claim, a stale "current" statement, a
dead link, a version this repo hasn't been checked against yet. See
[CONTRIBUTING.md](CONTRIBUTING.md). Please read
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) first.

## Security

Found a security issue? See [SECURITY.md](SECURITY.md) - please don't open
a public issue for it.

## License

Apache 2.0 - see [LICENSE](LICENSE). (This repo's curriculum and docs are
under the same license as the operator code, matching Kubernetes-ecosystem
convention - `kubebuilder`, `controller-runtime`, and Kubernetes itself are
all Apache 2.0.)
