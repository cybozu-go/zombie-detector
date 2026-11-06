# Maintenance procedure

This page describes how to update zombie-detector regularly.

## Prerequisites

CLI tools (gh, yq, kind, kustomize, kubectl, setup-envtest) are managed by [aqua](https://aquaproj.github.io/) in `aqua.yaml`.
Their checksums are recorded in `aqua-checksums.json` and verified on installation.
The Makefiles execute the tools via `aqua exec`, so install aqua in advance and put the `aqua` command in a directory included in `PATH`.
See [Install aqua](https://aquaproj.github.io/docs/install) for details.
Then run `make setup` to install the tools.

## Procedure

1. Update `go.mod`. In addition to go modules, update the Go version if needed.
2. Update Ubuntu versions in `.github/workflows/*.yaml` if needed.
3. Update the base image of the build stage in `Dockerfile`.
4. Update software versions. When using `make maintenance`, you are prompted to login to github.com.
   ```console
   $ make maintenance
   ```
   This updates the following:
   - gh, yq, kind, kustomize and the aqua standard registry in `aqua.yaml`, and `aqua-checksums.json`
   - aqua version in `.github/workflows/*.yaml`
   - pushgateway image in `e2e/manifests/pushgateway/kustomization.yaml`
   - metrics-server manifest in `e2e/manifests/metrics-server/components.yaml`
     and the image tag and digest in `e2e/manifests/metrics-server/kustomization.yaml`

   Only versions released at least 14 days ago are adopted.
   The cooldown period can be changed with `COOLDOWN_DAYS` (e.g. `make maintenance COOLDOWN_DAYS=7`).
5. Update `ENVTEST_K8S_VERSION` in `Makefile.versions`.
   Also update `kubernetes-sigs/controller-runtime/setup-envtest` in `aqua.yaml` to the same version as controller-runtime in `go.mod`.
   ```console
   $ # Use this command to list the available k8s versions for envtest
   $ aqua exec -- setup-envtest list
   ```
6. Update `E2ETEST_K8S_VERSION` and `E2ETEST_KINDEST_NODE_IMAGE_HASH` in `Makefile.versions`.
   Specify the version of `kindest/node` supported by the kind written in `aqua.yaml`.
   Also update `kubernetes/kubectl` in `aqua.yaml` to the same version as `E2ETEST_K8S_VERSION`.
7. If you changed `aqua.yaml` manually, update `aqua-checksums.json`.
   Since `require_checksum` is enabled, packages without checksums fail to install.
   ```console
   $ aqua update-checksum -prune
   ```
   See also [Notes on editing aqua.yaml manually](#notes-on-editing-aquayaml-manually).
8. Follow [release.md](/docs/release.md) to update software version.

## Notes on editing aqua.yaml manually

- Update only `kubernetes/kubectl` and `kubernetes-sigs/controller-runtime/setup-envtest` manually.
  The other packages and the standard registry are updated by `make maintenance` with the cooldown period.
- Write the version as the tag name, including its prefix if any (e.g. `kubernetes-sigs/kustomize@kustomize/v5.8.3`).
- You can select a version interactively with `aqua update -s <package>` (e.g. `aqua update -s kubernetes/kubectl`).
  Do not run `aqua update` without arguments, because it updates all packages to the latest versions ignoring the cooldown period.
- After editing, update `aqua-checksums.json` with `aqua update-checksum -prune`.
