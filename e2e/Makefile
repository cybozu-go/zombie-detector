ARCH ?= amd64
OS ?= linux

PROJECT_DIR := $(CURDIR)/../
BIN_DIR := $(PROJECT_DIR)/bin

CURL := curl -sSLf
GH := $(BIN_DIR)/gh
YQ := $(BIN_DIR)/yq
KIND := $(BIN_DIR)/kind
KUBECTL := $(BIN_DIR)/kubectl
KUSTOMIZE := $(BIN_DIR)/kustomize

KIND_CLUSTER_NAME := zombie-detector-cluster
KIND_CONFIG := kind-config.yaml
KUBECONFIG := $(CURDIR)/.kubeconfig
export KUBECONFIG

include $(PROJECT_DIR)/Makefile.common
include Makefile.versions

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
setup: kind kubectl kustomize

.PHONY: start
start:
	$(KIND) create cluster --name=$(KIND_CLUSTER_NAME) --config=$(KIND_CONFIG) --image=kindest/node:v$(E2ETEST_K8S_VERSION) --wait 1m
	$(MAKE) -C $(PROJECT_DIR) docker-build
	$(KIND) load docker-image zombie-detector:dev --name=$(KIND_CLUSTER_NAME)
	$(KUSTOMIZE) build ../config/namespace | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build ../config/rbac | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build ./manifests/pushgateway | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build ./manifests/metrics-server | $(KUBECTL) apply -f -

.PHONY: test
test:
	env RUN_E2E=1 \
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

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: kind
kind: $(KIND)
$(KIND): $(BIN_DIR)
	$(CURL) -o $@ https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-$(OS)-$(ARCH)
	chmod +x $@

.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(BIN_DIR)
	$(CURL) https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh \
	| bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(BIN_DIR)

.PHONY: kubectl
kubectl: $(KUBECTL)
$(KUBECTL): $(BIN_DIR)
	$(CURL) -o $@ https://dl.k8s.io/release/v$(E2ETEST_K8S_VERSION)/bin/$(OS)/$(ARCH)/kubectl
	chmod +x $@

# This target is assumed to be executed from the parent Makefile.
# If want to run this target alone, please configure gh, etc. using the following command.
# $ make -C ../ setup login-gh
.PHONY: maintenance
maintenance:
	$(MAKE) update-tools-version update-pushgateway update-metrics-server

.PHONY: update-tools-version
update-tools-version:
	$(call get-latest-gh,kubernetes-sigs/kind)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2-); \
	sed -i -e "s/KIND_VERSION := .*/KIND_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,kubernetes-sigs/kustomize)
	NEW_VERSION=$$(echo $(notdir $(latest_gh)) | cut -b 2-); \
	sed -i -e "s/KUSTOMIZE_VERSION := .*/KUSTOMIZE_VERSION := $${NEW_VERSION}/g" Makefile.versions

.PHONY: update-pushgateway
update-pushgateway:
	$(call get-latest-gh-package-tag,pushgateway)
	NEW_VERSION=$$(echo $(latest_tag)); \
	$(YQ) -i "(.images[] | select(.name==\"ghcr.io/cybozu/pushgateway\")).newTag = \"$(latest_tag)\"" manifests/pushgateway/kustomization.yaml

.PHONY: update-metrics-server
update-metrics-server:
	$(CURL) -o manifests/metrics-server/components.yaml \
		https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
