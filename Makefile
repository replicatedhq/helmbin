SHELL := /bin/bash

include common.mk

OS ?= linux
ARCH ?= $(shell go env GOARCH)

APP_NAME = embedded-cluster
ADMIN_CONSOLE_CHART_REPO_OVERRIDE =
ADMIN_CONSOLE_IMAGE_OVERRIDE =
ADMIN_CONSOLE_MIGRATIONS_IMAGE_OVERRIDE =
ADMIN_CONSOLE_KURL_PROXY_IMAGE_OVERRIDE =
K0S_VERSION = v1.32.1+k0s.0
K0S_GO_VERSION = v1.30.9+k0s.0
PREVIOUS_K0S_VERSION ?= v1.29.9+k0s.0-ec.0
PREVIOUS_K0S_GO_VERSION ?= v1.29.9+k0s.0
K0S_BINARY_SOURCE_OVERRIDE =
TROUBLESHOOT_VERSION = v0.116.1

KOTS_VERSION = v$(shell awk '/^version/{print $$2}' pkg/addons/adminconsole/static/metadata.yaml | sed -E 's/([0-9]+\.[0-9]+\.[0-9]+).*/\1/')
# When updating KOTS_BINARY_URL_OVERRIDE, also update the KOTS_VERSION above or
# scripts/ci-upload-binaries.sh may find the version in the cache and not upload the overridden binary.
KOTS_BINARY_URL_OVERRIDE =
# For dev env, build the kots binary in the kots repo with "make kots-linux-arm64" and set this to "../kots/bin/kots"
KOTS_BINARY_FILE_OVERRIDE =
# TODO: move this to a manifest file
LOCAL_ARTIFACT_MIRROR_IMAGE ?= proxy.replicated.com/anonymous/replicated/embedded-cluster-local-artifact-mirror:$(VERSION)
# These are used to override the binary urls in dev and e2e tests
METADATA_K0S_BINARY_URL_OVERRIDE =
METADATA_KOTS_BINARY_URL_OVERRIDE =
METADATA_OPERATOR_BINARY_URL_OVERRIDE =

ifeq ($(K0S_VERSION),v1.30.5+k0s.0-ec.1)
K0S_BINARY_SOURCE_OVERRIDE =
else ifeq ($(K0S_VERSION),v1.29.9+k0s.0-ec.0)
K0S_BINARY_SOURCE_OVERRIDE =
else ifeq ($(K0S_VERSION),v1.28.14+k0s.0-ec.0)
K0S_BINARY_SOURCE_OVERRIDE =
endif

ifneq ($(KOTS_BINARY_FILE_OVERRIDE),)
KOTS_VERSION = kots-dev-$(shell shasum -a 256 $(KOTS_BINARY_FILE_OVERRIDE) | cut -c1-8)
endif

