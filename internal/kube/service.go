package kube

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Service applies filters and composes ClusterReader calls for the GraphQL layer.
// Registry backs multi-context access (one clientset per kubeconfig context).
// Reader is used only in unit tests with a Fake (no Registry).
type Service struct {
	Reader   ClusterReader
	Registry *Registry
}

func NewService(r ClusterReader) *Service {
	return &Service{Reader: r}
}

// NewServiceFromRegistry builds a Service that resolves cluster clients per context name.
// No client is created at startup; clients are opened lazily on first use.
func NewServiceFromRegistry(reg *Registry) (*Service, error) {
	return &Service{Registry: reg}, nil
}

// ListContexts returns kubeconfig contexts (empty when no Registry is attached).
func (s *Service) ListContexts() []ContextInfo {
	if s.Registry == nil {
		return []ContextInfo{}
	}
	return s.Registry.Contexts()
}

// readerFor returns a ClusterReader for the given kubeconfig context name.
// With a Registry, kubeContext is required. Without a Registry (unit tests),
// the injected Reader is used and kubeContext is ignored.
func (s *Service) readerFor(kubeContext string) (ClusterReader, error) {
	if s.Registry != nil {
		if kubeContext == "" {
			return nil, fmt.Errorf("kube context is required")
		}
		return s.Registry.Client(kubeContext)
	}
	if s.Reader != nil {
		return s.Reader, nil
	}
	return nil, fmt.Errorf("no kubernetes cluster client configured")
}

// ListNamespaces returns namespaces matching the filter (including nested resource filters).
func (s *Service) ListNamespaces(ctx context.Context, kubeContext string, f *NamespaceFilter) ([]Namespace, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	all, err := r.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return all, nil
	}

	out := make([]Namespace, 0, len(all))
	for _, ns := range all {
		if !MatchNamespaceName(ns, f) {
			continue
		}
		if f.HasDeployment != nil {
			deps, err := s.ListDeployments(ctx, kubeContext, ns.Name, f.HasDeployment)
			if err != nil {
				return nil, err
			}
			if len(deps) == 0 {
				continue
			}
		}
		if f.HasPod != nil {
			pods, err := s.ListPods(ctx, kubeContext, ns.Name, f.HasPod)
			if err != nil {
				return nil, err
			}
			if len(pods) == 0 {
				continue
			}
		}
		out = append(out, ns)
	}
	return out, nil
}

func (s *Service) GetNamespace(ctx context.Context, kubeContext, name string) (*Namespace, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	return r.GetNamespace(ctx, name)
}

