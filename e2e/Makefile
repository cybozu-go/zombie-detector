E2ETEST_K8S_VERSION  := 1.26.3
KIND_VERSION=0.19.0

ARCH ?= amd64
OS ?= linux

PROJECT_DIR := $(CURDIR)/../
BIN_DIR := $(PROJECT_DIR)/bin

CURL := curl -sSLf
KUBECTL := $(BIN_DIR)/kubectl
KUSTOMIZE := $(BIN_DIR)/kustomize

KIND := $(BIN_DIR)/kind
KIND_CLUSTER_NAME := zombie-detector-cluster
KIND_CONFIG := kind-config.yaml

export KUBECONFIG

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
setup: bin_dir kubectl kustomize kind 

.PHONY: start
start:
	$(KIND) create cluster --name=$(KIND_CLUSTER_NAME) --config=$(KIND_CONFIG) --image=kindest/node:v$(E2ETEST_K8S_VERSION) --wait 1m
	$(MAKE) -C ../ docker-build
	$(KIND) load docker-image zombie-detector:dev --name=$(KIND_CLUSTER_NAME)
	$(KUSTOMIZE) build ../config/namespace | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build ../config/rbac | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build ./manifests/pushgateway | $(KUBECTL) apply -f -



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

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(BIN_DIR)
	$(CURL) $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(BIN_DIR)

.PHONY: kind
kind: $(KIND)
$(KIND): $(BIN_DIR)
	$(CURL) -o $(KIND) https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-$(OS)-$(ARCH)
	chmod a+x $(KIND)

.PHONY: kubectl
kubectl: $(KUBECTL)
$(KUBECTL): $(BIN_DIR)
	$(CURL) -o $(BIN_DIR)/kubectl https://storage.googleapis.com/kubernetes-release/release/v$(E2ETEST_K8S_VERSION)/bin/$(OS)/$(ARCH)/kubectl && chmod a+x $(BIN_DIR)/kubectl

.PHONY: bin_dir
$(BIN_DIR):
	mkdir -p $(BIN_DIR)