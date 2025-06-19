# Maintenance procedure

This page describes how to update zombie-detector regularly.

1. Update `go.mod`. In addition to go modules, update the Go version if needed.
2. Update Ubuntu versions in `.github/workflows/*.yaml` if needed.
3. Update the base image of the build stage in `Dockerfile`.
4. Update software versions. When using `make maintenance`, you are prompted to login to github.com.
   ```console
   $ make maintenance
   ```
5. Update `ENVTEST_K8S_VERSION` in `Makefile.versions`.
   ```console
   $ # Use this command to list the available k8s versions for envtest
   $ ./bin/setup-envtest list
   ```
6. Update `E2ETEST_K8S_VERSION` in `e2e/Makefile.versions`.
   Specify the version of `kindest/node` supported by the kind written in `e2e/Makefile`.
7. Follow [release.md](/docs/release.md) to update software version.
