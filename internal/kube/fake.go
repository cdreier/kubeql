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
	mu              sync.RWMutex
	Namespaces      []Namespace
	Deployments     []Deployment
	CronJobs        []CronJob
	Jobs            []Job
	Pods            []Pod
	ConfigMaps      []ConfigMap
	Secrets         []Secret
	APIResources    []APIResource
	CustomResources []CustomResource
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

func (f *Fake) ListCronJobs(ctx context.Context, namespace, labelSelector string) ([]CronJob, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []CronJob
	for _, cj := range f.CronJobs {
		if namespace != "" && cj.Namespace != namespace {
			continue
		}
		if labelSelector != "" && !matchLabels(cj.Labels, labelSelector) {
			continue
		}
		out = append(out, cj)
	}
	return out, nil
}

func (f *Fake) GetCronJob(ctx context.Context, namespace, name string) (*CronJob, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, cj := range f.CronJobs {
		if cj.Namespace == namespace && cj.Name == name {
			cp := cj
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *Fake) ListJobs(ctx context.Context, namespace, labelSelector string) ([]Job, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []Job
	for _, j := range f.Jobs {
		if namespace != "" && j.Namespace != namespace {
			continue
		}
		if labelSelector != "" && !matchLabels(j.Labels, labelSelector) {
			continue
		}
		out = append(out, j)
	}
	return out, nil
}

func (f *Fake) GetJob(ctx context.Context, namespace, name string) (*Job, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, j := range f.Jobs {
		if j.Namespace == namespace && j.Name == name {
			cp := j
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

func (f *Fake) CronJobYAML(ctx context.Context, namespace, name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := "cronjob/" + namespace + "/" + name
	if y, ok := f.YAML[key]; ok {
		return y, nil
	}
	return fmt.Sprintf("apiVersion: batch/v1\nkind: CronJob\nmetadata:\n  name: %s\n  namespace: %s\n", name, namespace), nil
}

func (f *Fake) JobYAML(ctx context.Context, namespace, name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := "job/" + namespace + "/" + name
	if y, ok := f.YAML[key]; ok {
		return y, nil
	}
	return fmt.Sprintf("apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: %s\n  namespace: %s\n", name, namespace), nil
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

func (f *Fake) ListAPIResources(ctx context.Context) ([]APIResource, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]APIResource, len(f.APIResources))
	copy(out, f.APIResources)
	return out, nil
}

func (f *Fake) ListCustomResources(ctx context.Context, group, version, resource, namespace string) ([]CustomResource, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []CustomResource
	for _, cr := range f.CustomResources {
		if cr.Group != group || cr.Version != version || cr.Resource != resource {
			continue
		}
		if namespace != "" && cr.Namespace != namespace {
			continue
		}
		out = append(out, cr)
	}
	return out, nil
}

func (f *Fake) GetCustomResource(ctx context.Context, group, version, resource, namespace, name string) (*CustomResource, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, cr := range f.CustomResources {
		if cr.Group == group && cr.Version == version && cr.Resource == resource &&
			cr.Namespace == namespace && cr.Name == name {
			cp := cr
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *Fake) CustomResourceYAML(ctx context.Context, group, version, resource, namespace, name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := "crd/" + group + "/" + version + "/" + resource + "/" + namespace + "/" + name
	if y, ok := f.YAML[key]; ok {
		return y, nil
	}
	kind := resource
	for _, cr := range f.CustomResources {
		if cr.Group == group && cr.Version == version && cr.Resource == resource &&
			cr.Namespace == namespace && cr.Name == name {
			kind = cr.Kind
			break
		}
	}
	apiVersion := version
	if group != "" {
		apiVersion = group + "/" + version
	}
	return fmt.Sprintf("apiVersion: %s\nkind: %s\nmetadata:\n  name: %s\n  namespace: %s\n", apiVersion, kind, name, namespace), nil
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
