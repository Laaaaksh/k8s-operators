# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

- **What this repo is**: a sequenced Kubernetes operators curriculum (`curriculum/`) built around one real, incrementally-grown operator (`code/webapp-operator/`), plus honestly-dated resource curation (`resources/`). See the root `README.md` for the pitch and `curriculum/README.md` for the stage table.
- **The operator is one project, not per-stage snapshots**: unlike a series with many small independent samples, `code/webapp-operator` is a single Go module built up across all 12 curriculum stages. There are no per-stage directories or git tags - each `curriculum/NN-*.md` file names the specific files/functions it added. Read a stage's file list before changing something that might be load-bearing for an earlier stage's narrative.
- **Toolchain**: Go 1.26+, `kubebuilder` v4.15.0 CLI (scaffolding only, not required to build/test), `controller-runtime` v0.24.1, Kubernetes client libraries pinned at v0.36.0 against a cluster that's currently v1.37.0-current (see `resources/curated-resources.md` for the version table and why the client-lib lag is fine under Kubernetes' skew policy). `make manifests generate` (controller-gen) regenerates `api/*/zz_generated.deepcopy.go` and `config/crd,rbac,webhook`; CI fails if that output doesn't match what's committed.
- **Testing model**: `make test` runs everything except `test/e2e` against `envtest` (a real kube-apiserver + etcd, no kubelet - see `curriculum/04-envtest.md`) - no real cluster needed, no mocks. `make test-e2e` additionally spins up a real `kind` cluster, installs cert-manager, builds and loads the image, deploys via Kustomize, and tears down after - this is the only test path that exercises the real HTTPS conversion webhook end-to-end and real leader-election Lease contention. Both are wired into CI (`.github/workflows/ci.yml`); prefer `make test` while iterating and reserve `make test-e2e` for changes to webhooks, RBAC, leader election, or packaging.
- **RBAC discipline**: `+kubebuilder:rbac` markers in `internal/controller/webapp_controller.go` are audited against actual client calls, not copy-pasted from scaffolding - notably no `delete` verb on resources the controller only creates/updates. See `curriculum/08-least-privilege-rbac.md` for the audit method; don't add a verb "just in case" without updating that doc's reasoning.
- **Citation discipline**: every external link in `curriculum/*.md` and `resources/*.md` must be a URL someone actually opened and confirmed live - `resources/curated-resources.md` states dates and current/aged/dead verdicts per entry. Before adding or changing a citation, fetch the URL yourself; don't restate one from memory.
- **Packaging**: `dist/install.yaml` (plain Kustomize output) and `dist/chart/` (Helm, generated via `kubebuilder edit --plugins=helm/v2-alpha`) are both committed, generated artifacts - regenerate with `make build-installer` / re-run the kubebuilder command after changing `config/`, don't hand-edit them.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