// ListDeployments lists deployments in namespace (empty = all) with optional filter.
func (s *Service) ListDeployments(ctx context.Context, kubeContext, namespace string, f *DeploymentFilter) ([]Deployment, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	deps, err := r.ListDeployments(ctx, namespace, f.LabelSelectorString())
	if err != nil {
		return nil, err
	}
	out := make([]Deployment, 0, len(deps))
	for _, d := range deps {
		if !MatchDeployment(d, f) {
			continue
		}
		if f != nil && f.HasPod != nil {
			pods, err := s.PodsForDeployment(ctx, kubeContext, d, f.HasPod)
			if err != nil {
				return nil, err
			}
			if len(pods) == 0 {
				continue
			}
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *Service) GetDeployment(ctx context.Context, kubeContext, namespace, name string) (*Deployment, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	return r.GetDeployment(ctx, namespace, name)
}

// ListPods lists pods in namespace (empty = all) with optional filter.
func (s *Service) ListPods(ctx context.Context, kubeContext, namespace string, f *PodFilter) ([]Pod, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	pods, err := r.ListPods(ctx, namespace, f.LabelSelectorString())
	if err != nil {
		return nil, err
	}
	out := make([]Pod, 0, len(pods))
	for _, p := range pods {
		if MatchPod(p, f) {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Service) GetPod(ctx context.Context, kubeContext, namespace, name string) (*Pod, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	return r.GetPod(ctx, namespace, name)
}

// PodsForDeployment returns pods owned by the deployment (via selector), optionally filtered.
func (s *Service) PodsForDeployment(ctx context.Context, kubeContext string, d Deployment, f *PodFilter) ([]Pod, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	selector := SelectorToString(d.Selector)
	pods, err := r.ListPods(ctx, d.Namespace, selector)
	if err != nil {
		return nil, err
	}
	out := make([]Pod, 0, len(pods))
	extra := f.LabelSelectorString()
	for _, p := range pods {
		if extra != "" && !labelsMatchSelector(p.Labels, extra) {
			continue
		}
		// Apply remaining filter fields without re-applying labelSelector at list level.
		pf := f
		if f != nil {
			cp := *f
			cp.LabelSelector = nil
			pf = &cp
		}
		if MatchPod(p, pf) {
			out = append(out, p)
		}
	}
	return out, nil
}

// DeploymentRestarts sums restart counts of pods belonging to the deployment.
func (s *Service) DeploymentRestarts(ctx context.Context, kubeContext string, d Deployment) (int32, error) {
	pods, err := s.PodsForDeployment(ctx, kubeContext, d, nil)
	if err != nil {
		return 0, err
	}
	var total int32
	for _, p := range pods {
		total += p.Restarts
	}
	return total, nil
}

func (s *Service) ListConfigMaps(ctx context.Context, kubeContext, namespace string) ([]ConfigMap, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	return r.ListConfigMaps(ctx, namespace)
}

func (s *Service) GetConfigMap(ctx context.Context, kubeContext, namespace, name string) (*ConfigMap, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	return r.GetConfigMap(ctx, namespace, name)
}

func (s *Service) ListSecrets(ctx context.Context, kubeContext, namespace string) ([]Secret, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	return r.ListSecrets(ctx, namespace)
}

func (s *Service) GetSecret(ctx context.Context, kubeContext, namespace, name string) (*Secret, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	return r.GetSecret(ctx, namespace, name)
}

func (s *Service) ConfigMapsForDeployment(ctx context.Context, kubeContext string, d Deployment) ([]ConfigMap, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	out := make([]ConfigMap, 0, len(d.ConfigMapRefs))
	for _, ref := range d.ConfigMapRefs {
		cm, err := r.GetConfigMap(ctx, d.Namespace, ref.Name)
		if err != nil {
			return nil, err
		}
		if cm == nil {
			out = append(out, ConfigMap{
				Name:      ref.Name,
				Namespace: d.Namespace,
				Refs:      ref.Via,
				Missing:   true,
			})
			continue
		}
		cm.Refs = ref.Via
		out = append(out, *cm)
	}
	return out, nil
}

func (s *Service) SecretsForDeployment(ctx context.Context, kubeContext string, d Deployment) ([]Secret, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	out := make([]Secret, 0, len(d.SecretRefs))
	for _, ref := range d.SecretRefs {
		sec, err := r.GetSecret(ctx, d.Namespace, ref.Name)
		if err != nil {
			return nil, err
		}
		if sec == nil {
			out = append(out, Secret{
				Name:      ref.Name,
				Namespace: d.Namespace,
				Refs:      ref.Via,
				Missing:   true,
			})
			continue
		}
		sec.Refs = ref.Via
		out = append(out, *sec)
	}
	return out, nil
}

func (s *Service) ConfigMapYAML(ctx context.Context, kubeContext, namespace, name string) (string, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return "", err
	}
	return r.ConfigMapYAML(ctx, namespace, name)
}

func (s *Service) SecretYAML(ctx context.Context, kubeContext, namespace, name string) (string, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return "", err
	}
	return r.SecretYAML(ctx, namespace, name)
}

func (s *Service) DeploymentYAML(ctx context.Context, kubeContext, namespace, name string) (string, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return "", err
	}
	return r.DeploymentYAML(ctx, namespace, name)
}

func (s *Service) PodYAML(ctx context.Context, kubeContext, namespace, name string) (string, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return "", err
	}
	return r.PodYAML(ctx, namespace, name)
}

func (s *Service) StreamPodLogs(ctx context.Context, kubeContext string, opts LogOptions) (io.ReadCloser, error) {
	r, err := s.readerFor(kubeContext)
	if err != nil {
		return nil, err
	}
	return r.StreamPodLogs(ctx, opts)
}

// labelsMatchSelector is a minimal equality selector matcher (k=v,k2=v2).
func labelsMatchSelector(labels map[string]string, selector string) bool {
	if selector == "" {
		return true
	}
	for _, part := range strings.Split(selector, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return false
		}
		if labels[kv[0]] != kv[1] {
			return false
		}
	}
	return true
}
