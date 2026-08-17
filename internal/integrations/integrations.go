// Package integrations names KICK's optional integrations and the two switches
// that control each one, so every message about a disabled or unusable
// integration can point at the exact flag and Helm value that fix it.
package integrations

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Condition reasons reported when an integration cannot serve a request.
const (
	ReasonDisabled         = "IntegrationDisabled"
	ReasonKindNotInstalled = "IntegrationKindNotInstalled"
)

// Integration is an optional capability that must be switched on in the
// operator and granted in RBAC. Both happen through the same Helm value.
type Integration struct {
	// Name matches the GitOps provider name for integrations that register one.
	Name      string
	Title     string
	Flag      string
	HelmValue string
}

var (
	ArgoCD = Integration{Name: "argocd", Title: "Argo CD", Flag: "--enable-argocd", HelmValue: "integrations.argocd.enabled"}
	Flux   = Integration{Name: "flux", Title: "Flux", Flag: "--enable-flux", HelmValue: "integrations.flux.enabled"}
	Kargo  = Integration{Name: "kargo", Title: "Kargo", Flag: "--enable-kargo", HelmValue: "integrations.kargo.enabled"}

	ArgoRollouts    = Integration{Title: "Argo Rollouts", Flag: "--enable-argo-rollouts", HelmValue: "integrations.argoRollouts.enabled"}
	SecretsStoreCSI = Integration{Title: "Secrets Store CSI", Flag: "--enable-csi-integration", HelmValue: "integrations.secretsStoreCSI.enabled"}
)

// DisabledMessage states that the operator was started without the integration
// and names both switches that turn it on.
func (i Integration) DisabledMessage() string {
	return fmt.Sprintf("the %s integration is disabled; enable it with %s=true (Helm: --set %s=true)", i.Title, i.Flag, i.HelmValue)
}

// KindNotInstalledMessage states that the integration is switched on but the
// cluster does not serve its API. The kind is probed once at start-up.
func (i Integration) KindNotInstalledMessage(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf("the %s integration is enabled (%s, Helm: %s) but the cluster does not serve %s; install its CRDs and restart the KICK manager",
		i.Title, i.Flag, i.HelmValue, gvk)
}
