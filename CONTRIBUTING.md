# Contributing to k8s-operators

Thanks for considering a contribution. This is a curriculum plus one real
operator, open source under the Apache 2.0 license.

## Getting started

```bash
git clone https://github.com/<your-username>/k8s-operators.git   # your fork
cd k8s-operators/code/webapp-operator
```

You need Go 1.26+ and Docker. Everything else - `kubebuilder`, `envtest`
binaries, `controller-gen`, `kustomize`, `golangci-lint` - is fetched
on-demand into `code/webapp-operator/bin/` the first time you run `make`.

```bash
make test        # unit + envtest integration tests, no cluster needed
make lint         # golangci-lint
```

If you have Docker and want to see the operator run for real:

```bash
make test-e2e     # creates its own kind cluster, deploys, tests, tears down
```

`curriculum/00-toolchain-setup.md` covers all of this in more depth,
including what to do if you don't have Docker.

## Contribution workflow

1. Fork the repo, clone your fork (command above).
2. Create a descriptively named branch off `main`.
3. Make focused commits.
4. If you touched anything under `code/webapp-operator/`, run `make test`
   and paste the output in the PR. If you touched `api/`, `config/`, or RBAC
   markers, also run `make manifests generate` and commit the result - CI
   fails if generated files don't match source.
5. If you touched `resources/curated-resources.md`, open every link you're
   adding or changing and confirm it resolves before submitting. Never add a
   link you haven't personally opened.
6. Open a pull request against `main`.

A PR can merge only when CI's `test`, `lint`, `helm`, and `e2e` jobs pass and
review feedback is resolved.

## What contributions are useful

- Fixing a wrong claim, a stale "current" statement, or a dead link.
- Corrections from testing against a Kubernetes version or `controller-runtime`
  release this repo hasn't been checked against - say which versions you used.
- Curriculum sequencing feedback: if a stage assumes something the previous
  stage didn't actually teach, that's a real bug in a course, not a nitpick.
- A new checkpoint test that exercises a real failure mode (a race, a
  misconfigured RBAC rule, a webhook edge case) the existing tests miss -
  open an issue first so scope is agreed before you write it.

## Adding to the operator

`code/webapp-operator` is one project, built up incrementally across the
curriculum stages - there's no per-stage snapshot directory the way some
sample repos work. Each curriculum stage's markdown file names the specific
files and functions it added; use those as a map of what "belongs" to which
concept before changing something load-bearing elsewhere.

- Business logic goes in `internal/`, one package per concern
  (`controller`, `registry`, `webhook/v1beta1`) - match that split rather
  than growing one package.
- Comments explain *why* a line matters for the concept being taught (a
  cache-consistency gotcha, an RBAC decision, a lossy conversion), not what
  the Kubernetes API call does - link to docs for that.
- RBAC markers (`+kubebuilder:rbac:...`) should grant exactly the verbs a
  reconciler or webhook actually calls - see
  `curriculum/08-least-privilege-rbac.md` for the audit method this repo
  uses, and don't add a verb "just in case."
- Every new controller/webhook behavior needs an envtest (or, for something
  that genuinely needs a real cluster - leader election, the conversion
  webhook's live HTTPS path - a test that says so in a comment) proving it,
  not just a claim in prose. See `curriculum/04-envtest.md`.

## Code style

- `gofmt`-clean, `go vet`-clean, `make lint`-clean (golangci-lint config is
  in `code/webapp-operator/.golangci.yml`).
- Comments explain *why*, not *what*.

## Reporting issues

Open a GitHub issue before starting anything larger than a typo fix, so scope
is agreed first. Use the bug report template for something broken, and the
resource suggestion template for anything about
`resources/curated-resources.md`.
