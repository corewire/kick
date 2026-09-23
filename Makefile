IMG ?= ghcr.io/corewire/kick:dev

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOVULNCHECK ?= $(LOCALBIN)/govulncheck
CHAINSAW ?= $(LOCALBIN)/chainsaw
KAMERA ?= $(LOCALBIN)/kamera
KIND ?= kind
TILT ?= tilt

include hack/tool-versions.mk

# Tools are built with the Go version the project targets; golangci-lint refuses
# to lint a Go version newer than the one it was built with.
GO_TOOLCHAIN := go$(shell awk '/^go /{print $$2}' go.mod)

KIND_CLUSTER_NAME ?= kick-dev
KIND_CONTEXT ?= kind-$(KIND_CLUSTER_NAME)
KIND_KUBECONFIG ?= $(shell pwd)/.kubeconfig-kind-kick-dev
IMAGE_ARCHIVE ?= $(shell pwd)/dist/kick-image.tar
E2E_NAMESPACE ?= kick-e2e
PKGS := $(shell go list ./... | grep -v '/ai-docs/' || true)

# Scenario IDs per e2e suite. `core` is everything not claimed by a suite that
# needs extra infrastructure, so adding a suite here removes it from core
# automatically instead of requiring two edits that can drift apart.
E2E_IDS_ARGOCD ?= 024 025 026 027 028 029 030 031 032 033 034 035 036 037 038 039 040 041 042
E2E_IDS_RECOVERY ?= 048 049 050 051
E2E_IDS_ROLLOUTS ?= 060 061 062 063
E2E_IDS_CSI ?= 064 065 066 067
E2E_IDS_KARGO ?= 068 069 070 071
E2E_IDS_NONCORE := $(E2E_IDS_ARGOCD) $(E2E_IDS_RECOVERY) $(E2E_IDS_ROLLOUTS) $(E2E_IDS_CSI) $(E2E_IDS_KARGO)
QUICK_GO_TEST_REGEX ?= KickPolicy|RegistryGateResolver
QUICK_E2E ?= 073

E2E_CHAINSAW_CONFIG := test/e2e/chainsaw-configuration.yaml
E2E_CHAINSAW_CONFIG_INTEGRATION := test/e2e/chainsaw-configuration-integration.yaml

# Per-scenario timings. Off by default so local runs stay side-effect free; set
# E2E_REPORT_DIR (CI does) to write one JUnit report per suite, which
# tools/e2e_timing_summary.py turns into a slowest-scenarios table.
E2E_REPORT_DIR ?=
E2E_REPORT_NAME ?= all
E2E_REPORT_FLAGS = $(if $(E2E_REPORT_DIR),--report-format JUNIT-TEST --report-path $(E2E_REPORT_DIR) --report-name $(E2E_REPORT_NAME),)
# chainsaw does not create the report directory and fails after the run if it is
# missing, which would turn a green suite into a red job.
E2E_REPORT_MKDIR = $(if $(E2E_REPORT_DIR),mkdir -p $(E2E_REPORT_DIR);,)

empty :=
space := $(empty) $(empty)
# Turn "024 025" into "/KICK-E2E-(024|025)\b".
e2e_regex = /KICK-E2E-($(subst $(space),|,$(strip $(1))))\b

# $(1) suite name, $(2) grep flags, $(3) ID list, $(4) chainsaw config
define e2e_suite
@scenario_dirs="$$(find test/e2e/scenarios -mindepth 1 -maxdepth 1 -type d | sort | grep $(2) '$(call e2e_regex,$(3))')"; \
if [[ -z "$$scenario_dirs" ]]; then \
	echo "no $(1) scenarios selected"; \
	exit 1; \
fi; \
$(E2E_REPORT_MKDIR) \
KUBECONFIG=$(KIND_KUBECONFIG) $(CHAINSAW) test --config $(4) --kube-context $(KIND_CONTEXT) $(E2E_REPORT_FLAGS) $$scenario_dirs
endef

