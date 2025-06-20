ARCH ?= amd64
OS ?= linux

TAG ?= dev

BIN_DIR := $(CURDIR)/bin

CURL := curl -sSLf
GH := $(BIN_DIR)/gh
YQ := $(BIN_DIR)/yq
ENVTEST := $(BIN_DIR)/setup-envtest

include Makefile.common
include Makefile.versions

.PHONY: setup
setup: gh yq envtest

.PHONY: test
test: envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(BIN_DIR) -p path)" go test ./... -coverprofile cover.out -v

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(BIN_DIR)
	test -s $@ || GOBIN=$(BIN_DIR) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: docker-build
docker-build:
	docker build -t zombie-detector:$(TAG) .

.PHONY: maintenance
maintenance:
	$(MAKE) update-tools-versions
	$(MAKE) update-actions
	$(MAKE) -C ./e2e maintenance

.PHONY: update-tools-versions
update-tools-versions: login-gh
	$(call get-latest-gh,actions/cache)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2); \
	sed -i -e "s/ACTIONS_CACHE_VERSION := .*/ACTIONS_CACHE_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,actions/checkout)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2); \
	sed -i -e "s/ACTIONS_CHECKOUT_VERSION := .*/ACTIONS_CHECKOUT_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,actions/setup-go)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2); \
	sed -i -e "s/ACTIONS_SETUP_GO_VERSION := .*/ACTIONS_SETUP_GO_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,cli/cli)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2-); \
	sed -i -e "s/GH_VERSION := .*/GH_VERSION := $${NEW_VERSION}/g" Makefile.versions

	$(call get-latest-gh,mikefarah/yq)
	NEW_VERSION=$$(echo $(latest_gh) | cut -b 2-); \
	sed -i -e "s/YQ_VERSION := .*/YQ_VERSION := $${NEW_VERSION}/g" Makefile.versions

.PHONY: update-actions
update-actions:
	$(YQ) -i '(.. | select(has("uses")) | select(.uses | contains("actions/cache"))).uses = "actions/cache@v$(ACTIONS_CACHE_VERSION)"' .github/workflows/ci.yaml
	$(YQ) -i '(.. | select(has("uses")) | select(.uses | contains("actions/cache"))).uses = "actions/cache@v$(ACTIONS_CACHE_VERSION)"' .github/workflows/release.yaml
	$(YQ) -i '(.. | select(has("uses")) | select(.uses | contains("actions/checkout"))).uses = "actions/checkout@v$(ACTIONS_CHECKOUT_VERSION)"' .github/workflows/ci.yaml
	$(YQ) -i '(.. | select(has("uses")) | select(.uses | contains("actions/checkout"))).uses = "actions/checkout@v$(ACTIONS_CHECKOUT_VERSION)"' .github/workflows/release.yaml
	$(YQ) -i '(.. | select(has("uses")) | select(.uses | contains("actions/setup-go"))).uses = "actions/setup-go@v$(ACTIONS_SETUP_GO_VERSION)"' .github/workflows/ci.yaml
	$(YQ) -i '(.. | select(has("uses")) | select(.uses | contains("actions/setup-go"))).uses = "actions/setup-go@v$(ACTIONS_SETUP_GO_VERSION)"' .github/workflows/release.yaml

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: gh
gh: $(GH)
$(GH): $(BIN_DIR)
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

.PHONY: yq
yq: $(YQ)
$(YQ): $(BIN_DIR)
	wget -qO $@ https://github.com/mikefarah/yq/releases/download/v$(YQ_VERSION)/yq_$(OS)_$(ARCH)
	chmod +x $@
