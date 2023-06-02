include Makefile.versions

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

GH := $(BIN_DIR)/gh
YQ := $(BIN_DIR)/yq

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

.PHONY: update-tools-version
update-tools-version: login-gh
	$(call get-latest-gh,actions/cache)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2); \
	sed -i -e "s/ACTIONS_CACHE_VERSION := .*/ACTIONS_CACHE_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,actions/checkout)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2); \
	sed -i -e "s/ACTIONS_CHECKOUT_VERSION := .*/ACTIONS_CHECKOUT_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,actions/setup-go)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2); \
	sed -i -e "s/ACTIONS_SETUP_GO_VERSION := .*/ACTIONS_SETUP_GO_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,kubernetes-sigs/kind)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2-); \
	sed -i -e "s/KIND_VERSION := .*/KIND_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,kubernetes-sigs/kustomize)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2-); \
	sed -i -e "s/KUSTOMIZE_VERSION := .*/KUSTOMIZE_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,mikefarah/yq)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2-); \
	sed -i -e "s/YQ_VERSION := .*/YQ_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-tag,pushgateway)
	NEW_VERSION=$$(docker run quay.io/cybozu/pushgateway:$(latest_tag) kustomize version | grep -o -e 'Version:[^ ]*' | cut -d'/' -f2 | cut -b 2-); \
	sed -i -e "s/PUSHGATEWAY_IMAGE_VERSION := .*/PUSHGATEWAY_IMAGE_VERSION := $${NEW_VERSION}/g" Makefile.versions

# usage: get-latest-tag NAME
define get-latest-tag
$(eval latest_tag := $(shell wget -O - https://quay.io/api/v1/repository/cybozu/$1/tag/ | jq -r '.tags[] | .name' | awk '/.*\..*\./ {print $$1; exit}'))
endef

# usage get-latest-gh OWNER/REPO
define get-latest-gh
$(eval latest_gh := $(shell $(GH) release list --repo $1 | grep Latest | cut -f3))
endef

$(GH):
	mkdir -p $(BIN_DIR)
	wget -qO - https://github.com/cli/cli/releases/download/v$(GH_VERSION)/gh_$(GH_VERSION)_$(OS)_$(ARCH).tar.gz | tar -zx -O gh_$(GH_VERSION)_$(OS)_$(ARCH)/bin/gh > $@
	chmod +x $@

.PHONY: login-gh
login-gh:
	if ! $(GH) auth status 2>/dev/null; then \
		echo; \
		echo '!! You need login to GitHub to proceed. Please follow the next command with "Authenticate Git with your GitHub credentials? (Y)".'; \
		echo; \
		$(GH) auth login -h github.com -p HTTPS -w; \
	fi


.PHONY: logout-gh
logout-gh:
	$(GH) auth logout

$(YQ):
	mkdir -p $(BIN_DIR)
	wget -qO $@ https://github.com/mikefarah/yq/releases/download/v$(YQ_VERSION)/yq_$(OS)_$(ARCH)
	chmod +x $@
