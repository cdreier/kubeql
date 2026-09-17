package kube

import (
	"context"
	"io"
	"time"
)

// ClusterReader is the read-only surface the GraphQL layer depends on.
// Implementations may talk to a real cluster or an in-memory fake for tests.
type ClusterReader interface {
	ListNamespaces(ctx context.Context) ([]Namespace, error)
	GetNamespace(ctx context.Context, name string) (*Namespace, error)

	ListDeployments(ctx context.Context, namespace string, labelSelector string) ([]Deployment, error)
	GetDeployment(ctx context.Context, namespace, name string) (*Deployment, error)

	ListCronJobs(ctx context.Context, namespace string, labelSelector string) ([]CronJob, error)
	GetCronJob(ctx context.Context, namespace, name string) (*CronJob, error)
	ListJobs(ctx context.Context, namespace string, labelSelector string) ([]Job, error)
	GetJob(ctx context.Context, namespace, name string) (*Job, error)

	ListPods(ctx context.Context, namespace string, labelSelector string) ([]Pod, error)
	GetPod(ctx context.Context, namespace, name string) (*Pod, error)

	ListConfigMaps(ctx context.Context, namespace string) ([]ConfigMap, error)
	GetConfigMap(ctx context.Context, namespace, name string) (*ConfigMap, error)
	ListSecrets(ctx context.Context, namespace string) ([]Secret, error)
	GetSecret(ctx context.Context, namespace, name string) (*Secret, error)

	ListAPIResources(ctx context.Context) ([]APIResource, error)
	ListCustomResources(ctx context.Context, group, version, resource, namespace string) ([]CustomResource, error)
	GetCustomResource(ctx context.Context, group, version, resource, namespace, name string) (*CustomResource, error)
	CustomResourceYAML(ctx context.Context, group, version, resource, namespace, name string) (string, error)

	// *YAML return the object serialized as YAML.
	DeploymentYAML(ctx context.Context, namespace, name string) (string, error)
	CronJobYAML(ctx context.Context, namespace, name string) (string, error)
	JobYAML(ctx context.Context, namespace, name string) (string, error)
	PodYAML(ctx context.Context, namespace, name string) (string, error)
	ConfigMapYAML(ctx context.Context, namespace, name string) (string, error)
	SecretYAML(ctx context.Context, namespace, name string) (string, error)

	// StreamPodLogs follows container logs. Caller must close the returned reader.
	StreamPodLogs(ctx context.Context, opts LogOptions) (io.ReadCloser, error)
}

// LogOptions configures a log stream.
type LogOptions struct {
	Namespace string
	Pod       string
	Container string
	TailLines *int64
	Follow    bool
}

// Namespace is a cluster namespace.
type Namespace struct {
	Name string
}

// Deployment is a subset of apps/v1 Deployment used by the GraphQL layer.
type Deployment struct {
	Name              string
	Namespace         string
	Replicas          int32
	ReadyReplicas     int32
	AvailableReplicas int32
	Status            string
	Labels            map[string]string
	// Selector is the pod label selector belonging to this deployment.
	Selector map[string]string
	// ConfigMapRefs / SecretRefs come from the pod template (env, volumes, …).
	ConfigMapRefs []ObjectRef
	SecretRefs    []ObjectRef
}

// CronJob is a subset of batch/v1 CronJob.
type CronJob struct {
	Name               string
	Namespace          string
	Schedule           string
	TimeZone           string
	Suspend            bool
	ConcurrencyPolicy  string
	LastScheduleTime   *time.Time
	LastSuccessfulTime *time.Time
	Active             int32
	Status             string
	Labels             map[string]string
	ConfigMapRefs      []ObjectRef
	SecretRefs         []ObjectRef
}

// Job is a subset of batch/v1 Job.
type Job struct {
	Name           string
	Namespace      string
	Completions    int32
	Succeeded      int32
	Failed         int32
	Active         int32
	Status         string
	StartTime      *time.Time
	CompletionTime *time.Time
	Labels         map[string]string
	Selector       map[string]string
	OwnerKind      string
	OwnerName      string
	ConfigMapRefs  []ObjectRef
	SecretRefs     []ObjectRef
}

// ObjectRef is a named cluster object plus how a deployment uses it.
type ObjectRef struct {
	Name string
	Via  []string
}

// ConfigMap is a subset of core/v1 ConfigMap.
type ConfigMap struct {
	Name      string
	Namespace string
	Data      map[string]string
	Refs      []string
	Missing   bool
}

// Secret is a subset of core/v1 Secret. Data values are decoded when UTF-8.
type Secret struct {
	Name      string
	Namespace string
	Type      string
	Data      map[string]string
	Refs      []string
	Missing   bool
}

// Pod is a subset of core/v1 Pod used by the GraphQL layer.
type Pod struct {
	Name          string
	Namespace     string
	Phase         string
	Ready         bool
	Restarts      int32
	LastRestartAt *time.Time
	CreatedAt     *time.Time
	NodeName      string
	Labels        map[string]string
	Containers    []Container
	// Optional metrics (populated when metrics-server is available).
	CPUUsage    *string
	MemoryUsage *string
	// Optional limits from the pod spec (sum of container limits).
	CPULimit    *string
	MemoryLimit *string
}

// Container is a container status within a pod.
type Container struct {
	Name         string
	Image        string
	Ready        bool
	RestartCount int32
	State        string
}

// APIResource is a discovered extension API (typically a CRD), not a built-in kind.
type APIResource struct {
	Group      string
	Version    string
	Kind       string
	Resource   string // plural, e.g. kustomizations
	Namespaced bool
}

// CustomResource is a generic CR instance (Flux Kustomization, cert-manager Certificate, …).
type CustomResource struct {
	APIVersion string
	Kind       string
	Group      string
	Version    string
	Resource   string
	Name       string
	Namespace  string // empty when cluster-scoped
	Labels     map[string]string
	Ready      *bool // status.conditions type=Ready/Healthy/Available; nil if unknown
	Reason     string
	Message    string
	CreatedAt  *time.Time
}
