package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cdreier/kubeql/graph/model"
	"github.com/cdreier/kubeql/internal/kube"
)

func toNamespace(kubeContext string, ns kube.Namespace) *model.Namespace {
	return &model.Namespace{Context: kubeContext, Name: ns.Name}
}

func toDeployment(kubeContext string, d kube.Deployment) *model.Deployment {
	return &model.Deployment{
		Context:           kubeContext,
		Name:              d.Name,
		Namespace:         d.Namespace,
		Replicas:          int(d.Replicas),
		ReadyReplicas:     int(d.ReadyReplicas),
		AvailableReplicas: int(d.AvailableReplicas),
		Status:            d.Status,
		Labels:            toLabels(d.Labels),
	}
}

func toPod(kubeContext string, p kube.Pod) *model.Pod {
	var nodeName *string
	if p.NodeName != "" {
		n := p.NodeName
		nodeName = &n
	}
	containers := make([]*model.Container, 0, len(p.Containers))
	for _, c := range p.Containers {
		containers = append(containers, &model.Container{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        c.Ready,
			RestartCount: int(c.RestartCount),
			State:        c.State,
		})
	}
	return &model.Pod{
		Context:       kubeContext,
		Name:          p.Name,
		Namespace:     p.Namespace,
		Phase:         p.Phase,
		Ready:         p.Ready,
		Restarts:      int(p.Restarts),
		LastRestartAt: p.LastRestartAt,
		CreatedAt:     p.CreatedAt,
		NodeName:      nodeName,
		Labels:        toLabels(p.Labels),
		Containers:    containers,
		CPUUsage:      p.CPUUsage,
		CPULimit:      p.CPULimit,
		MemoryUsage:   p.MemoryUsage,
		MemoryLimit:   p.MemoryLimit,
	}
}

func toKeyValues(m map[string]string) []*model.KeyValue {
	if len(m) == 0 {
		return []*model.KeyValue{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*model.KeyValue, 0, len(keys))
	for _, k := range keys {
		out = append(out, &model.KeyValue{Key: k, Value: m[k]})
	}
	return out
}

func toConfigMap(kubeContext string, cm kube.ConfigMap) *model.ConfigMap {
	keys := make([]string, 0, len(cm.Data))
	for k := range cm.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	refs := cm.Refs
	if refs == nil {
		refs = []string{}
	}
	return &model.ConfigMap{
		Context:   kubeContext,
		Name:      cm.Name,
		Namespace: cm.Namespace,
		Refs:      refs,
		Missing:   cm.Missing,
		Keys:      keys,
		Data:      toKeyValues(cm.Data),
	}
}

func toSecret(kubeContext string, s kube.Secret) *model.Secret {
	keys := make([]string, 0, len(s.Data))
	for k := range s.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	refs := s.Refs
	if refs == nil {
		refs = []string{}
	}
	return &model.Secret{
		Context:   kubeContext,
		Name:      s.Name,
		Namespace: s.Namespace,
		Type:      s.Type,
		Refs:      refs,
		Missing:   s.Missing,
		Keys:      keys,
		Data:      toKeyValues(s.Data),
	}
}

func toConfigMaps(kubeContext string, list []kube.ConfigMap) []*model.ConfigMap {
	out := make([]*model.ConfigMap, 0, len(list))
	for _, cm := range list {
		out = append(out, toConfigMap(kubeContext, cm))
	}
	return out
}

func toSecrets(kubeContext string, list []kube.Secret) []*model.Secret {
	out := make([]*model.Secret, 0, len(list))
	for _, s := range list {
		out = append(out, toSecret(kubeContext, s))
	}
	return out
}

func toLabels(m map[string]string) []*model.Label {
	if len(m) == 0 {
		return []*model.Label{}
	}
	out := make([]*model.Label, 0, len(m))
	for k, v := range m {
		out = append(out, &model.Label{Key: k, Value: v})
	}
	return out
}

func toNamespaces(kubeContext string, list []kube.Namespace) []*model.Namespace {
	out := make([]*model.Namespace, 0, len(list))
	for _, ns := range list {
		out = append(out, toNamespace(kubeContext, ns))
	}
	return out
}

func toKubeContext(c kube.ContextInfo) *model.KubeContext {
	var ns *string
	if c.Namespace != "" {
		n := c.Namespace
		ns = &n
	}
	return &model.KubeContext{
		Name:             c.Name,
		Cluster:          c.Cluster,
		User:             c.User,
		DefaultNamespace: ns,
		Current:          c.Current,
	}
}

func toKubeContexts(list []kube.ContextInfo) []*model.KubeContext {
	out := make([]*model.KubeContext, 0, len(list))
	for _, c := range list {
		out = append(out, toKubeContext(c))
	}
	return out
}

func toDeployments(kubeContext string, list []kube.Deployment) []*model.Deployment {
	out := make([]*model.Deployment, 0, len(list))
	for _, d := range list {
		out = append(out, toDeployment(kubeContext, d))
	}
	return out
}

func toPods(kubeContext string, list []kube.Pod) []*model.Pod {
	out := make([]*model.Pod, 0, len(list))
	for _, p := range list {
		out = append(out, toPod(kubeContext, p))
	}
	return out
}

func mapNamespaceFilter(f *model.NamespaceFilter) *kube.NamespaceFilter {
	if f == nil {
		return nil
	}
	return &kube.NamespaceFilter{
		NameContains:  f.NameContains,
		HasDeployment: mapDeploymentFilter(f.HasDeployment),
		HasPod:        mapPodFilter(f.HasPod),
	}
}

func mapDeploymentFilter(f *model.DeploymentFilter) *kube.DeploymentFilter {
	if f == nil {
		return nil
	}
	return &kube.DeploymentFilter{
		NameContains:  f.NameContains,
		LabelSelector: f.LabelSelector,
		HasPod:        mapPodFilter(f.HasPod),
	}
}

func mapPodFilter(f *model.PodFilter) *kube.PodFilter {
	if f == nil {
		return nil
	}
	return &kube.PodFilter{
		NameContains:  f.NameContains,
		LabelSelector: f.LabelSelector,
		Phase:         f.Phase,
		Ready:         f.Ready,
	}
}

func nsOrEmpty(ns *string) string {
	if ns == nil {
		return ""
	}
	return *ns
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func deploymentLiveKey(d kube.Deployment, pods []kube.Pod) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d|%d|%d|%s", d.Replicas, d.ReadyReplicas, d.AvailableReplicas, d.Status)
	for _, p := range pods {
		fmt.Fprintf(&b, ";%s|%s|%v|%d|%s|%s",
			p.Name, p.Phase, p.Ready, p.Restarts, derefStr(p.CPUUsage), derefStr(p.MemoryUsage))
	}
	return b.String()
}
