# Stage 7 — Leader election

**You'll be able to:** explain exactly what `--leader-elect` does and
doesn't protect against, and prove failover works with a real test instead
of trusting the flag.

**Time:** 1–2 hours.

**Build:** `internal/leaderelection/leaderelection_test.go` - there's no new
production code this stage, deliberately (see below).

## Read, in this order

1. [controller-runtime: `manager.Options`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Options) - `LeaderElection`, `LeaderElectionID`, `LeaderElectionNamespace`, `LeaseDuration`/`RenewDeadline`/`RetryPeriod`.
2. [Kubernetes: Lease objects](https://kubernetes.io/docs/concepts/architecture/leases/) - the primitive leader election is built on: one object, one holder, a renewal deadline.

## The flag was already there

Kubebuilder v4.15.0 scaffolds `--leader-elect` (default `false`) into
`cmd/main.go` for every new project - `LeaderElection: enableLeaderElection`
and `LeaderElectionID: "53ec4676.example.com"` were already in this repo's
`ctrl.Options` before this stage existed. That's not unusual, and it's
worth naming directly: most of what's "hard" about leader election in a
real operator isn't writing it, it's **understanding what it actually
buys you and verifying it**, which is exactly what most tutorials skip
(and exactly what the [gap this repo exists to close](../README.md#what-this-doesnt-cover)
calls out - the Kubebuilder book itself has no leader-election chapter at
all, only the bare fact that the flag exists).

## What it actually protects against

Running two-plus replicas of a controller gives you availability during a
rolling upgrade or a node failure - but if both replicas reconcile the same
objects at once, you get double the API load and, worse, races: two
replicas racing to `CreateOrUpdate` the same Deployment can genuinely
conflict (see [Stage 4](04-envtest.md)'s note on the resourceVersion-conflict
errors this repo's own controller produces under rapid concurrent writes -
those are healthy and self-correct via requeue; leader election is what
keeps you from having *two independent controllers* generating that same
kind of contention against each other, all the time, as a matter of
course rather than a brief race).

Leader election solves this with one Lease object in the cluster
(`53ec4676.example.com`, this operator's `LeaderElectionID`): every replica
tries to acquire it; exactly one succeeds and starts reconciling; the
others block, watching the Lease, ready to take over the instant the
current holder stops renewing it (crashes, is evicted, or is shut down for
a rolling update).

**What it does not protect against**: a single replica that hangs but
doesn't crash - it can keep renewing its Lease while wedged, blocking
failover, until whatever's watching pod health kills it. Leader election
answers "how many replicas act," not "is this one replica healthy."

## Verified with a real test, not a comment claiming it works

`internal/leaderelection/leaderelection_test.go` starts two real managers
against the same envtest API server, both configured exactly like this
operator's own `cmd/main.go` (same `LeaderElectionID`, shortened
Lease/renew/retry durations so the test runs in seconds instead of the
production defaults' ~15s), and asserts:

```go
Eventually(func() bool {
    // exactly one of mgrA.Elected() / mgrB.Elected() has fired
}, 10*time.Second, 100*time.Millisecond).Should(BeTrue())

By("stopping the leader lets the other one take over")
cancelA()
// ... mgrB.Elected() fires within 15 seconds
```

Run it and read the log output - it's the real
`k8s.io/client-go/tools/leaderelection` package's own logging, not this
repo's:

```
I ... "Attempting to acquire leader lease..." lock="default/53ec4676.example.com"
I ... "Successfully acquired lease" lock="default/53ec4676.example.com"
E ... "Error initially creating lease lock" err="leases.coordination.k8s.io \"53ec4676.example.com\" already exists"
```

That `Error initially creating lease lock ... already exists` line is the
second manager losing the race to create the Lease and falling back to
watching the one the winner just created - not a bug, the mechanism
working exactly as designed.

## Checkpoint

- Why does the test use `LeaseDuration: 2 * time.Second` instead of the
  production default (~15s)? What would change about the test's *behavior*
  (not just its runtime) if it used the real 15-second default?
- `cmd/main.go`'s comment mentions `LeaderElectionReleaseOnCancel` -
  currently commented out. What does setting it to `true` trade away, and
  why does the comment say it's only safe if "the binary immediately ends
  when the Manager is stopped"? Does this operator's `main` function
  satisfy that condition?
- If you scaled this operator's Deployment to 3 replicas right now (see
  `config/manager/manager.yaml`), how many of them would actually run
  `Reconcile` at once? How many would be doing *nothing but* watching a
  Lease?

Next: [Stage 8 — Least-privilege RBAC](08-least-privilege-rbac.md).
