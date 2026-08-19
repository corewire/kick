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
CHAINSAW ?= $(LOCALBIN)/chainsaw
KAMERA ?= $(LOCALBIN)/kamera
KIND ?= kind
TILT ?= tilt

KUSTOMIZE_VERSION ?= v5.6.0
CONTROLLER_TOOLS_VERSION ?= v0.17.2
ENVTEST_VERSION ?= release-0.24
ENVTEST_K8S_VERSION ?= 1.36
GOLANGCI_LINT_VERSION ?= v2.12.2
CHAINSAW_VERSION ?= v0.2.15
KAMERA_VERSION ?= main
KIND_CLUSTER_NAME ?= kick-dev
KIND_CONTEXT ?= kind-$(KIND_CLUSTER_NAME)
KIND_KUBECONFIG ?= $(shell pwd)/.kubeconfig-kind-kick-dev
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
KUBECONFIG=$(KIND_KUBECONFIG) $(CHAINSAW) test --config $(4) --kube-context $(KIND_CONTEXT) $$scenario_dirs
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
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) test/e2e/setup/gitea/install-gitea.sh

# Applies the Argo CD settings the integration scenarios depend on.
.PHONY: e2e-argocd-config
e2e-argocd-config:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) test/e2e/setup/argocd/configure-argocd.sh

# Installs the Argo Rollouts controller and its CRDs.
.PHONY: e2e-rollouts
e2e-rollouts:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) test/e2e/setup/rollouts/install-rollouts.sh

# Installs the Secrets Store CSI driver, OpenBao and the OpenBao CSI provider.
.PHONY: e2e-csi
e2e-csi:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) test/e2e/setup/csi/install-csi.sh

# Installs cert-manager and the Kargo control plane.
.PHONY: e2e-kargo
e2e-kargo:
	KICK_E2E_KUBECONFIG=$(KIND_KUBECONFIG) test/e2e/setup/kargo/install-kargo.sh

# Shared prerequisites of every integration suite. Optional integration CRDs are
# installed by the per-suite targets before e2e-install starts the manager.
.PHONY: e2e-base-setup
e2e-base-setup: e2e-namespace e2e-git-server e2e-argocd-config

.PHONY: e2e-integration-setup
e2e-integration-setup: e2e-base-setup e2e-install

.PHONY: e2e-rollouts-setup
e2e-rollouts-setup: e2e-base-setup e2e-rollouts e2e-install

.PHONY: e2e-csi-setup
e2e-csi-setup: e2e-base-setup e2e-csi e2e-install

.PHONY: e2e-kargo-setup
e2e-kargo-setup: e2e-base-setup e2e-kargo e2e-install

.PHONY: test-e2e
test-e2e: chainsaw e2e-namespace
	KUBECONFIG=$(KIND_KUBECONFIG) $(CHAINSAW) test --config test/e2e/chainsaw-configuration.yaml --kube-context $(KIND_CONTEXT) test/e2e/scenarios

.PHONY: test-e2e-core
test-e2e-core: chainsaw e2e-namespace
	$(call e2e_suite,core,-Ev,$(E2E_IDS_NONCORE),$(E2E_CHAINSAW_CONFIG))

.PHONY: test-e2e-argocd
test-e2e-argocd: chainsaw e2e-integration-setup
	$(call e2e_suite,argocd,-E,$(E2E_IDS_ARGOCD),$(E2E_CHAINSAW_CONFIG_INTEGRATION))

.PHONY: test-e2e-recovery
test-e2e-recovery: chainsaw e2e-namespace
	$(call e2e_suite,recovery,-E,$(E2E_IDS_RECOVERY),$(E2E_CHAINSAW_CONFIG))

.PHONY: test-e2e-rollouts
test-e2e-rollouts: chainsaw e2e-rollouts-setup
	$(call e2e_suite,argo-rollouts,-E,$(E2E_IDS_ROLLOUTS),$(E2E_CHAINSAW_CONFIG_INTEGRATION))

.PHONY: test-e2e-csi
test-e2e-csi: chainsaw e2e-csi-setup
	$(call e2e_suite,csi,-E,$(E2E_IDS_CSI),$(E2E_CHAINSAW_CONFIG_INTEGRATION))

.PHONY: test-e2e-kargo
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
kind-load:
	docker build -t $(IMG) .
	$(KIND) load docker-image $(IMG) --name $(KIND_CLUSTER_NAME)

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

.PHONY: feature-coverage
feature-coverage:
	python3 tools/gen_api_field_coverage.py --output traceability/api-field-coverage.generated.yaml
	python3 tools/check_feature_coverage.py --report traceability/feature-coverage-report.md

.PHONY: api-field-coverage-gen
api-field-coverage-gen:
	python3 tools/gen_api_field_coverage.py --output traceability/api-field-coverage.generated.yaml

.PHONY: feature-coverage-test
feature-coverage-test:
	python3 -m unittest tools/check_feature_coverage_test.py

.PHONY: tools
tools: kustomize controller-gen setup-envtest golangci-lint chainsaw

.PHONY: kamera
kamera: $(KAMERA)

$(KAMERA): $(LOCALBIN)
	@echo "Downloading github.com/tgoodwin/kamera/cmd/kamera@$(KAMERA_VERSION)"
	GOBIN=$(LOCALBIN) GOTOOLCHAIN=local go install github.com/tgoodwin/kamera/cmd/kamera@$(KAMERA_VERSION)

.PHONY: verify
verify: fmt vet lint static-check shellcheck test helm-lint helm-template docs-gen-check feature-coverage

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
GOBIN=$(LOCALBIN) GOTOOLCHAIN=local go install $$package; \
mv $(1) $(1)-$(3); \
}; \
ln -sf $(1)-$(3) $(1)
endef
