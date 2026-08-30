# Common pitfalls

Specific things that stop people building their first operator - not a
general "troubleshooting Kubernetes" list. Every item here either happened
while building this repo, or is a documented, verifiable gap in the
official tooling. Checked here before assuming you've found a new problem.

## Toolchain and environment

### `make test` tries to reach a real cluster and fails

`make test` should need zero cluster access - it drives
[envtest](../curriculum/04-envtest.md), which downloads its own
`kube-apiserver`/`etcd` pair. If you see connection errors to `localhost:8080`
or a real cluster's API server, `KUBEBUILDER_ASSETS` probably isn't set to
where those binaries actually live, or you ran `go test` directly instead
of `make test` (which sets it for you via `setup-envtest`). Fix:

```bash
export KUBEBUILDER_ASSETS="$(setup-envtest use -p path 1.36.x! --bin-dir bin/k8s)"
```

or just use `make test`, which does this already.

### `envtest list` shows fewer Kubernetes versions than you expect

`setup-envtest list` (no flags) only shows what's cached locally plus a
snapshot of the remote index - it is not the authoritative list of every
version available. `setup-envtest use` (no version pinned) always fetches
the actual latest; this repo pins `1.36.x!` to match kubebuilder v4.15.0's
default client generation (see [Stage 0](../curriculum/00-toolchain-setup.md)
on why that's a deliberate, safe one-minor-version lag, not a mismatch).

### `kubectl get webapp.v1beta1` returns "the server doesn't have a resource type"

kubectl's `TYPE.VERSION.GROUP` syntax needs the **full group name**, not
just the version:

```bash
kubectl get webapp.v1beta1.apps.example.com demo   # works
kubectl get webapp.v1beta1 demo                     # doesn't - looks for a resource literally named "webapp.v1beta1"
```

Easy to trip on this exact repo's [Stage 10](../curriculum/10-versioning.md)
checkpoint, where fetching the same object as two different versions is
the whole point.

## Reconciler mental model

### "It compiled and the tests pass" says nothing about RBAC

Nothing in `go build`, `go vet`, or even a passing `make test` run checks
whether the RBAC markers on your reconciler grant what the code actually
needs, or grant *more* than that. envtest runs with a fully-privileged
admin config by default - your tests will pass even if the real deployed
ServiceAccount would be denied every call. The only real check is `kubectl
auth can-i --as=<serviceaccount>` against a real cluster, or reading
`config/rbac/role.yaml` against the reconciler's actual client calls by
hand. See [Stage 8](../curriculum/08-least-privilege-rbac.md) for the audit
method this repo uses.

### Assuming a controller needs `delete` on everything it creates

The single most commonly over-granted verb. If a child object is
garbage-collected via `ownerReferences` ([Stage 3](../curriculum/03-watches-and-ownership.md)),
your controller never calls `Delete` on it - Kubernetes' own garbage
collector does, as a different actor with its own permissions. Kubebuilder's
default scaffold grants `delete` on every resource a controller touches
regardless; this repo's own RBAC markers deliberately don't, and CI has no
way to catch this for you - it's a manual audit every time you add a new
owned resource type.

### A reconciler that calls `RequeueAfter` "just in case"

If your controller `Owns()` or `Watches()` the resources whose state
changes actually matter to it, a watch event re-triggers `Reconcile`
already - polling with `RequeueAfter` on top duplicates that, on a delay,
for no benefit. `RequeueAfter` has a real use (waiting out an external
rate limit, retrying a known-transient external dependency), but "I'm not
sure what triggers this again" is a sign something's missing from your
`Owns()`/`Watches()` list, not a reason to poll.

## The one this repo shipped with, and how it was found

### A cached client can race its own recent write

`internal/registry/registry.go` originally re-fetched a ConfigMap
immediately after creating it, to get the server-assigned
`resourceVersion` before returning it to the caller. That passed every
[envtest](../curriculum/04-envtest.md) run - hundreds of them - and failed
on the very first reconcile against a real kind cluster, producing a
spurious `Warning HostNameConflict ConfigMap "webapp-registry" not found`
on an object that had, in fact, just been created successfully.

The cause: `mgr.GetClient()` is cache-backed. A `Get` immediately after a
`Create` can race the cache's own watch stream before it's observed the
create - a real network-timing race that a single-process, localhost-only
envtest run essentially never exposes, because there's no meaningful
latency for the race to land in. The fix was to stop re-fetching at all -
`client.Create` already populates the object you passed it with the
server's response, including `resourceVersion`, so there was never a
reason to read it back.

**The general lesson**: envtest proves your reconcile logic and watch
wiring are correct. It does not prove your code is race-free under real
network latency against a cache - for that, you need at least one pass
against a real cluster before you trust a caching pattern. See
[Stage 4](../curriculum/04-envtest.md) for the full writeup, including the
actual code diff.

### Resource-version conflicts under rapid status updates - not a bug

Watching this repo's own operator reconcile a freshly created WebApp
against a real cluster produces log lines like:

```
ERROR  Reconciler error ... error: "Operation cannot be fulfilled on deployments.apps \"demo\": the object has been modified; please apply your changes to the latest version and try again"
```

This looks alarming the first time you see it and is, in fact, expected,
healthy behavior: multiple watch events (the Deployment's `readyReplicas`
ticking up as pods start) can trigger overlapping reconciles fast enough
to race each other's `Update` calls. controller-runtime's default
behavior - return the error, let the manager's rate limiter requeue -
self-corrects within milliseconds. Don't reflexively wrap every `Update`
call in `client-go`'s `retry.RetryOnConflict` to make the error message go
away; that trades a cheap, automatic requeue for an in-process retry loop
that accomplishes the same thing with extra code. (This repo *does* use
`RetryOnConflict` in `internal/registry` - but there, the write is to a
ConfigMap shared across every WebApp in the cluster, where contention is
the normal case, not an occasional race during startup. Match the pattern
to how *contended* the resource actually is.)

## Where to look next

If something's wrong and it isn't here, check whether it's covered by a
specific curriculum stage's own "Checkpoint" section first - several of
those checkpoints exist specifically because the answer surprised someone
while this repo was being built.
