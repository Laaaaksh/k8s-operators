# Stage 0 — Toolchain setup

**You'll be able to:** build, test, and (optionally) run the operator this
curriculum builds, on your own machine.

**Time:** 30–60 minutes.

**Build:** confirm `make test` passes inside `code/webapp-operator`, with no
Kubernetes cluster running anywhere.

## What you need, and why

| Tool | Why | Verified version |
|---|---|---|
| [Go](https://go.dev/dl/) | Everything here is a Go module | 1.26.5 (repo requires 1.26+, per `go.mod`) |
| [Docker](https://docs.docker.com/get-docker/) | Builds the operator's container image; also how `envtest` downloads its own binaries under the hood in some setups | any recent version |
| `kubectl` | Talk to a cluster | any version within the [supported skew](https://kubernetes.io/releases/version-skew-policy/) of the cluster you use |
| [kind](https://kind.sigs.k8s.io/) | A real (if disposable) Kubernetes cluster on your laptop - only needed for Stages 7, 10, and 11's live checks, and `make test-e2e` | v0.33.0 |
| [kubebuilder](https://book.kubebuilder.io/quick-start.html) CLI | Scaffolds new APIs/webhooks - not required to build or test the operator that already exists here, only if you want to extend it the way this repo was built | v4.15.0 |

You do **not** need `kubebuilder`, `kind`, or even a running Docker daemon
just to read the code and run `make test` - that target uses
[envtest](https://book.kubebuilder.io/reference/envtest.html), which
downloads a real `kube-apiserver` and `etcd` binary pair (no kubelet, no
Docker) the first time you run it. See [Stage 4](04-envtest.md) for why that
matters.

## Install

```bash
# Go 1.26+
go version   # already installed on most dev machines; see go.dev/dl if not

# kubectl
brew install kubectl   # or your platform's package manager

# kind - only needed for Stages 7/10/11's live checks and `make test-e2e`
brew install kind

# kubebuilder CLI - only needed if you want to scaffold something new
go install sigs.k8s.io/kubebuilder/v4@latest
```

Then, from `code/webapp-operator`:

```bash
make test
```

The first run downloads `controller-gen`, `setup-envtest`, and a pinned
`kube-apiserver`/`etcd` pair into `bin/` (gitignored, machine-local) - this
takes a minute or two once. Every subsequent run is fast. You should see
something like:

```
ok  	.../internal/controller	7.5s	coverage: 81.0% of statements
ok  	.../internal/registry	5.0s	coverage: 76.4% of statements
ok  	.../internal/webhook/v1beta1	6.3s	coverage: 85.7% of statements
```

If that passes, you're set for every stage through [Stage 6](06-webhooks.md).
[Stage 7](07-leader-election.md) also only needs envtest. [Stage 10](10-versioning.md)
and [Stage 11](11-packaging.md) have one optional live-cluster check each
that needs `kind` and Docker - clearly marked, and skippable.

## Pinned versions this repo was built and verified against

| Component | Version | Verified |
|---|---|---|
| Kubernetes (cluster, via kind) | v1.37.0 | 2026-08-30, real kind cluster |
| kubebuilder CLI | v4.15.0 | 2026-08-30 |
| controller-runtime (and `envtest`) | v0.24.1 | 2026-08-30, `go.mod` |
| controller-gen | v0.21.0 | 2026-08-30 |
| client-go / apimachinery | v0.36.0 | 2026-08-30, `go.mod` - see the note below |
| cert-manager | v1.21.1 | 2026-08-30, installed and exercised on a real kind cluster |
| kind | v0.33.0 | 2026-08-30 |
| Kustomize | v5.8.1 | 2026-08-30 (bundled with `kubectl` 1.36+, and this repo's `bin/kustomize`) |
| Helm | v4.2.4 | 2026-08-30 |

**Why client-go is pinned at v0.36.0 against a v1.37.0 cluster:** kubebuilder
v4.15.0 scaffolds against the Kubernetes 1.36 client line by default, and
that's what this repo kept. Kubernetes' own [version skew
policy](https://kubernetes.io/releases/version-skew-policy/) supports
clients up to one minor version behind the server, so this is not a
mismatch - it's one release behind, deliberately, to match what the tool
actually generates. Bumping is a one-line `go get
k8s.io/...@v0.37.0` away if you want to track the cluster exactly; nothing
in this curriculum depends on the extra minor version.

## No GPU-shaped problem here, but a real equivalent

This repo doesn't have CUDA's "you might not own the hardware" problem -
`envtest` runs on any laptop, and `kind` needs only Docker. What it does have:
every command in this curriculum was run for real, against these exact
pinned versions, including the live-cluster checks in Stages 7, 10, and 11 -
none of it is asserted from memory. If a command's output looks different on
your machine, the version table above is where to start checking why.

## Checkpoint

- Run `make test` in `code/webapp-operator`. Does it pass without a cluster
  running? (It should - if it's trying to reach a real cluster, something's
  misconfigured; see [`resources/common-pitfalls.md`](../resources/common-pitfalls.md).)
- What's the difference between what `kubebuilder` the CLI does and what
  `controller-runtime` the library does? (You'll use both, but only one of
  them ships inside the binary you actually run in production.)

Next: [Stage 1 — API design](01-api-design.md).
