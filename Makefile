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
KIND ?= kind
TILT ?= tilt

KUSTOMIZE_VERSION ?= v5.6.0
CONTROLLER_TOOLS_VERSION ?= v0.17.2
ENVTEST_VERSION ?= release-0.24
ENVTEST_K8S_VERSION ?= 1.36
GOLANGCI_LINT_VERSION ?= v2.12.2
CHAINSAW_VERSION ?= v0.2.15
KIND_CLUSTER_NAME ?= kick-dev
KIND_CONTEXT ?= kind-$(KIND_CLUSTER_NAME)
KIND_KUBECONFIG ?= $(shell pwd)/.kubeconfig-kind-kick-dev
PKGS := $(shell go list ./... | grep -v '/ai-docs/' || true)

.PHONY: fmt
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*' -not -path './ai-docs/*')

.PHONY: vet
vet:
	go vet $(PKGS)

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: test
test: setup-envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $(PKGS) -coverprofile cover.out

.PHONY: test-e2e
test-e2e: chainsaw
	$(CHAINSAW) test --config test/e2e/chainsaw-configuration.yaml --kubeconfig $(KIND_KUBECONFIG) --kube-context $(KIND_CONTEXT) test/e2e/scenarios

.PHONY: test-e2e-core
test-e2e-core: chainsaw
	$(CHAINSAW) test --config test/e2e/chainsaw-configuration.yaml --kubeconfig $(KIND_KUBECONFIG) --kube-context $(KIND_CONTEXT) --exclude-test-regex "KICK-E2E-(024|025|026|027|028|029|030|031|032|033|034|035|036|037|038|039|040|041|042|048|049|050|051)" test/e2e/scenarios

.PHONY: test-e2e-argocd
test-e2e-argocd: chainsaw
	$(CHAINSAW) test --config test/e2e/chainsaw-configuration.yaml --kubeconfig $(KIND_KUBECONFIG) --kube-context $(KIND_CONTEXT) --include-test-regex "KICK-E2E-(024|025|026|027|028|029|030|031|032|033|034|035|036|037|038|039|040|041|042)" test/e2e/scenarios

.PHONY: test-e2e-recovery
test-e2e-recovery: chainsaw
	$(CHAINSAW) test --config test/e2e/chainsaw-configuration.yaml --kubeconfig $(KIND_KUBECONFIG) --kube-context $(KIND_CONTEXT) --include-test-regex "KICK-E2E-(048|049|050|051)" test/e2e/scenarios

.PHONY: test-e2e-scenario
test-e2e-scenario: chainsaw
	@if [[ -z "$(E2E)" ]]; then echo "E2E is required, e.g. make test-e2e-scenario E2E=KICK-E2E-032"; exit 1; fi
	$(CHAINSAW) test --config test/e2e/chainsaw-configuration.yaml --kubeconfig $(KIND_KUBECONFIG) --kube-context $(KIND_CONTEXT) --include-test-regex "$(E2E)" test/e2e/scenarios

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
	KUBECONFIG=$(KIND_KUBECONFIG) $(TILT) up --k8s-context $(KIND_CONTEXT)

.PHONY: tilt-down
tilt-down:
	KUBECONFIG=$(KIND_KUBECONFIG) $(TILT) down --k8s-context $(KIND_CONTEXT)

.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: codegen
codegen: generate manifests

.PHONY: helm-lint
helm-lint:
	helm lint charts/kick

.PHONY: helm-template
helm-template:
	helm template kick charts/kick >/dev/null

.PHONY: docs-gen
docs-gen:
	bash hack/gen-docs.sh

.PHONY: docs-gen-check
docs-gen-check: docs-gen
	git diff --exit-code -- llms.txt llms-full.txt

.PHONY: feature-coverage
feature-coverage:
	python3 tools/check_feature_coverage.py --report traceability/feature-coverage-report.md

.PHONY: feature-coverage-test
feature-coverage-test:
	python3 -m unittest tools/check_feature_coverage_test.py

.PHONY: tools
tools: kustomize controller-gen setup-envtest golangci-lint chainsaw

.PHONY: verify
verify: fmt vet lint test helm-lint helm-template docs-gen-check feature-coverage

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
