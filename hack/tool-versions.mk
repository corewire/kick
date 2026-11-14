# Pinned versions of every downloaded tool.
#
# Kept in its own file so CI can key its tool cache on this file and Go. Keying
# on the Makefile invalidated every cached binary on unrelated Makefile edits,
# which cost a full tool rebuild (~3 minutes) per run.
KUSTOMIZE_VERSION ?= v5.8.3
CONTROLLER_TOOLS_VERSION ?= v0.23.0
ENVTEST_VERSION ?= v0.25.2
ENVTEST_K8S_VERSION ?= 1.37.0
GOLANGCI_LINT_VERSION ?= v2.15.0
CHAINSAW_VERSION ?= v0.2.15
KAMERA_VERSION ?= v0.0.18
GOVULNCHECK_VERSION ?= v1.9.0
KIND_VERSION ?= v0.33.0
HELM_VERSION ?= v4.3.2
ACTIONLINT_VERSION ?= v1.7.12
