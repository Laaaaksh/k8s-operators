# Stage 11 — Packaging

**You'll be able to:** ship this operator as Kustomize manifests and a Helm
chart, verify both are accepted by a real API server, and make an informed
call on where OLM fits today (and where it doesn't, yet).

**Time:** 2–3 hours.

**Build:** `dist/install.yaml`, `dist/chart/`, and
`test/manifests/manifests_test.go`.

## Read, in this order

1. [Helm: chart best practices](https://helm.sh/docs/chart_best_practices/) - conventions for CRDs, RBAC, and values in a chart, doc-versioned to Helm v4.2.4.
2. This stage's own "OLM in 2026" section below - there's no single current, accurate page to point you at instead; the state is genuinely mid-transition, and the primary sources (linked below) each tell only part of the story.

## Kustomize first: it's already the source of truth

Every other stage's manifests already live under `config/` as plain
Kustomize - `config/crd`, `config/rbac`, `config/manager`, `config/webhook`,
`config/certmanager`, assembled by `config/default/kustomization.yaml`.
`dist/install.yaml` is that assembly, flattened into one file:

```bash
make build-installer   # kustomize build config/default > dist/install.yaml
```

This is the lowest-ceremony path to installing the operator anywhere:
`kubectl apply -f dist/install.yaml`. It's also the thing every other
packaging format in this stage is generated *from*, not a parallel
artifact maintained by hand.

## Helm: generated from Kustomize, not hand-authored

`dist/chart/` was generated with kubebuilder's `helm/v2-alpha` plugin
(`kubebuilder edit --plugins=helm/v2-alpha`) - it renders Kustomize's
`config/default` output into Helm templates, rather than the older
`helm/v1-alpha` plugin's more manual copy-based approach (deprecated as of
kubebuilder v4.15.0, in favor of `v2-alpha`'s dynamic generation - this
repo was built after that deprecation and uses the current plugin).
Re-run that command after changing `config/`; don't hand-edit
`dist/chart/templates/`.

```bash
helm lint dist/chart              # 0 issues
helm template webapp-operator dist/chart > /dev/null   # renders cleanly
```

## Verified against a real API server, not just "it parses"

`helm lint` and `helm template` prove the chart *renders*. Neither proves
the API server will *accept* what comes out - a bad RBAC verb or a
malformed field would pass both silently. `test/manifests/manifests_test.go`
closes that gap: it applies the real, generated `dist/install.yaml` -
every object in it, not a hand-picked excerpt - against a real envtest
`kube-apiserver`, and asserts each one is accepted:

```go
t.Logf("applied %d/%d objects from dist/install.yaml (cert-manager.io objects skipped)", applied, len(objs))
// applied 18/21 objects from dist/install.yaml (cert-manager.io objects skipped)
```

The three skipped objects are `cert-manager.io` resources (`Issuer`, two
`Certificate`s, from [Stage 6](06-webhooks.md)) - envtest only has the
built-in Kubernetes API types installed, not cert-manager's CRDs, and
installing them just to validate three objects outside this repo's own
schema would be testing cert-manager, not this operator. Everything this
repo actually owns the schema of gets checked for real.

This was also confirmed on a genuine kind cluster with cert-manager
actually installed (v1.21.1, verified 2026-08-30) - `kubectl apply -f
dist/install.yaml` against it produced a running, leader-elected,
webhook-serving operator with zero manual fixes, via `make test-e2e`'s
automated CI path (see `.github/workflows/ci.yml`'s `e2e` job).

## Where OLM fits, as of 2026-08-30

Worth stating plainly because it materially affects what you'd reach for:
**classic Operator Lifecycle Manager (v0) is in explicit maintenance mode.**
Its own README states this directly - no new features, only fixes for
issues that cause a cluster outage with no workaround, everything else
closed. It's still tagged periodically (v0.46.0, 2026-07-23), but it is
not the forward path.

The stated successor, **OLM v1** (`operator-controller`), is actively
developed (v1.11.0, 2026-07-23) - but its own docs state there is
"currently no concrete migration strategy" from v0, and v1 "supports [only]
a subset of the existing content supported by OLMv0."

**Net for this stage: Kustomize and Helm - both built here - are the
primary, unambiguous distribution paths today.** OLM v1 is worth tracking
if you need OperatorHub-style discovery/lifecycle management and can
tolerate its current incompleteness; classic OLM (v0) is legacy - don't
build new distribution around it. (`sdk.operatorframework.io`'s
best-practices page still frames OLM as "the standard" for operator
distribution, but that specific page is dated 2023-06-21 - stale relative
to 2026's OLM v0/v1 transition; use it for its semver/CRD-versioning
advice, not that framing.)

**Operator SDK vs. Kubebuilder, while we're here:** independently
maintained, not merged. Operator SDK uses Kubebuilder under the hood for
Go scaffolding and layers on OLM integration, OperatorHub publishing, and
multi-language support; Kubebuilder's own design docs describe an
in-progress intent to unify further, not a completed merger. If you only
need what this repo builds - a Go operator, Kustomize/Helm packaging - the
extra layer Operator SDK adds isn't buying you anything yet.

## Checkpoint

- Run `test/manifests/manifests_test.go` yourself:
  `go test ./test/manifests/... -v` (from `code/webapp-operator`). What's
  the actual count it reports, and does it match this stage's claim of
  18/21?
- `dist/install.yaml`'s image reference is the placeholder
  `controller:latest`. What has to happen - concretely, which commands -
  between cloning this repo and getting a real, running operator on a
  cluster you control? (This is exactly what `make test-e2e` automates -
  read `test/e2e/e2e_suite_test.go`'s `BeforeSuite` if you want the exact
  sequence.)
- If you needed OperatorHub discoverability today, would you reach for
  classic OLM (v0) or OLM v1, given the "no concrete migration strategy"
  gap between them? What would you tell a team that already has OLM v0
  bundles for other operators in the same cluster?

You've now built one operator across every stage this repo covers. See the
root [README](../README.md)'s "what this doesn't cover" for a candid list
of what's deliberately out of scope, and where to go next.
