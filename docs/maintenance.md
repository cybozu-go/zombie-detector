# Maintenance procedure

This page describes how to update zombie-detector regularly.

## Regular Update

1. Update `go.mod`.
2. Update Go & Ubuntu versions if needed.
3. Update software versions. When using `make maintenance`, you are prompted to login to github.com.
   ```console
   $ make maintenance
   ```
4. Follow [release.md](/docs/release.md) to update software version.

## Kubernetes Update

1. Update `go.mod`.
2. Update `ENVTEST_K8S_VERSION` in `Makefile`.  
   ```console
   $ # Use this command to list the available k8s versions for envtest
   $ ./bin/setup-envtest list
   ```
3. Follow [release.md](/docs/release.md) to update software version.
