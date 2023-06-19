include Makefile.versions

ARCH ?= amd64
OS ?= linux

TAG ?= dev
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

LOCALBIN ?= $(shell pwd)/bin
GH := $(LOCALBIN)/gh
YQ := $(LOCALBIN)/yq

$(LOCALBIN):
	mkdir -p $(LOCALBIN)
ENVTEST ?= $(LOCALBIN)/setup-envtest


.PHONY: test
test: envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out -v

.PHONY: setup
setup: envtest

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: docker-build
docker-build:
	docker build -t zombie-detector:$(TAG) .

.PHONY: maintenance
maintenance: update-tools-versions update-actions
	$(MAKE) -C ./e2e update-tools-version

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

.PHONY: gh
gh: $(GH)
$(GH): $(LOCALBIN)
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
$(YQ):
	mkdir -p $(LOCALBIN)
	wget -qO $@ https://github.com/mikefarah/yq/releases/download/v$(YQ_VERSION)/yq_$(OS)_$(ARCH)
	chmod +x $@

# usage get-latest-gh OWNER/REPO
define get-latest-gh
$(eval latest_gh := $(shell $(GH) release list --repo $1 | grep Latest | cut -f3))
endef