LD_FLAGS = \
	-X github.com/replicatedhq/embedded-cluster/pkg/versions.K0sVersion=$(K0S_VERSION) \
	-X github.com/replicatedhq/embedded-cluster/pkg/versions.Version=$(VERSION) \
	-X github.com/replicatedhq/embedded-cluster/pkg/versions.TroubleshootVersion=$(TROUBLESHOOT_VERSION) \
	-X github.com/replicatedhq/embedded-cluster/pkg/versions.LocalArtifactMirrorImage=$(LOCAL_ARTIFACT_MIRROR_IMAGE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/versions.K0sBinaryURLOverride=$(METADATA_K0S_BINARY_URL_OVERRIDE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/versions.KOTSBinaryURLOverride=$(METADATA_KOTS_BINARY_URL_OVERRIDE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/versions.OperatorBinaryURLOverride=$(METADATA_OPERATOR_BINARY_URL_OVERRIDE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole.ChartRepoOverride=$(ADMIN_CONSOLE_CHART_REPO_OVERRIDE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole.KurlProxyImageOverride=$(ADMIN_CONSOLE_KURL_PROXY_IMAGE_OVERRIDE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole.KotsVersion=$(KOTS_VERSION) \
	-X github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole.AdminConsoleChartRepoOverride=$(ADMIN_CONSOLE_CHART_REPO_OVERRIDE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole.AdminConsoleImageOverride=$(ADMIN_CONSOLE_IMAGE_OVERRIDE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole.AdminConsoleMigrationsImageOverride=$(ADMIN_CONSOLE_MIGRATIONS_IMAGE_OVERRIDE) \
	-X github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole.AdminConsoleKurlProxyImageOverride=$(ADMIN_CONSOLE_KURL_PROXY_IMAGE_OVERRIDE)
DISABLE_FIO_BUILD ?= 0

export PATH := $(shell pwd)/bin:$(PATH)

.DEFAULT_GOAL := default
default: build-ttl.sh

split-hyphen = $(word $2,$(subst -, ,$1))
random-string = $(shell LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c6)

.PHONY: cmd/installer/goods/bins/k0s
cmd/installer/goods/bins/k0s:
	$(MAKE) output/bins/k0s-$(K0S_VERSION)-$(ARCH)
	mkdir -p cmd/installer/goods/bins
	cp output/bins/k0s-$(K0S_VERSION)-$(ARCH) $@

output/bins/k0s-%:
	mkdir -p output/bins
	if [ "$(K0S_BINARY_SOURCE_OVERRIDE)" != "" ]; then \
	    curl --retry 5 --retry-all-errors -fL -o $@ "$(K0S_BINARY_SOURCE_OVERRIDE)" ; \
	else \
	    curl --retry 5 --retry-all-errors -fL -o $@ "https://github.com/k0sproject/k0s/releases/download/$(call split-hyphen,$*,1)/k0s-$(call split-hyphen,$*,1)-$(call split-hyphen,$*,2)" ; \
	fi
	chmod +x $@
	touch $@

.PHONY: cmd/installer/goods/bins/kubectl-support_bundle
cmd/installer/goods/bins/kubectl-support_bundle:
	$(MAKE) output/bins/kubectl-support_bundle-$(TROUBLESHOOT_VERSION)-$(ARCH)
	mkdir -p cmd/installer/goods/bins
	cp output/bins/kubectl-support_bundle-$(TROUBLESHOOT_VERSION)-$(ARCH) $@

output/bins/kubectl-support_bundle-%:
	mkdir -p output/bins
	mkdir -p output/tmp
	curl --retry 5 --retry-all-errors -fL -o output/tmp/support-bundle.tar.gz "https://github.com/replicatedhq/troubleshoot/releases/download/$(call split-hyphen,$*,1)/support-bundle_$(OS)_$(call split-hyphen,$*,2).tar.gz"
	tar -xzf output/tmp/support-bundle.tar.gz -C output/tmp
	mv output/tmp/support-bundle $@
	rm -rf output/tmp
	touch $@

.PHONY: cmd/installer/goods/bins/kubectl-preflight
cmd/installer/goods/bins/kubectl-preflight:
	$(MAKE) output/bins/kubectl-preflight-$(TROUBLESHOOT_VERSION)-$(ARCH)
	mkdir -p cmd/installer/goods/bins
	cp output/bins/kubectl-preflight-$(TROUBLESHOOT_VERSION)-$(ARCH) $@

output/bins/kubectl-preflight-%:
	mkdir -p output/bins
	mkdir -p output/tmp
	curl --retry 5 --retry-all-errors -fL -o output/tmp/preflight.tar.gz https://github.com/replicatedhq/troubleshoot/releases/download/$(call split-hyphen,$*,1)/preflight_$(OS)_$(call split-hyphen,$*,2).tar.gz
	tar -xzf output/tmp/preflight.tar.gz -C output/tmp
	mv output/tmp/preflight $@
	rm -rf output/tmp
	touch $@

.PHONY: cmd/installer/goods/bins/local-artifact-mirror
cmd/installer/goods/bins/local-artifact-mirror:
	mkdir -p cmd/installer/goods/bins
	$(MAKE) -C local-artifact-mirror build OS=$(OS) ARCH=$(ARCH)
	cp local-artifact-mirror/bin/local-artifact-mirror-$(OS)-$(ARCH) $@
	touch $@

ifndef FIO_VERSION
FIO_VERSION = $(shell curl --retry 5 --retry-all-errors -fsSL https://api.github.com/repos/axboe/fio/releases/latest | jq -r '.tag_name' | cut -d- -f2)
endif

output/bins/fio-%:
	mkdir -p output/bins
	docker build -t fio --build-arg FIO_VERSION=$(call split-hyphen,$*,1) --build-arg PLATFORM=$(OS)/$(call split-hyphen,$*,2) fio
	docker rm -f fio && docker run --name fio fio
	docker cp fio:/output/fio $@
	docker rm -f fio
	touch $@

.PHONY: cmd/installer/goods/bins/fio
cmd/installer/goods/bins/fio:
ifneq ($(DISABLE_FIO_BUILD),1)
	$(MAKE) output/bins/fio-$(FIO_VERSION)-$(ARCH)
	mkdir -p cmd/installer/goods/bins
	cp output/bins/fio-$(FIO_VERSION)-$(ARCH) $@
endif

.PHONY: cmd/installer/goods/internal/bins/kubectl-kots
cmd/installer/goods/internal/bins/kubectl-kots:
	mkdir -p cmd/installer/goods/internal/bins
	if [ "$(KOTS_BINARY_URL_OVERRIDE)" != "" ]; then \
		$(MAKE) output/bins/kubectl-kots-override ; \
		cp output/bins/kubectl-kots-override $@ ; \
	elif [ "$(KOTS_BINARY_FILE_OVERRIDE)" != "" ]; then \
		cp $(KOTS_BINARY_FILE_OVERRIDE) $@ ; \
	else \
		$(MAKE) output/bins/kubectl-kots-$(KOTS_VERSION)-$(ARCH) ; \
		cp output/bins/kubectl-kots-$(KOTS_VERSION)-$(ARCH) $@ ; \
	fi
	touch $@

output/bins/kubectl-kots-%:
	mkdir -p output/bins
	mkdir -p output/tmp
	curl --retry 5 --retry-all-errors -fL -o output/tmp/kots.tar.gz "https://github.com/replicatedhq/kots/releases/download/$(call split-hyphen,$*,1)/kots_$(OS)_$(call split-hyphen,$*,2).tar.gz"
	tar -xzf output/tmp/kots.tar.gz -C output/tmp
	mv output/tmp/kots $@
	touch $@

.PHONY: output/bins/kubectl-kots-override
output/bins/kubectl-kots-override:
	mkdir -p output/bins
	mkdir -p output/tmp
	curl --retry 5 --retry-all-errors -fL -o output/tmp/kots.tar.gz "$(KOTS_BINARY_URL_OVERRIDE)"
	tar -xzf output/tmp/kots.tar.gz -C output/tmp
	mv output/tmp/kots $@
	touch $@

.PHONY: output/bin/embedded-cluster-release-builder
output/bin/embedded-cluster-release-builder:
	mkdir -p output/bin
	CGO_ENABLED=0 go build -o output/bin/embedded-cluster-release-builder e2e/embedded-cluster-release-builder/main.go

.PHONY: initial-release
initial-release: export EC_VERSION = $(VERSION)-$(CURRENT_USER)
initial-release: export APP_VERSION = appver-dev-$(call random-string)
initial-release: export RELEASE_YAML_DIR = e2e/kots-release-install
initial-release: export V2_ENABLED = 0
initial-release: check-env-EC_VERSION check-env-APP_VERSION
	UPLOAD_BINARIES=0 \
		./scripts/build-and-release.sh

.PHONY: rebuild-release
rebuild-release: export EC_VERSION = $(VERSION)-$(CURRENT_USER)
rebuild-release: export RELEASE_YAML_DIR = e2e/kots-release-install
rebuild-release: check-env-EC_VERSION check-env-APP_VERSION
	UPLOAD_BINARIES=0 \
	SKIP_RELEASE=1 \
		./scripts/build-and-release.sh

.PHONY: upgrade-release
upgrade-release: RANDOM_STRING = $(call random-string)
upgrade-release: export EC_VERSION = $(VERSION)-$(CURRENT_USER)-upgrade-$(RANDOM_STRING)
upgrade-release: export APP_VERSION = appver-dev-$(call random-string)-upgrade-$(RANDOM_STRING)
upgrade-release: export V2_ENABLED = 0
upgrade-release: check-env-EC_VERSION check-env-APP_VERSION
	UPLOAD_BINARIES=1 \
	RELEASE_YAML_DIR=e2e/kots-release-upgrade \
		./scripts/build-and-release.sh

.PHONY: go.mod
go.mod: Makefile
	go get github.com/k0sproject/k0s@$(K0S_GO_VERSION)
	go mod tidy

.PHONY: static
static: cmd/installer/goods/bins/k0s \
	cmd/installer/goods/bins/kubectl-preflight \
	cmd/installer/goods/bins/kubectl-support_bundle \
	cmd/installer/goods/bins/local-artifact-mirror \
	cmd/installer/goods/bins/fio \
	cmd/installer/goods/internal/bins/kubectl-kots

.PHONY: static-dryrun
static-dryrun:
	@mkdir -p cmd/installer/goods/bins cmd/installer/goods/internal/bins
	@touch cmd/installer/goods/bins/k0s \
		cmd/installer/goods/bins/kubectl-preflight \
		cmd/installer/goods/bins/kubectl-support_bundle \
		cmd/installer/goods/bins/local-artifact-mirror \
		cmd/installer/goods/bins/fio \
		cmd/installer/goods/internal/bins/kubectl-kots

.PHONY: embedded-cluster-linux-amd64
embedded-cluster-linux-amd64: export OS = linux
embedded-cluster-linux-amd64: export ARCH = amd64
embedded-cluster-linux-amd64: static go.mod embedded-cluster
	mkdir -p ./output/bin
	cp ./build/embedded-cluster-$(OS)-$(ARCH) ./output/bin/$(APP_NAME)

.PHONY: embedded-cluster-linux-arm64
embedded-cluster-linux-arm64: export OS = linux
embedded-cluster-linux-arm64: export ARCH = arm64
embedded-cluster-linux-arm64: static go.mod embedded-cluster
	mkdir -p ./output/bin
	cp ./build/embedded-cluster-$(OS)-$(ARCH) ./output/bin/$(APP_NAME)

.PHONY: embedded-cluster-darwin-arm64
embedded-cluster-darwin-arm64: export OS = darwin
embedded-cluster-darwin-arm64: export ARCH = arm64
embedded-cluster-darwin-arm64: go.mod embedded-cluster
	mkdir -p ./output/bin
	cp ./build/embedded-cluster-$(OS)-$(ARCH) ./output/bin/$(APP_NAME)

.PHONY: embedded-cluster
embedded-cluster:
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build \
		-tags osusergo,netgo \
		-ldflags="-s -w $(LD_FLAGS) -extldflags=-static" \
		-o ./build/embedded-cluster-$(OS)-$(ARCH) \
		./cmd/installer

.PHONY: envtest
envtest:
	$(MAKE) -C operator manifests envtest

.PHONY: unit-tests
unit-tests: envtest
	mkdir -p cmd/installer/goods/bins cmd/installer/goods/internal/bins
	touch cmd/installer/goods/bins/BUILD cmd/installer/goods/internal/bins/BUILD # compilation will fail if no files are present
	KUBEBUILDER_ASSETS="$(shell ./operator/bin/setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/operator/bin -p path)" \
		go test -tags exclude_graphdriver_btrfs -v ./pkg/... ./cmd/...
	$(MAKE) -C operator test

.PHONY: vet
vet: static
	go vet -tags exclude_graphdriver_btrfs ./...

.PHONY: e2e-tests
e2e-tests: embedded-release
	go test -tags exclude_graphdriver_btrfs -timeout 60m -ldflags="$(LD_FLAGS)" -parallel 1 -failfast -v ./e2e

.PHONY: e2e-test
e2e-test:
	go test -tags exclude_graphdriver_btrfs -timeout 60m -ldflags="$(LD_FLAGS)" -v ./e2e -run ^$(TEST_NAME)$$

.PHONY: dryrun-tests
dryrun-tests: export DRYRUN_MATCH = Test
dryrun-tests: static-dryrun
	@./scripts/dryrun-tests.sh

.PHONY: build-ttl.sh
build-ttl.sh:
	$(MAKE) -C local-artifact-mirror build-ttl.sh \
		IMAGE_NAME=$(CURRENT_USER)/embedded-cluster-local-artifact-mirror
	make embedded-cluster-linux-$(ARCH) \
		LOCAL_ARTIFACT_MIRROR_IMAGE=proxy.replicated.com/anonymous/$(shell cat local-artifact-mirror/build/image)

.PHONY: clean
clean:
	rm -rf output
	rm -rf cmd/installer/goods/bins/*
	rm -rf cmd/installer/goods/internal/bins/*
	rm -rf build
	rm -rf bin

.PHONY: lint
lint:
	golangci-lint run -c .golangci.yml ./... --build-tags exclude_graphdriver_btrfs

.PHONY: lint-and-fix
lint-and-fix:
	golangci-lint run --fix -c .golangci.yml ./... --build-tags exclude_graphdriver_btrfs

.PHONY: scan
scan:
	trivy fs \
		--scanners vuln \
		--exit-code=1 \
		--severity="HIGH,CRITICAL" \
		--ignore-unfixed \
		./

.PHONY: buildtools
buildtools:
	mkdir -p cmd/installer/goods/bins cmd/installer/goods/internal/bins
	touch cmd/installer/goods/bins/BUILD cmd/installer/goods/internal/bins/BUILD # compilation will fail if no files are present
	go build -tags exclude_graphdriver_btrfs -o ./output/bin/buildtools ./cmd/buildtools

.PHONY: list-distros
list-distros:
	@$(MAKE) -C dev/distros list

.PHONY: create-node%
create-node%: DISTRO = debian-bookworm
create-node%: NODE_PORT = 30000
create-node%: K0S_DATA_DIR = /var/lib/embedded-cluster/k0s
create-node%:
	@docker run -d \
		--name node$* \
		--hostname node$* \
		--privileged \
		--restart=unless-stopped \
		-v $(K0S_DATA_DIR) \
		-v $(shell pwd):/replicatedhq/embedded-cluster \
		-v $(shell dirname $(shell pwd))/kots:/replicatedhq/kots \
		$(if $(filter node0,node$*),-p $(NODE_PORT):$(NODE_PORT)) \
		$(if $(filter node0,node$*),-p 30003:30003) \
		replicated/ec-distro:$(DISTRO)

	@$(MAKE) ssh-node$*

.PHONY: ssh-node%
ssh-node%:
	@docker exec -it -w /replicatedhq/embedded-cluster node$* bash

.PHONY: delete-node%
delete-node%:
	@docker rm -f --volumes node$*

.PHONY: %-up
%-up:
	@dev/scripts/up.sh $*

.PHONY: %-down
%-down:
	@dev/scripts/down.sh $*

.PHONY: test-lam-e2e
test-lam-e2e: cmd/installer/goods/bins/local-artifact-mirror
	sudo go test ./cmd/local-artifact-mirror/e2e/... -v

.PHONY: bin/installer
bin/installer:
	@mkdir -p bin
	go build -o bin/installer ./cmd/installer

# make test-embed channel=<channelid> app=<appslug>
.PHONY: test-embed
test-emded: export OS=linux
test-embed: export ARCH=amd64
test-embed: VERSION=1.19.0+k8s-1.30
test-embed: static embedded-cluster
	@echo "Cleaning up previous release directory..."
	rm -rf ./hack/release
	@echo "Creating release directory..."
	mkdir -p ./hack/release

	@echo "Fetching channel JSON for channel: $(channel)"
	$(eval CHANNEL_JSON := $(shell replicated channel inspect $(channel) --output json))
	@echo "CHANNEL_JSON: $(CHANNEL_JSON)"

	@echo "Extracting release label, sequence, and channel slug..."
	$(eval RELEASE_LABEL := $(shell echo '$(CHANNEL_JSON)' | jq -r '.releaseLabel'))
	$(eval RELEASE_SEQUENCE := $(shell echo '$(CHANNEL_JSON)' | jq -r '.releaseSequence'))
	$(eval CHANNEL_SLUG := $(shell echo '$(CHANNEL_JSON)' | jq -r '.channelSlug'))
	$(eval CHANNEL_ID := $(shell echo '$(CHANNEL_JSON)' | jq -r '.id'))

	@echo "Extracted values:"
	@echo "  RELEASE_LABEL: $(RELEASE_LABEL)"
	@echo "  RELEASE_SEQUENCE: $(RELEASE_SEQUENCE)"
	@echo "  CHANNEL_SLUG: $(CHANNEL_SLUG)"

	@echo "Downloading release sequence $(RELEASE_SEQUENCE) for app $(app)..."
	replicated release download $(RELEASE_SEQUENCE) --app=$(app) -d ./hack/release || { echo "Error: Failed to download release. Check RELEASE_SEQUENCE or app name."; exit 1; }

	@echo "Writing release.yaml..."
	@mkdir -p ./hack/release  # Ensure directory exists
	@echo '# channel release object' > ./hack/release/release.yaml
	@echo 'channelID: "${CHANNEL_ID}"' >> ./hack/release/release.yaml
	@echo 'channelSlug: "${CHANNEL_SLUG}"' >> ./hack/release/release.yaml
	@echo 'appSlug: "$(app)"' >> ./hack/release/release.yaml
	@echo 'versionLabel: "${RELEASE_LABEL}"' >> ./hack/release/release.yaml

	@echo "Creating tarball of the release directory..."
	tar czvf ./hack/release.tgz -C ./hack/release .

	@echo "Embedding release into binary..."
	go run ./hack/dev-embed.go --binary ./build/embedded-cluster-linux-amd64  --release ./hack/release.tgz --output ./build/$(app) \
		--label $(RELEASE_LABEL) --sequence $(RELEASE_SEQUENCE) --channel $(CHANNEL_SLUG)

	chmod +x ./build/$(app)
	@echo "Test embed completed successfully."
