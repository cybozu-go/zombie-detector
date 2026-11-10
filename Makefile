TAG ?= dev

BIN_DIR := $(CURDIR)/bin

include Makefile.common
include Makefile.versions

.PHONY: setup
setup: ## Install tools with aqua.
	aqua i -l

.PHONY: test
test: ## Run tests.
	KUBEBUILDER_ASSETS="$$($(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(BIN_DIR) -p path)" go test ./... -coverprofile cover.out -v

.PHONY: docker-build
docker-build:
ifneq ($(wildcard $(HOME)/.netrc),)
	# To pass Takumi Guard credentials, mount the .netrc file to the container if it exists.
	docker build --build-arg GOPROXY=$(shell go env GOPROXY) --secret id=netrc,src=$(HOME)/.netrc -t zombie-detector:$(TAG) .
else
	docker build --build-arg GOPROXY=$(shell go env GOPROXY) -t zombie-detector:$(TAG) .
endif

# update-aqua is completed first, because the other targets use the tools updated by it.
.PHONY: maintenance
maintenance:
	$(MAKE) update-aqua
	$(MAKE) update-pushgateway update-metrics-server

# usage: update-aqua-package PACKAGE [TAG_PREFIX]
# Update the package version in aqua.yaml and its checksums in aqua-checksums.json.
# The checksums are updated each time, because aqua cannot execute a package whose checksum is not recorded.
# PACKAGE must be the same as the GitHub repository name.
define update-aqua-package
	$(call get-latest-gh,$1,$2)
	$(YQ) -i '(.packages[] | select(.name | test("^$1@"))).name = "$1@$(latest_gh)"' aqua.yaml
	aqua update-checksum
endef

.PHONY: update-aqua
update-aqua: login-gh
	$(call update-aqua-package,cli/cli)
	$(call update-aqua-package,mikefarah/yq)
	$(call update-aqua-package,kubernetes-sigs/kind)
	$(call update-aqua-package,kubernetes-sigs/kustomize,kustomize/)

	$(call get-latest-gh,aquaproj/aqua-registry)
	$(YQ) -i '(.registries[] | select(.type == "standard")).ref = "$(latest_gh)"' aqua.yaml

	$(call get-latest-gh,aquaproj/aqua)
	sed -i -e 's/aqua_version: .*/aqua_version: $(latest_gh)/' .github/workflows/*.yaml

	aqua update-checksum -prune
	aqua i -l

.PHONY: update-pushgateway
update-pushgateway: login-gh
	$(call get-latest-gh-package-tag,pushgateway)
	$(YQ) -i '(.images[] | select(.name == "ghcr.io/cybozu/pushgateway")).newTag = "$(latest_tag)"' e2e/manifests/pushgateway/kustomization.yaml

METRICS_SERVER_IMAGE := registry.k8s.io/metrics-server/metrics-server
METRICS_SERVER_MANIFEST := e2e/manifests/metrics-server/components.yaml
# The list of image digests promoted to registry.k8s.io.
METRICS_SERVER_IMAGES_YAML := https://raw.githubusercontent.com/kubernetes/k8s.io/main/registry.k8s.io/images/k8s-staging-metrics-server/images.yaml

# Update the metrics-server manifest and pin the image by tag and digest.
# - Helm chart releases (metrics-server-helm-chart-*) are excluded with the "v" prefix.
# - The source URL of the downloaded manifest is added as a header comment.
# - The image digest is taken from the image promotion manifest of registry.k8s.io (METRICS_SERVER_IMAGES_YAML).
.PHONY: update-metrics-server
update-metrics-server: login-gh
	$(call get-latest-gh,kubernetes-sigs/metrics-server,v)
	$(eval image_digest := $(shell curl -sSfL $(METRICS_SERVER_IMAGES_YAML) \
		| $(YQ) '.[] | select(.name == "metrics-server") | .dmap | to_entries[] | select(.value[] == "$(latest_gh)") | .key'))
	$(if $(filter 1,$(words $(image_digest))),,$(error expected exactly one digest of $(METRICS_SERVER_IMAGE):$(latest_gh), but got "$(image_digest)"))
	curl -sSLf -o $(METRICS_SERVER_MANIFEST).tmp \
		https://github.com/kubernetes-sigs/metrics-server/releases/download/$(latest_gh)/components.yaml
	echo "# Downloaded from https://github.com/kubernetes-sigs/metrics-server/releases/download/$(latest_gh)/components.yaml" > $(METRICS_SERVER_MANIFEST)
	cat $(METRICS_SERVER_MANIFEST).tmp  >> $(METRICS_SERVER_MANIFEST)
	rm $(METRICS_SERVER_MANIFEST).tmp
	$(YQ) -i '(.images[] | select(.name == "$(METRICS_SERVER_IMAGE)")) |= {"name": .name, "newTag": "$(latest_gh)", "digest": "$(image_digest)"}' e2e/manifests/metrics-server/kustomization.yaml

.PHONY: login-gh
login-gh:
	if ! $(GH) auth status 2>/dev/null; then \
		echo; \
		echo '!! You need login to GitHub to proceed. Please follow the next command with "Authenticate Git with your GitHub credentials? (Y)".'; \
		echo; \
		$(GH) auth login -h github.com -p HTTPS -w -s read:packages; \
	fi

.PHONY: logout-gh
logout-gh:
	$(GH) auth logout
