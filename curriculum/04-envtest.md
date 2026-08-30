# Stage 4 — envtest

**You'll be able to:** write and run integration tests against a real
Kubernetes API server - not a mock, not a fake client - with no cluster,
kubelet, or Docker daemon involved, and know exactly what that setup can
and can't prove.

**Time:** 2–3 hours.

**Build:** `internal/controller/suite_test.go`,
`internal/controller/webapp_controller_test.go`, and
`internal/registry/suite_test.go` - the tests behind every "Checkpoint" in
Stages 2 and 3.

## Read, in this order

1. [Kubebuilder book: envtest](https://book.kubebuilder.io/reference/envtest.html) - what it is and why it exists.
2. [controller-runtime: `pkg/envtest`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest) - the `Environment` type this repo's suite files construct directly.
3. [`setup-envtest`](https://github.com/kubernetes-sigs/controller-runtime/tree/main/tools/setup-envtest) - the CLI that downloads pinned `kube-apiserver`/`etcd` binaries; `make test` already wires this up, but it's worth knowing it's a separate, swappable tool.

## What envtest actually is

`envtest.Environment` starts a real `kube-apiserver` and `etcd` binary pair
- the exact ones Kubernetes ships, not a reimplementation - and gives you a
`*rest.Config` pointed at them. Every `Get`/`Create`/`Update`/`List`/`Watch`
your reconciler makes goes through the genuine API machinery: OpenAPI schema
validation, admission webhooks (if you register them, see below),
defaulting, the works.

What it does **not** run: a scheduler, a kubelet, or any built-in
controller (Deployment, ReplicaSet, endpoint controllers - none of it). A
Pod a Deployment "creates" never actually starts, because nothing ever
schedules or runs it. This is the honest boundary this repo's tests work
around explicitly rather than pretend doesn't exist -
`webapp_controller_test.go`'s first test needs "the Deployment is ready" to
assert on `Available` phase, so it does what a kubelet would:

```go
By("simulating pods becoming ready, the way a kubelet would drive Deployment status")
Expect(k8sClient.Get(ctx, key, &dep)).To(Succeed())
dep.Status.ObservedGeneration = dep.Generation
dep.Status.ReadyReplicas = 2
dep.Status.Replicas = 2
Expect(k8sClient.Status().Update(ctx, &dep)).To(Succeed())
```

That's real Kubernetes behavior - `status` is just another subresource
anyone with the RBAC to write it can write - not a mock. But it's you
driving it, not a kubelet, and the test says so in a comment rather than
implying otherwise. If you need real Pods actually running (this repo
never does), that's what [Stage 11](11-packaging.md)'s `kind`-based
checks and `make test-e2e` are for.

## Run the real controller, not just `Reconcile()` in isolation

Kubebuilder's default scaffold for a new controller test calls
`Reconcile()` directly, once, and asserts on the result. That only proves
the function's *logic* - it says nothing about whether
[Stage 3](03-watches-and-ownership.md)'s `Owns()` calls actually route the
right events to it. This repo's `suite_test.go` instead starts a **real
manager**, with the real `SetupWithManager`, running against envtest:

```go
mgr, err := ctrl.NewManager(cfg, ctrl.Options{...})
Expect((&WebAppReconciler{...}).SetupWithManager(mgr)).To(Succeed())
go func() {
    defer GinkgoRecover()
    Expect(mgr.Start(ctx)).To(Succeed())
}()
```

That's what makes "recreates a deleted owned Service without any test code
calling Reconcile" a meaningful test - nothing in the test body calls
`Reconcile`. The only thing that can make it pass is the watch wiring
working, exactly like it would in production.

## A real bug this exact setup caught

`internal/registry/registry.go`'s `getOrCreateRegistry` originally did
this after creating the shared registry ConfigMap:

```go
if err := c.Create(ctx, &cm); err != nil { ... }
// Re-fetch: another reconcile may have created it first, and we
// need its real resourceVersion before the caller can Update it.
if err := c.Get(ctx, ..., &cm); err != nil {
    return nil, err
}
```

That passed every envtest run. It failed on the very first reconcile
against a **real kind cluster** (v1.37.0, verified 2026-08-30), producing a
spurious `Warning HostNameConflict ConfigMap "webapp-registry" not found`
event on an object that had just been created successfully. The cause:
`mgr.GetClient()` is cache-backed, and a `Get` for an object you just
created can race the cache's watch before it's observed the create - a
genuine network-latency-dependent race that a single-process envtest run,
with everything on localhost, essentially never exposes. The fix (see the
current `getOrCreateRegistry`) is to use the object `Create` already
populated instead of re-fetching it - `client.Create` fills in the
server-assigned fields (`resourceVersion`, `UID`) in place, so there was
never a need to read it back at all.

The lesson isn't "envtest is broken" - every other bug this repo shipped
with was caught by it. It's narrower and more useful than that: **envtest
proves your reconcile logic and watch wiring are correct; it does not
prove your code is race-free under real network latency against a cache.**
For that, you need a real cluster at least once - which is exactly why
[Stage 10](10-versioning.md) and [Stage 11](11-packaging.md) each have a
live-cluster checkpoint, and why `make test-e2e` exists at all.

## When *not* to reach for envtest

`internal/controller/metrics_test.go` tests `recordPhaseMetric` with a
plain `go test`, no envtest, no Ginkgo suite. Metrics logic is pure
computation against a `prometheus.Registry` - it never touches the API
server. Booting a full `kube-apiserver` to test a map lookup would be
slower and no more correct. Match the tool to what you're actually
verifying.

## Checkpoint

- Run `go test ./internal/controller/... ./internal/registry/... ./internal/webhook/...`
  from `code/webapp-operator` (envtest binaries download automatically the
  first time). All green?
- `internal/registry/suite_test.go`'s `envtest.Environment{}` has no
  `CRDDirectoryPaths` set, unlike `internal/controller/suite_test.go`'s.
  Why doesn't the registry package's test suite need any CRDs installed at
  all?
- The cached-client race above only showed up on a real cluster, never in
  hundreds of envtest runs. What's different about envtest's client-to-API-server
  path that makes that race effectively unreachable there? (Hint: it's not
  that envtest doesn't use a cache.)

Next: [Stage 5 — Finalizers](05-finalizers.md).
