# Pinned versions of every downloaded tool.
#
# Kept in its own file so CI can key its tool cache on this file alone. Keying
# on the Makefile invalidated every cached binary on unrelated Makefile edits,
# which cost a full tool rebuild (~3 minutes) per run.
KUSTOMIZE_VERSION ?= v5.6.0
CONTROLLER_TOOLS_VERSION ?= v0.17.2
ENVTEST_VERSION ?= release-0.24
ENVTEST_K8S_VERSION ?= 1.36
GOLANGCI_LINT_VERSION ?= v2.12.2
CHAINSAW_VERSION ?= v0.2.15
KAMERA_VERSION ?= main
GOVULNCHECK_VERSION ?= v1.1.4
KIND_VERSION ?= v0.24.0
