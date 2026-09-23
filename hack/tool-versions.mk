# Pinned versions of every downloaded tool.
#
# Kept in its own file so CI can key its tool cache on this file alone. Keying
# on the Makefile invalidated every cached binary on unrelated Makefile edits,
# which cost a full tool rebuild (~3 minutes) per run.
KUSTOMIZE_VERSION ?= v5.8.1
CONTROLLER_TOOLS_VERSION ?= v0.22.0
ENVTEST_VERSION ?= release-0.25
ENVTEST_K8S_VERSION ?= 1.37
GOLANGCI_LINT_VERSION ?= v2.13.2
CHAINSAW_VERSION ?= v0.2.15
KAMERA_VERSION ?= main
GOVULNCHECK_VERSION ?= v1.8.0
KIND_VERSION ?= v0.33.0
