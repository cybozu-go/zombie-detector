PROJECT_DIR := $(CURDIR)/../

KIND_CLUSTER_NAME := zombie-detector-cluster
KIND_CONFIG := kind-config.yaml
KUBECONFIG := $(CURDIR)/.kubeconfig
export KUBECONFIG

include $(PROJECT_DIR)/Makefile.common
include $(PROJECT_DIR)/Makefile.versions

.PHONY: help
help:
	@echo "Choose one of the following target"
	@echo
	@echo "setup      Setup tools"
	@echo "start      Start kind cluster and install accurate"
	@echo "test       Run e2e tests"
	@echo "logs       Save logs as logs.tar.gz"
	@echo "stop       Stop the kind cluster"

.PHONY: setup
setup:
	$(MAKE) -C $(PROJECT_DIR) setup

.PHONY: start
start:
	$(KIND) create cluster --name=$(KIND_CLUSTER_NAME) --config=$(KIND_CONFIG) --image=kindest/node:v$(E2ETEST_K8S_VERSION)@$(E2ETEST_KINDEST_NODE_IMAGE_HASH) --wait 1m
	$(MAKE) -C $(PROJECT_DIR) docker-build
	$(KIND) load docker-image zombie-detector:dev --name=$(KIND_CLUSTER_NAME)
	$(KUSTOMIZE) build ../config/namespace | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build ../config/rbac | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build ./manifests/pushgateway | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build ./manifests/metrics-server | $(KUBECTL) apply -f -

.PHONY: test
test:
	env RUN_E2E=1 KUBECTL="$$(aqua which kubectl)" \
		go test -v -race . -ginkgo.v -ginkgo.fail-fast

.PHONY: logs
logs:
	rm -rf logs.tar.gz logs
	$(KIND) export logs --name=$(KIND_CLUSTER_NAME) ./logs
	tar czf logs.tar.gz logs
	rm -rf logs

.PHONY: stop
stop:
	$(KIND) delete cluster --name=$(KIND_CLUSTER_NAME)
	-docker image rm zombie-detector:dev
	-docker image prune -f
