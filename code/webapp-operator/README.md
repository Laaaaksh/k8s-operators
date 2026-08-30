# webapp-operator

The one real, cumulative operator built across every stage of
[../../curriculum](../../curriculum). Given a `WebApp` custom resource
(image, port, and either a fixed replica count or a min/max scaling range),
it manages a Deployment and Service, tracks a shared hostname registry with
finalizer-backed cleanup, validates and defaults input via admission
webhooks, supports leader-election HA, runs under an audited
least-privilege RBAC role, exposes a custom Prometheus metric, and ships as
both Kustomize manifests and a Helm chart.

See [`../README.md`](../README.md) for what this file structure means -
this is one project built up incrementally, not per-stage snapshots.

## Getting Started

### Prerequisites

Verified against these exact versions - see
[`../../curriculum/00-toolchain-setup.md`](../../curriculum/00-toolchain-setup.md)
for the full pinned table and why each one was chosen:

- Go 1.26+
- Docker (any recent version) - only needed for `make docker-build` and `make test-e2e`
- `kubectl` - only needed to talk to a real cluster
- Kubernetes v1.37.0, if you deploy to a real cluster (client libraries here are pinned one minor version behind, per Kubernetes' skew policy - see the toolchain doc)

`make test` needs none of the above except Go - it downloads its own
`envtest` binaries on first run.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/webapp-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don't work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/webapp-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create a WebApp:**

```sh
kubectl apply -f config/samples/apps_v1alpha1_webapp.yaml
```

Or the fastest zero-registry-required path: `make test-e2e` builds and
loads the image into a disposable `kind` cluster, deploys, and tears down
automatically - see
[`../../curriculum/11-packaging.md`](../../curriculum/11-packaging.md).

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs (CRDs) from the cluster:**

```sh
make uninstall
```

**Undeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Two packaging formats are generated from the same `config/` Kustomize
source and committed to this repo - see
[`../../curriculum/11-packaging.md`](../../curriculum/11-packaging.md) for
how each was verified and where OLM fits (and doesn't, yet) as a third
option.

### `dist/install.yaml` - a flattened Kustomize bundle

```sh
kubectl apply -f dist/install.yaml
```

Regenerate after changing `config/`: `make build-installer IMG=<registry>/webapp-operator:tag`.

### `dist/chart/` - a Helm chart

```sh
helm install webapp-operator dist/chart
```

Regenerate after changing `config/`: `kubebuilder edit --plugins=helm/v2-alpha`
(this repo was built with kubebuilder v4.15.0's `helm/v2-alpha` plugin -
`helm/v1-alpha` is deprecated as of that release; don't use it for new
changes).

## Contributing

See [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md) and
[`../../AGENTS.md`](../../AGENTS.md) for this project's specific
conventions (RBAC audit discipline, test-tool selection, generated-file
regeneration).

**NOTE:** Run `make help` for the full list of `make` targets.

More information on the underlying scaffolding tool can be found via the
[Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html).

## License

Apache 2.0 - see [`../../LICENSE`](../../LICENSE).
