# Stage 9 — Observability

**You'll be able to:** add a domain-specific Prometheus metric, emit
Kubernetes Events correctly, and enable pprof - and know which of the three
answers which question, instead of reaching for all three by reflex.

**Time:** 2–3 hours.

**Build:** `internal/controller/metrics.go`, the `Recorder` field and
`Eventf` calls in `internal/controller/webapp_controller.go`, and the
`--pprof-bind-address` flag in `cmd/main.go`.

## Read, in this order

1. [Kubebuilder book: metrics](https://book.kubebuilder.io/reference/metrics.html) - the built-in metrics every controller-runtime manager exposes for free, and how to add your own.
2. [controller-runtime: `pkg/metrics`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/metrics) - the shared `Registry` this repo's custom metric registers onto.
3. [Kubernetes: Event v1 API reference](https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/) - the schema `kubectl describe`/`kubectl get events` read from. (There's no separate Events *concept* page any more - this API reference is the current source.)
4. [Prometheus Operator: design / `ServiceMonitor`](https://prometheus-operator.dev/docs/getting-started/design/) - if your cluster runs Prometheus Operator, this is how it discovers this operator's `/metrics` endpoint (`config/prometheus/` in this repo).
5. [`net/http/pprof`](https://pkg.go.dev/net/http/pprof) - the standard library package behind `--pprof-bind-address`.

## Three tools, three different questions

- **Metrics** answer "is the fleet healthy, in aggregate, over time" -
  cheap to query, cheap to alert on, expensive to make specific to one
  object.
- **Events** answer "what happened to *this one* object, and when" -
  exactly what `kubectl describe webapp demo` shows, gone after their
  (short, cluster-configured) retention window.
- **pprof** answers "why is this process itself slow or leaking memory
  right now" - a live snapshot of one running binary, nothing to do with
  any Kubernetes object at all.

Reaching for the wrong one is a common, avoidable failure mode: a Warning
Event per failed reconcile floods `kubectl get events` cluster-wide under
load and tells you nothing in aggregate; a metric with a
`namespace`/`name` label per object works for a handful of WebApps and
becomes a cardinality problem at scale if misused (see below for how this
repo avoids that specific trap).

## The metric: what's free, and what earns its keep

controller-runtime already registers `controller_runtime_reconcile_total`
(with a `result` label: `success`/`error`/`requeue`) and
`controller_runtime_reconcile_time_seconds` for every controller, for
free. `internal/controller/metrics.go` doesn't duplicate that - it adds
exactly one metric that answers something controller-runtime has no way
to know: what phase is each WebApp in, right now.

```go
var webappPhase = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{Name: "webapp_operator_webapp_phase", ...},
    []string{"namespace", "name", "phase"},
)
```

This is a kube-state-metrics-style `_info` gauge: always `1` for the
current phase, absent (not `0`) for every other phase - `sum by (phase)`
gives you a count per phase. The part easy to get wrong: **a stale label
combination left behind after a transition is a real bug, not a cosmetic
one** - a WebApp that moved from `Progressing` to `Available` and left
`{phase="Progressing"} 1` behind would make any dashboard summing "how
many WebApps are Progressing" silently over-count forever. `recordPhaseMetric`
handles this by deleting every *other* phase's series on every reconcile,
and `deletePhaseMetric` clears all of them when a WebApp is actually
deleted - covered by a plain `go test` (`metrics_test.go`, no envtest
needed, see [Stage 4](04-envtest.md)'s note on matching the test tool to
what you're testing) that asserts the series count is exactly right after
a transition and exactly zero after deletion.

## Events: the API migrated under this repo, mid-build

controller-runtime v0.24 deprecated `mgr.GetEventRecorderFor` (the
`record.EventRecorder` type, backed by the older core `v1` Event API) in
favor of `mgr.GetEventRecorder` (`events.EventRecorder`, backed by the
newer `events.k8s.io/v1` API) - a real, current API surface change this
repo hit live via `golangci-lint`'s `staticcheck` check flagging the old
call as `SA1019: deprecated`. Most existing blog posts and even some
still-circulating operator boilerplate use the old signature; this repo
uses the new one:

```go
Recorder events.EventRecorder   // not record.EventRecorder
```

The new interface's `Eventf` also has a richer signature than the old
one - `regarding`, `related`, `eventtype`, `reason`, **`action`** (new -
separate from `reason`), `note`, `args`:

```go
r.Recorder.Eventf(webapp, dep, corev1.EventTypeNormal, "DeploymentReconciled", "Reconcile", "Deployment %s %s", dep.Name, op)
```

`related` (here, the Deployment) is what lets `kubectl describe deploy` and
`kubectl describe webapp` both plausibly surface events connected to the
same underlying action - something the old API had no clean way to
express.

## A real gotcha this repo shipped, on a real cluster

`WebAppReconciler.Recorder` is a plain field, not something Go's compiler
requires you to set. Every place a `WebAppReconciler` is constructed -
`cmd/main.go`, `suite_test.go` - has to set it explicitly:

```go
Recorder: mgr.GetEventRecorder("webapp-controller"),
```

Forget it, and everything compiles, `go vet` is silent, and every test
that doesn't happen to hit an `Eventf` call still passes - it panics with
a nil-pointer dereference only the first time a real code path (a
hostname conflict, say) actually tries to emit an event, possibly weeks
after the change that broke it shipped. There's no compiler or linter
check that would have caught this before it broke; the only real defenses
are consistent construction (grep for every `WebAppReconciler{` site
before merging a `Recorder`-touching change) and integration tests that
actually exercise the event-emitting paths, not just the happy path.

## pprof: off by default, and why that default matters

`--pprof-bind-address` is empty (disabled) unless explicitly set. pprof's
endpoints are unauthenticated and unencrypted by design - anyone who can
reach the port gets a live profile of the process, including in-flight
data shapes. Fine to enable on a port reachable only from inside the
pod's own network namespace (a `kubectl port-forward` for a one-off
debugging session); never expose it the way the metrics endpoint is
exposed (behind the `metrics-auth-role` RBAC-gated proxy this repo's
`cmd/main.go` already wires up for `/metrics` - pprof has no equivalent
gate here).

## Checkpoint

- `recordPhaseMetric` is called unconditionally on every reconcile, even
  when the phase didn't change. Why is that safe and not wasted work -
  what does calling `.Delete()` on a label combination that was never set
  actually cost?
- If you added a `webapp_operator_reconcile_errors_total` counter,
  incremented once per failed reconcile, what would it tell you that
  controller-runtime's own `controller_runtime_reconcile_total{result="error"}`
  doesn't already? Is there a good reason to add it anyway, or would it
  be pure duplication?
- Enable `--pprof-bind-address=:6060` locally (`make run
  ARGS="--pprof-bind-address=:6060"`, or edit `config/manager/manager.yaml`
  for a real deploy) and hit `http://localhost:6060/debug/pprof/` with
  `kubectl port-forward`. What's actually listening on that port, and
  what specifically would go wrong if that port were exposed via a
  `NodePort` Service instead?

Next: [Stage 10 — Versioning](10-versioning.md).
