[![GitHub release](https://img.shields.io/github/release/cybozu-go/zombie-detector.svg?maxAge=60)][releases]
[![CI](https://github.com/cybozu-go/zombie-detector/actions/workflows/ci.yaml/badge.svg)](https://github.com/cybozu-go/zombie-detector/actions/workflows/ci.yaml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/cybozu-go/zombie-detector?tab=overview)](https://pkg.go.dev/github.com/cybozu-go/zombie-detector?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/zombie-detector)](https://goreportcard.com/report/github.com/cybozu-go/zombie-detector)

Zombie detector
============================
Zombie detector is CronJob to detect resources that elapsed for a long time since deletion.

## Features
- It detects resources that remain undeleted after a certain period with a ```deletionTimestamp```.
- Information about detected resources are pushed into [Pushgateway](https://github.com/prometheus/pushgateway).
- We can use this both inside and outside cluster.

## Build
CLI
```
go build
```
Docker Image
```
make docker-build
```

## Usage
```
Usage:
  zombie-detector [flags]
  zombie-detector [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Show the version of zombie-detector

Flags:
  -h, --help                 help for zombie-detector
      --incluster            execute in cluster or not (default true)
      --pushgateway string   URL of Pushgateway's endpoint
      --threshold string     threshold of detection (default "24h")
```
### example

```
zombie-detector --incluster=false --pushgateway=<YOUR PUSHGATEWAY ADDRESS> --threshold=24h30m
```
[releases]: https://github.com/cybozu-go/zombie-detector/releases

## Example manifest
We can run zombie-detector periodically as CronJob in a Kubernetes Cluster.

These are example manifests.

cronjob.yaml
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: zombie-detector-cronjob
  namespace: zombie-detector
spec:
  schedule: "0 0 */1 * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: zombie-detector-sa
          containers:
          - name: zombie-detector
            image: zombie-detector:dev
            command:
            - ./zombie-detector
            - --threshold=24h
            - --pushgateway=http://<YOUR PUSHGATEWAY SERVICE ADRESS>.monitoring.svc.cluster.local:9091
          restartPolicy: OnFailure
```
rbac.yaml
```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: zombie-detector-sa
  namespace: zombie-detector
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: zombie-detector-role
rules:
- apiGroups:
  - '*'
  resources:
  - '*'
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - '*'
  resources:
  - '*/*'
  verbs:
  - get
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: zombie-detector-rolebinding
subjects:
  - kind: ServiceAccount
    name: zombie-detector-sa
    namespace: zombie-detector
roleRef:
  kind: ClusterRole
  name: zombie-detector-role
  apiGroup: rbac.authorization.k8s.io

```
