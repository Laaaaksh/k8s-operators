# Curated resources

Every entry here was actually opened and checked at the time of writing
(2026-08-30) - not recalled from memory. Dates are the resource's own
stated publish/update date where one exists, or the version it's current
against. Where something has aged, that is said plainly, along with what to
read instead or alongside it. Curriculum stages link the specific entries
relevant to them; this page is the full, browsable list with the reasoning
behind each recommendation.

If a link here breaks, or you find something better, please open an issue
using the "Resource suggestion" template - see
[`CONTRIBUTING.md`](../CONTRIBUTING.md).

## Pinned versions this repo was built and verified against

| Component | Version | Verified from |
|---|---|---|
| Kubernetes (cluster) | v1.37.0 (GA, 2026-08-26) | [kubernetes/kubernetes releases](https://github.com/kubernetes/kubernetes/releases); confirmed on a real kind cluster |
| kubebuilder CLI | v4.15.0 (2026-06-15) | [kubernetes-sigs/kubebuilder releases](https://github.com/kubernetes-sigs/kubebuilder/releases) |
| controller-runtime (and `envtest`) | v0.24.1 (2026-05-12) | [kubernetes-sigs/controller-runtime releases](https://github.com/kubernetes-sigs/controller-runtime/releases); `go.mod` |
| controller-gen | v0.21.0 (2026-05-06) | [kubernetes-sigs/controller-tools releases](https://github.com/kubernetes-sigs/controller-tools/releases) |
| client-go / apimachinery | v0.36.0 | `go.mod` - see [Stage 0](../curriculum/00-toolchain-setup.md) for why this trails the cluster by one minor version deliberately |
| cert-manager | v1.21.1 (2026-07-29) | [cert-manager/cert-manager releases](https://github.com/cert-manager/cert-manager/releases); installed and exercised on a real kind cluster |
| kind | v0.33.0 (2026-08-26) | [kubernetes-sigs/kind releases](https://github.com/kubernetes-sigs/kind/releases) |
| Kustomize | v5.8.1 (2026-02-09) | [kubernetes-sigs/kustomize releases](https://github.com/kubernetes-sigs/kustomize/releases); matches `kubectl` 1.37's vendored version |
| Helm | v4.2.4 (2026-08-13) | [helm/helm releases](https://github.com/helm/helm/releases) - v3.21.4 (2026-08-14) is also still maintained in parallel; this repo's chart was verified against v4.2.4 |

## Official documentation, by curriculum stage

| Stage | Resource | What it covers | Verdict |
|---|---|---|---|
| 0 | [Kubebuilder Quick Start](https://book.kubebuilder.io/quick-start.html) | Toolchain setup: Go, Docker, kubectl, cluster access | Current |
| 1 | [Kubernetes: Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) | The control-loop mental model | Current (last modified 2024-09-01, concept is stable) |
| 1 | [Kubernetes: Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) | CRDs and OpenAPI validation basics | Current |
| 1 | [Kubebuilder book: API design](https://book.kubebuilder.io/cronjob-tutorial/api-design.html) | Designing a CRD's Go types, validation/default markers | Current |
| 1 | [Kubebuilder book: marker reference](https://book.kubebuilder.io/reference/markers.html) | Full `+kubebuilder:*` marker vocabulary | Current |
| 2 | [Kubebuilder book: controller implementation](https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html) | Reconciler basics, `Status().Update()` | Current - doesn't cover finalizers ([Stage 5](../curriculum/05-finalizers.md)) or watch mechanics ([Stage 3](../curriculum/03-watches-and-ownership.md)) |
| 2 | [Kubernetes API conventions: typical status properties](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties) | The Conditions pattern | Current |
| 2 | [`metav1.Condition` reference](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition) | The Go type behind Conditions | Current (pinned to v0.36.0 in this repo's dependency graph) |
| 2 | [kstatus README](https://github.com/kubernetes-sigs/cli-utils/blob/master/pkg/kstatus/README.md) | How tools infer health generically from Conditions | Current |
| 3 | [controller-runtime: `pkg/builder`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/builder) | `Owns()`/`Watches()` | Current, v0.24.1 |
| 3 | [controller-runtime: `pkg/predicate`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/predicate) | Event-filtering predicates | Current, v0.24.1 |
| 3 | [Kubebuilder book: watching resources](https://book.kubebuilder.io/reference/watching-resources.html) | Owned/secondary resources, requeue behavior | Current |
| 3 | [Kubernetes blog: "How the controller-runtime Cache Actually Works"](https://kubernetes.io/blog/2026/07/29/controller-runtime-cache-explained/) | The cache/informer mechanics under `Owns`/`Watches` - fills a real gap the reference docs above under-explain | Current, published 2026-07-29 (the article itself notes it was revised post-publication for accuracy) |
| 4 | [Kubebuilder book: envtest](https://book.kubebuilder.io/reference/envtest.html) | envtest integration testing | Current |
| 4 | [controller-runtime: `pkg/envtest`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest) | The `Environment` API | Current, v0.24.1 |
| 4 | [`setup-envtest`](https://github.com/kubernetes-sigs/controller-runtime/tree/main/tools/setup-envtest) | The CLI that downloads pinned envtest binaries | Current |
| 5 | [Kubernetes: finalizers concept](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/) | Finalizer mechanics, `deletionTimestamp` | Current (last modified 2025-04-27) |
| 5 | [Kubebuilder book: using finalizers](https://book.kubebuilder.io/reference/using-finalizers.html) | The reconciler-side finalizer pattern | Current |
| 6 | [Kubernetes: extensible admission controllers](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) | Validating/mutating webhook configuration, `matchPolicy` | Current |
| 6 | [Kubebuilder book: webhook implementation](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html) | `CustomDefaulter`/`CustomValidator` | Current - requires controller-runtime v0.21+; this repo is on v0.24.1 |
| 6 | [Kubebuilder book: cert-manager](https://book.kubebuilder.io/cronjob-tutorial/cert-manager.html) | Wiring a webhook's TLS via cert-manager | Current |
| 6 | ~~`cert-manager.io/docs/usage/webhook/`~~ | An older cert-manager+webhook usage page | **Dead - confirmed 404.** Use the CA Injector page below instead. |
| 6 | [cert-manager: CA Injector](https://cert-manager.io/docs/concepts/ca-injector/) | How `caBundle` gets injected into webhook configs automatically | Current - replaces the dead page above |
| 6 | [Kubernetes: CRD validation rules (CEL)](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules) | `x-kubernetes-validations`, cross-field and transition (`oldSelf`) rules, and the explicit "no cross-object validation" boundary | Current - feature GA since Kubernetes v1.29. See [Stage 6](../curriculum/06-webhooks.md)'s note on when this replaces a webhook and when it doesn't. |
| 6 | [Kubebuilder book: `XValidation` marker](https://book.kubebuilder.io/reference/markers/crd-validation) | The `+kubebuilder:validation:XValidation` marker that generates `x-kubernetes-validations` | Current |
| 7 | [controller-runtime: `manager.Options`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Options) | `LeaderElection*` fields | Current, v0.24.1 |
| 7 | [Kubernetes: Lease objects](https://kubernetes.io/docs/concepts/architecture/leases/) | The primitive leader election is built on | Current (last modified 2026-03-31) |
| 8 | [Kubebuilder book: RBAC markers](https://book.kubebuilder.io/reference/markers/rbac.html) | `+kubebuilder:rbac` marker syntax | Current, but reference-only - no tutorial walks through auditing a real role, which is what [Stage 8](../curriculum/08-least-privilege-rbac.md) exists to do |
| 8 | [Kubernetes: RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) | Role/ClusterRole/RoleBinding fundamentals | Current |
| 9 | [Kubebuilder book: metrics](https://book.kubebuilder.io/reference/metrics.html) | Built-in Prometheus metrics, adding custom ones | Current, actively version-tracked against recent kubebuilder releases |
| 9 | [controller-runtime: `pkg/metrics`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/metrics) | The shared metrics registry | Current, v0.24.1 |
| 9 | [Prometheus Operator: design / `ServiceMonitor`](https://prometheus-operator.dev/docs/getting-started/design/) | How Prometheus Operator discovers a metrics endpoint | Current |
| 9 | [Kubernetes: Event v1 API reference](https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/) | The `events.k8s.io/v1` schema | Current - there's no separate Events *concept* page any more; this API reference is the current source |
| 9 | [`net/http/pprof`](https://pkg.go.dev/net/http/pprof) | Profiling endpoints | Current, stdlib |
| 10 | [Kubernetes: CRD versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/) | Storage version, conversion strategies | Current |
| 10 | [Kubebuilder book: multi-version API tutorial](https://book.kubebuilder.io/multiversion-tutorial/tutorial.html) | The full hub-and-spoke conversion tutorial | Current - genuinely a separate tutorial from the CronJob build [Stage 6](../curriculum/06-webhooks.md) draws on, never integrated with it |
| 10 | [Kubebuilder book: conversion](https://book.kubebuilder.io/multiversion-tutorial/conversion.html) | `conversion.Hub`/`conversion.Convertible` | Current |
| 11 | [Helm: chart best practices](https://helm.sh/docs/chart_best_practices/) | Chart conventions including CRDs and RBAC | Doc-versioned to Helm v4.2.4 |

## Confirmed dead or superseded, and what this repo uses instead

| Was | Status | Use instead |
|---|---|---|
| `cert-manager.io/docs/usage/webhook/` | 404, confirmed 2026-08-30 | [cert-manager: CA Injector](https://cert-manager.io/docs/concepts/ca-injector/) |
| `sdk.operatorframework.io` best-practices page's "OLM is the standard" framing | Page itself dated 2023-06-21 - stale relative to OLM's 2026 v0/v1 transition | [Stage 11](../curriculum/11-packaging.md)'s OLM section, sourced from OLM's own repos directly |

## Books

| Resource | What it covers | Current as of | Verdict |
|---|---|---|---|
| ~~*Programming Kubernetes* (O'Reilly, 2019) - Burns & Beda~~ | client-go, informers, controllers/operators | 2019, no 2nd edition | **Aged - background context only.** Predates controller-runtime's Builder API maturity, typed `metav1.Condition` conventions, and the CRD-v1-only era every stage of this repo assumes. Reader reviews as recently as August 2026 independently describe it as dated. Use [book.kubebuilder.io](https://book.kubebuilder.io) instead - it's actively version-tracked against current releases. |

## Blog, tutorial series, and video

| Resource | What it covers | Date | Verdict |
|---|---|---|---|
| [golinuxcloud: Kubernetes Operator Tutorial series](https://www.golinuxcloud.com/kubernetes-operator-controller-runtime-go/) (foundations, [leader election](https://www.golinuxcloud.com/operator-leader-election-explained/), [webhooks](https://www.golinuxcloud.com/kubernetes-mutating-validating-webhooks-operators/), and more, one linked table of contents) | CRDs → webhooks → internals → HA/metrics → OLM → Helm | Self-dated June 2026, author Deepak Prasad | Current, and genuinely a linked, sequenced series rather than scattered posts - the closest thing to this repo's shape that already existed. Worth reading per-stage alongside the official docs above, not instead of them - it's a good second explanation, not the primary source this repo builds its claims from. |
| [`iximiuz/client-go-examples`](https://github.com/iximiuz/client-go-examples) | Raw client-go primitives (informers, dynamic client, workqueues) that kubebuilder/controller-runtime wrap for you | 1,123 stars, last commit 2025-12-12 | Aged-but-useful - not actively pushed monthly, but not abandoned either. A good "what's actually happening underneath `Owns()`" supplement once you've finished this curriculum, explicitly a client-go primer rather than an operator-pattern resource. |
| [Kubernetes blog: "How the controller-runtime Cache Actually Works"](https://kubernetes.io/blog/2026/07/29/controller-runtime-cache-explained/) | Cache-vs-live-API mental model, `Get()`/`List()` semantics, `APIReader` | 2026-07-29 | Current - directly relevant to [Stage 4](../curriculum/04-envtest.md)'s cached-client race. |
| [dev.to/piyushjajoo: "Kubernetes Operators: A Deep Dive into the Internals"](https://dev.to/piyushjajoo/kubernetes-operators-a-deep-dive-into-the-internals-221m) | Control-theory framing, DeltaFIFO/Reflector, resourceVersion CAS, rate-limiter composition | Published 2024-02-25 | Current despite its age - checked against controller-runtime/CRD-v1 conventions as of 2026, still accurate. Adds real depth on *why* the mechanisms work the way they do, past what this curriculum's stages cover. |
| [KubeCon EU 2026 - Saxo Bank keynote](https://www.youtube.com/watch?v=Ue3uXQc0eTo) | Operators + GitOps as a production intent-to-infrastructure layer (1,800+ automated operations, Saxo Bank) | 2026-03-25 | A five-minute keynote vignette, not a technical deep-dive - cited here for the demand/production-seriousness signal, not as a how-to resource. |

## What's deliberately not repeated here

Everything Kustomize/Helm/OLM-specific is in
[`curriculum/11-packaging.md`](../curriculum/11-packaging.md), including the
full reasoning on OLM's 2026 maintenance-mode status, rather than
duplicated on this page.
