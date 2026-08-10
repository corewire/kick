package policy

import (
	"context"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ReasonNoMatchingPolicy  = "NoMatchingPolicy"
	ReasonPolicyDeleted     = "PolicyDeleted"
	ReasonPolicyNoLonger    = "PolicyNoLongerMatches"
	ReasonConflictingPolicy = "ConflictingPolicies"
)

// MatchResult captures policy selection outcome for one workload.
type MatchResult struct {
	Managed bool
	Reason  string
	Policy  *kickv1alpha1.KickPolicy
	Matches []kickv1alpha1.KickPolicy
}

// WorkloadMatcher determines whether a workload is managed by one policy.
type WorkloadMatcher interface {
	MatchWorkload(ctx context.Context, namespace string, labels map[string]string) (MatchResult, error)
}

// DeploymentPolicyMatcher selects KickPolicy resources in deployment namespace.
type DeploymentPolicyMatcher struct {
	Client client.Client
}

func (m *DeploymentPolicyMatcher) MatchDeployment(ctx context.Context, deployment *appsv1.Deployment) (MatchResult, error) {
	if deployment == nil {
		return MatchResult{Managed: false, Reason: ReasonNoMatchingPolicy}, nil
	}
	return m.MatchWorkload(ctx, deployment.Namespace, deployment.Labels)
}

func (m *DeploymentPolicyMatcher) MatchWorkload(ctx context.Context, namespace string, workloadLabels map[string]string) (MatchResult, error) {
	var list kickv1alpha1.KickPolicyList
	if err := m.Client.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return MatchResult{}, err
	}

	matches := make([]kickv1alpha1.KickPolicy, 0, len(list.Items))
	for _, pol := range list.Items {
		if pol.Spec.Suspend {
			continue
		}
		selector, err := selectorForPolicy(&pol)
		if err != nil {
			continue
		}
		if selector.Matches(labels.Set(workloadLabels)) {
			matches = append(matches, pol)
		}
	}

	switch len(matches) {
	case 0:
		reason := ReasonNoMatchingPolicy
		if len(list.Items) == 0 {
			reason = ReasonPolicyDeleted
		}
		return MatchResult{Managed: false, Reason: reason, Matches: matches}, nil
	case 1:
		p := matches[0]
		return MatchResult{Managed: true, Policy: &p, Matches: matches}, nil
	default:
		return MatchResult{Managed: false, Reason: ReasonConflictingPolicy, Matches: matches}, nil
	}
}

func selectorForPolicy(policy *kickv1alpha1.KickPolicy) (labels.Selector, error) {
	if policy == nil || policy.Spec.Discovery.WorkloadSelector == nil {
		return labels.Everything(), nil
	}
	return metav1.LabelSelectorAsSelector(policy.Spec.Discovery.WorkloadSelector)
}