.PHONY: fmt
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*' -not -path './ai-docs/*')

.PHONY: vet
vet:
	go vet $(PKGS)

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: static-check
static-check: golangci-lint
	$(GOLANGCI_LINT) run --config .golangci.static.yml

.PHONY: shellcheck
shellcheck:
	@command -v shellcheck >/dev/null || { echo "shellcheck not installed"; exit 1; }
	shellcheck -x $$(find hack test -name '*.sh')

.PHONY: test
test: setup-envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $(PKGS) -coverprofile cover.out

.PHONY: test-race
test-race:
	go test -race $(PKGS)

# Fast local loop: strict lint + focused go tests + one e2e scenario.
.PHONY: test-quick
test-quick: static-check setup-envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $(PKGS) -run '$(QUICK_GO_TEST_REGEX)' -count=1
	$(MAKE) test-e2e-scenario E2E=$(QUICK_E2E)

.PHONY: e2e-namespace
e2e-namespace:
	@KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL) --context $(KIND_CONTEXT) create namespace $(E2E_NAMESPACE) --dry-run=client -o yaml | KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL) --context $(KIND_CONTEXT) apply -f - >/dev/null

# Deploys the manager with the flags the integration scenarios rely on.
#
# The manager probes the optional integration CRDs once at startup, so the
# rollout is always restarted: installing a CRD after the manager started would
# otherwise leave the integration silently inactive.
.PHONY: e2e-install
e2e-install: manifests kustomize
	$(KUSTOMIZE) build config/e2e | $(KUBECTL) --kubeconfig $(KIND_KUBECONFIG) --context $(KIND_CONTEXT) apply -f -
	$(KUBECTL) --kubeconfig $(KIND_KUBECONFIG) --context $(KIND_CONTEXT) -n kick-system rollout restart deployment/kick-controller-manager
	$(KUBECTL) --kubeconfig $(KIND_KUBECONFIG) --context $(KIND_CONTEXT) -n kick-system rollout status deployment/kick-controller-manager --timeout=180s

# Installs the in-cluster Git server the GitOps scenarios sync from.
.PHONY: e2e-git-server
e2e-git-server:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) bash test/e2e/setup/gitea/install-gitea.sh

# Applies the Argo CD settings the integration scenarios depend on.
.PHONY: e2e-argocd-config
e2e-argocd-config:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) bash test/e2e/setup/argocd/configure-argocd.sh

# Installs the Argo CD control plane and configures it for integration scenarios.
.PHONY: e2e-argocd
e2e-argocd:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) bash test/e2e/setup/argocd/install-argocd.sh

# Installs the Argo Rollouts controller and its CRDs.
.PHONY: e2e-rollouts
e2e-rollouts:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) bash test/e2e/setup/rollouts/install-rollouts.sh

# Installs the Secrets Store CSI driver, OpenBao and the OpenBao CSI provider.
.PHONY: e2e-csi
e2e-csi:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) bash test/e2e/setup/csi/install-csi.sh

# Installs cert-manager and the Kargo control plane.
.PHONY: e2e-kargo
e2e-kargo:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) bash test/e2e/setup/kargo/install-kargo.sh

# Shared prerequisites of every integration suite. Optional integration CRDs are
# installed by the per-suite targets before e2e-install starts the manager.
.PHONY: e2e-base-setup
e2e-base-setup: e2e-namespace e2e-git-server e2e-argocd

# Prerequisites of the suites that only need KICK itself.
.PHONY: e2e-core-setup
e2e-core-setup: e2e-namespace e2e-install

.PHONY: e2e-integration-setup
e2e-integration-setup: e2e-base-setup e2e-install

.PHONY: e2e-rollouts-setup
e2e-rollouts-setup: e2e-base-setup e2e-rollouts e2e-install

.PHONY: e2e-csi-setup
e2e-csi-setup: e2e-base-setup e2e-csi e2e-install

.PHONY: e2e-kargo-setup
e2e-kargo-setup: e2e-base-setup e2e-kargo e2e-install

.PHONY: test-e2e
test-e2e: chainsaw e2e-base-setup e2e-rollouts e2e-csi e2e-kargo e2e-install
	@$(E2E_REPORT_MKDIR) true
	KUBECONFIG=$(KIND_KUBECONFIG) $(CHAINSAW) test --config $(E2E_CHAINSAW_CONFIG_INTEGRATION) --kube-context $(KIND_CONTEXT) $(E2E_REPORT_FLAGS) test/e2e/scenarios

.PHONY: test-e2e-core
test-e2e-core: E2E_REPORT_NAME = core
test-e2e-core: chainsaw e2e-core-setup
	$(call e2e_suite,core,-Ev,$(E2E_IDS_NONCORE),$(E2E_CHAINSAW_CONFIG))

.PHONY: test-e2e-argocd
test-e2e-argocd: E2E_REPORT_NAME = argocd
test-e2e-argocd: chainsaw e2e-integration-setup
	$(call e2e_suite,argocd,-E,$(E2E_IDS_ARGOCD),$(E2E_CHAINSAW_CONFIG_INTEGRATION))

.PHONY: test-e2e-recovery
test-e2e-recovery: E2E_REPORT_NAME = recovery
test-e2e-recovery: chainsaw e2e-core-setup
	$(call e2e_suite,recovery,-E,$(E2E_IDS_RECOVERY),$(E2E_CHAINSAW_CONFIG))

.PHONY: test-e2e-rollouts
test-e2e-rollouts: E2E_REPORT_NAME = rollouts
test-e2e-rollouts: chainsaw e2e-rollouts-setup
	$(call e2e_suite,argo-rollouts,-E,$(E2E_IDS_ROLLOUTS),$(E2E_CHAINSAW_CONFIG_INTEGRATION))

.PHONY: test-e2e-csi
test-e2e-csi: E2E_REPORT_NAME = csi
test-e2e-csi: chainsaw e2e-csi-setup
	$(call e2e_suite,csi,-E,$(E2E_IDS_CSI),$(E2E_CHAINSAW_CONFIG_INTEGRATION))

.PHONY: test-e2e-kargo
test-e2e-kargo: E2E_REPORT_NAME = kargo
test-e2e-kargo: chainsaw e2e-kargo-setup
	$(call e2e_suite,kargo,-E,$(E2E_IDS_KARGO),$(E2E_CHAINSAW_CONFIG_INTEGRATION))

# Runs a single scenario by ID or directory-name fragment. Uses the integration
# timeout budget so it works for both core and integration scenarios; the
# suite prerequisites are not installed, so run the matching setup target first.
.PHONY: test-e2e-scenario
test-e2e-scenario: chainsaw e2e-namespace
	@if [[ -z "$(E2E)" ]]; then echo "E2E is required, e.g. make test-e2e-scenario E2E=032"; exit 1; fi
	@scenario_dir="$$(find test/e2e/scenarios -mindepth 1 -maxdepth 1 -type d | grep -i '/KICK-E2E-$(E2E)\|/$(E2E)' | head -n 1)"; \
	if [[ -z "$$scenario_dir" ]]; then \
		echo "no scenario directory matches E2E=$(E2E)"; \
		exit 1; \
	fi; \
	KUBECONFIG=$(KIND_KUBECONFIG) $(CHAINSAW) test --config $(E2E_CHAINSAW_CONFIG_INTEGRATION) --kube-context $(KIND_CONTEXT) "$$scenario_dir"

.PHONY: test-e2e-render
test-e2e-render: chainsaw
	$(CHAINSAW) test --config test/e2e/chainsaw-configuration.yaml --no-cluster test/e2e/scenarios

.PHONY: kind-create
kind-create:
	$(KIND) create cluster --name $(KIND_CLUSTER_NAME) --kubeconfig $(KIND_KUBECONFIG) --config hack/kind-config.yaml --wait 5m

.PHONY: kind-delete
kind-delete:
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: install
install: manifests kustomize
	$(KUSTOMIZE) build config/default | $(KUBECTL) --kubeconfig $(KIND_KUBECONFIG) --context $(KIND_CONTEXT) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/default | $(KUBECTL) --kubeconfig $(KIND_KUBECONFIG) --context $(KIND_CONTEXT) delete --ignore-not-found -f -

.PHONY: kind-load
kind-load: docker-build
	$(KIND) load docker-image $(IMG) --name $(KIND_CLUSTER_NAME)

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) .

# Build the manager image once and hand it to every e2e job as an artifact,
# instead of rebuilding it in each of them.
.PHONY: image-archive
image-archive: docker-build
	mkdir -p $(dir $(IMAGE_ARCHIVE))
	docker save -o $(IMAGE_ARCHIVE) $(IMG)

.PHONY: kind-load-archive
kind-load-archive:
	$(KIND) load image-archive $(IMAGE_ARCHIVE) --name $(KIND_CLUSTER_NAME)

.PHONY: tilt-up
tilt-up:
	KUBECONFIG=$(KIND_KUBECONFIG) KICK_KUBECONFIG=$(KIND_KUBECONFIG) $(TILT) up --context $(KIND_CONTEXT)

.PHONY: tilt-down
tilt-down:
	KUBECONFIG=$(KIND_KUBECONFIG) KICK_KUBECONFIG=$(KIND_KUBECONFIG) $(TILT) down --context $(KIND_CONTEXT)

# Single entry point for every generation task.
.PHONY: generate
generate: generate-deepcopy manifests api-field-coverage-gen docs-gen

.PHONY: generate-deepcopy
generate-deepcopy: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd paths="./..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/*.yaml charts/kick/crds/

.PHONY: codegen
codegen: generate

.PHONY: helm-lint
helm-lint:
	helm lint charts/kick

.PHONY: helm-template
helm-template:
	helm template kick charts/kick >/dev/null

.PHONY: docs-gen
docs-gen:
	bash hack/gen-docs.sh

.PHONY: docs-casts
docs-casts:
	bash hack/gen-asciinema.sh

.PHONY: docs-gen-check
docs-gen-check: docs-gen
	git diff --exit-code -- llms.txt llms-full.txt docs/static/llms-full.txt

.PHONY: pre-commit-check
pre-commit-check: docs-gen-check

.PHONY: install-hooks
install-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "Installed git hooks from .githooks/"

.PHONY: feature-coverage
feature-coverage:
	python3 tools/gen_api_field_coverage.py --output traceability/api-field-coverage.generated.yaml
	python3 tools/check_feature_coverage.py --report traceability/feature-coverage-report.md

.PHONY: api-field-coverage-gen
api-field-coverage-gen:
	python3 tools/gen_api_field_coverage.py --output traceability/api-field-coverage.generated.yaml

# Every Python helper under tools/, in one gate.
.PHONY: tools-test
tools-test:
	python3 -m unittest tools/check_feature_coverage_test.py tools/e2e_timing_summary_test.py

# Renders the JUnit reports written by the e2e suites as a markdown table.
.PHONY: e2e-timing-summary
e2e-timing-summary:
	@if [[ -z "$(E2E_REPORT_DIR)" ]]; then echo "E2E_REPORT_DIR is required"; exit 1; fi
	@python3 tools/e2e_timing_summary.py $(E2E_REPORT_DIR)

.PHONY: tools
tools: kustomize controller-gen setup-envtest golangci-lint chainsaw

# Lets CI read a pinned version without duplicating it, e.g. `make print-KIND_VERSION`.
print-%:
	@echo "$($*)"

.PHONY: kamera
kamera: $(KAMERA)

$(KAMERA): $(LOCALBIN)
	@echo "Downloading github.com/tgoodwin/kamera/cmd/kamera@$(KAMERA_VERSION)"
	GOBIN=$(LOCALBIN) GOTOOLCHAIN=$(GO_TOOLCHAIN) go install github.com/tgoodwin/kamera/cmd/kamera@$(KAMERA_VERSION)

.PHONY: verify
verify: fmt vet lint static-check shellcheck test helm-lint helm-template docs-gen-check feature-coverage

# Mirrors the CI jobs. `vet` is not repeated here: golangci-lint runs govet as
# part of `make lint`, so a separate pass only pays the compile cost twice.
.PHONY: ci-verify-local
ci-verify-local: tools
	$(MAKE) fmt
	$(MAKE) lint
	$(MAKE) static-check
	$(MAKE) test
	$(MAKE) test-race
	$(MAKE) generate
	git diff --exit-code
	$(MAKE) helm-lint
	$(MAKE) helm-template
	$(MAKE) docs-gen-check
	$(MAKE) tools-test
	$(MAKE) vulncheck
	$(MAKE) feature-coverage

.PHONY: vulncheck
vulncheck: govulncheck
	$(GOVULNCHECK) ./...

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK)
$(GOVULNCHECK): $(LOCALBIN)
	$(call go-install-tool,$(GOVULNCHECK),golang.org/x/vuln/cmd/govulncheck,$(GOVULNCHECK_VERSION))

.PHONY: ci-e2e-local
ci-e2e-local:
	@set -e; \
	cleanup() { $(MAKE) kind-delete KIND=kind; }; \
	trap cleanup EXIT; \
	$(MAKE) kind-create KIND=kind; \
	$(MAKE) kind-load KIND=kind; \
	$(MAKE) test-e2e KIND=kind

.PHONY: ci-e2e-core-local
ci-e2e-core-local: ci-e2e-local

.PHONY: ci-local
ci-local: ci-verify-local ci-e2e-local

.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: $(ENVTEST)
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: chainsaw
chainsaw: $(CHAINSAW)
$(CHAINSAW): $(LOCALBIN)
	$(call go-install-tool,$(CHAINSAW),github.com/kyverno/chainsaw,$(CHAINSAW_VERSION))

define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3); \
echo "Downloading $$package"; \
rm -f $(1) || true; \
GOBIN=$(LOCALBIN) GOTOOLCHAIN=$(GO_TOOLCHAIN) go install $$package; \
mv $(1) $(1)-$(3); \
}; \
ln -sf $(1)-$(3) $(1)
endef
