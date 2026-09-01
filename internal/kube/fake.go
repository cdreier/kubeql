package kube

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Fake is an in-memory ClusterReader for unit tests.
type Fake struct {
	mu          sync.RWMutex
	Namespaces  []Namespace
	Deployments []Deployment
	Pods        []Pod
	ConfigMaps  []ConfigMap
	Secrets     []Secret
	// YAML keyed by "kind/namespace/name"
	YAML map[string]string
	// Logs keyed by "namespace/pod[/container]"
	Logs map[string]string
}

func NewFake() *Fake {
	return &Fake{
		YAML: map[string]string{},
		Logs: map[string]string{},
	}
}

func (f *Fake) ListNamespaces(ctx context.Context) ([]Namespace, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]Namespace, len(f.Namespaces))
	copy(out, f.Namespaces)
	return out, nil
}

func (f *Fake) GetNamespace(ctx context.Context, name string) (*Namespace, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, ns := range f.Namespaces {
		if ns.Name == name {
			cp := ns
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *Fake) ListDeployments(ctx context.Context, namespace, labelSelector string) ([]Deployment, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []Deployment
	for _, d := range f.Deployments {
		if namespace != "" && d.Namespace != namespace {
			continue
		}
		if labelSelector != "" && !matchLabels(d.Labels, labelSelector) {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (f *Fake) GetDeployment(ctx context.Context, namespace, name string) (*Deployment, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, d := range f.Deployments {
		if d.Namespace == namespace && d.Name == name {
			cp := d
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *Fake) ListPods(ctx context.Context, namespace, labelSelector string) ([]Pod, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []Pod
	for _, p := range f.Pods {
		if namespace != "" && p.Namespace != namespace {
			continue
		}
		if labelSelector != "" && !matchLabels(p.Labels, labelSelector) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (f *Fake) GetPod(ctx context.Context, namespace, name string) (*Pod, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, p := range f.Pods {
		if p.Namespace == namespace && p.Name == name {
			cp := p
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *Fake) ListConfigMaps(ctx context.Context, namespace string) ([]ConfigMap, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []ConfigMap
	for _, cm := range f.ConfigMaps {
		if namespace != "" && cm.Namespace != namespace {
			continue
		}
		out = append(out, cm)
	}
	return out, nil
}

func (f *Fake) GetConfigMap(ctx context.Context, namespace, name string) (*ConfigMap, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, cm := range f.ConfigMaps {
		if cm.Namespace == namespace && cm.Name == name {
			cp := cm
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *Fake) ListSecrets(ctx context.Context, namespace string) ([]Secret, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []Secret
	for _, s := range f.Secrets {
		if namespace != "" && s.Namespace != namespace {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func (f *Fake) GetSecret(ctx context.Context, namespace, name string) (*Secret, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, s := range f.Secrets {
		if s.Namespace == namespace && s.Name == name {
			cp := s
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *Fake) ConfigMapYAML(ctx context.Context, namespace, name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := "configmap/" + namespace + "/" + name
	if y, ok := f.YAML[key]; ok {
		return y, nil
	}
	return fmt.Sprintf("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: %s\n  namespace: %s\n", name, namespace), nil
}

func (f *Fake) SecretYAML(ctx context.Context, namespace, name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := "secret/" + namespace + "/" + name
	if y, ok := f.YAML[key]; ok {
		return y, nil
	}
	return fmt.Sprintf("apiVersion: v1\nkind: Secret\nmetadata:\n  name: %s\n  namespace: %s\n", name, namespace), nil
}

func (f *Fake) DeploymentYAML(ctx context.Context, namespace, name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := "deployment/" + namespace + "/" + name
	if y, ok := f.YAML[key]; ok {
		return y, nil
	}
	return fmt.Sprintf("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: %s\n  namespace: %s\n", name, namespace), nil
}

func (f *Fake) PodYAML(ctx context.Context, namespace, name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := "pod/" + namespace + "/" + name
	if y, ok := f.YAML[key]; ok {
		return y, nil
	}
	return fmt.Sprintf("apiVersion: v1\nkind: Pod\nmetadata:\n  name: %s\n  namespace: %s\n", name, namespace), nil
}

func (f *Fake) StreamPodLogs(ctx context.Context, opts LogOptions) (io.ReadCloser, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	keys := []string{
		opts.Namespace + "/" + opts.Pod + "/" + opts.Container,
		opts.Namespace + "/" + opts.Pod,
	}
	for _, k := range keys {
		if opts.Container == "" && strings.HasSuffix(k, "/") {
			continue
		}
		if body, ok := f.Logs[k]; ok {
			return io.NopCloser(bytes.NewBufferString(body)), nil
		}
	}
	return io.NopCloser(bytes.NewBufferString("")), nil
}

// matchLabels is a minimal "k=v,k2=v2" matcher for the fake.
func matchLabels(labels map[string]string, selector string) bool {
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
