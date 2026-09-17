package kube

import (
	"strings"
)

// NamespaceFilter filters namespaces in the application layer.
type NamespaceFilter struct {
	NameContains  *string
	HasDeployment *DeploymentFilter
	HasPod        *PodFilter
}

// DeploymentFilter filters deployments in the application layer.
type DeploymentFilter struct {
	NameContains  *string
	LabelSelector *string
	HasPod        *PodFilter
}

// CronJobFilter filters cron jobs in the application layer.
type CronJobFilter struct {
	NameContains  *string
	LabelSelector *string
	Suspended     *bool
}

// PodFilter filters pods in the application layer.
type PodFilter struct {
	NameContains  *string
	LabelSelector *string
	Phase         *string
	Ready         *bool
}

// MatchNamespaceName returns true when ns matches the optional nameContains clause.
func MatchNamespaceName(ns Namespace, f *NamespaceFilter) bool {
	if f == nil || f.NameContains == nil {
		return true
	}
	return containsFold(ns.Name, *f.NameContains)
}

// MatchDeployment returns true when d matches name clauses.
// LabelSelector is expected to be applied at list time.
func MatchDeployment(d Deployment, f *DeploymentFilter) bool {
	if f == nil {
		return true
	}
	if f.NameContains != nil && !containsFold(d.Name, *f.NameContains) {
		return false
	}
	return true
}

// MatchCronJob returns true when cj matches name/suspend clauses.
// LabelSelector is expected to be applied at list time.
func MatchCronJob(cj CronJob, f *CronJobFilter) bool {
	if f == nil {
		return true
	}
	if f.NameContains != nil && !containsFold(cj.Name, *f.NameContains) {
		return false
	}
	if f.Suspended != nil && cj.Suspend != *f.Suspended {
		return false
	}
	return true
}

// MatchPod returns true when p matches the filter (excluding labelSelector).
func MatchPod(p Pod, f *PodFilter) bool {
	if f == nil {
		return true
	}
	if f.NameContains != nil && !containsFold(p.Name, *f.NameContains) {
		return false
	}
	if f.Phase != nil && !strings.EqualFold(p.Phase, *f.Phase) {
		return false
	}
	if f.Ready != nil && p.Ready != *f.Ready {
		return false
	}
	return true
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// CustomResourceFilter filters discovered CRDs and their instances.
type CustomResourceFilter struct {
	NameContains *string
	KindContains *string
	Group        *string
	Version      *string
	Kind         *string
	Resource     *string
}

// MatchAPIResource returns true when the discovered API matches group/kind clauses.
func MatchAPIResource(a APIResource, f *CustomResourceFilter) bool {
	if f == nil {
		return true
	}
	if f.Group != nil && a.Group != *f.Group {
		return false
	}
	if f.Version != nil && a.Version != *f.Version {
		return false
	}
	if f.Kind != nil && !strings.EqualFold(a.Kind, *f.Kind) {
		return false
	}
	if f.Resource != nil && a.Resource != *f.Resource {
		return false
	}
	if f.KindContains != nil {
		n := *f.KindContains
		if !containsFold(a.Kind, n) && !containsFold(a.Resource, n) && !containsFold(a.Group, n) {
			return false
		}
	}
	return true
}

// MatchCustomResource returns true when the instance matches name/kind clauses.
func MatchCustomResource(cr CustomResource, f *CustomResourceFilter) bool {
	if f == nil {
		return true
	}
	if f.NameContains != nil && !containsFold(cr.Name, *f.NameContains) {
		return false
	}
	if f.Group != nil && cr.Group != *f.Group {
		return false
	}
	if f.Version != nil && cr.Version != *f.Version {
		return false
	}
	if f.Kind != nil && !strings.EqualFold(cr.Kind, *f.Kind) {
		return false
	}
	if f.Resource != nil && cr.Resource != *f.Resource {
		return false
	}
	if f.KindContains != nil {
		n := *f.KindContains
		if !containsFold(cr.Kind, n) && !containsFold(cr.Resource, n) && !containsFold(cr.Group, n) {
			return false
		}
	}
	return true
}

// LabelSelector returns the selector string or empty.
func (f *DeploymentFilter) LabelSelectorString() string {
	if f == nil || f.LabelSelector == nil {
		return ""
	}
	return *f.LabelSelector
}

// LabelSelectorString returns the selector string or empty.
func (f *CronJobFilter) LabelSelectorString() string {
	if f == nil || f.LabelSelector == nil {
		return ""
	}
	return *f.LabelSelector
}

// LabelSelectorString returns the selector string or empty.
func (f *PodFilter) LabelSelectorString() string {
	if f == nil || f.LabelSelector == nil {
		return ""
	}
	return *f.LabelSelector
}

// SelectorToString converts a matchLabels map to a label selector string.
func SelectorToString(sel map[string]string) string {
	if len(sel) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sel))
	for k, v := range sel {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}
