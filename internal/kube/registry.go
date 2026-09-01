package kube

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// ContextInfo describes one kubeconfig context entry.
type ContextInfo struct {
	Name      string
	Cluster   string
	User      string
	Namespace string // optional default namespace on the context
	// Current is true when this matches CLI --context, kubeconfig current-context,
	// or the sole context (informational default; queries still pass context explicitly).
	Current bool
}

// Registry loads a kubeconfig once, exposes its contexts, and builds one
// client-go clientset per context on demand (lazy, cached).
// Multi-context GraphQL selection can later call Client(name) per request.
type Registry struct {
	path          string
	raw           *clientcmdapi.Config
	activeContext string
	clients       map[string]*Client
	mu            sync.Mutex
}

// OpenRegistry loads kubeconfig and lists contexts before any cluster client is created.
//
// preferredContext (e.g. --context) wins over kubeconfig current-context.
// If neither is set and exactly one context exists, that one becomes active.
// If neither is set and multiple contexts exist, ActiveContext is empty: the server
// still starts so `contexts` can be queried; cluster queries need --context (or later
// a per-request context) until an active context is chosen.
func OpenRegistry(kubeconfigPath, preferredContext string) (*Registry, error) {
	path, err := expandPath(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("resolve kubeconfig path: %w", err)
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if path != "" {
		loadingRules.ExplicitPath = path
	}

	raw, err := loadingRules.Load()
	if err != nil {
		hint := clientcmd.RecommendedHomeFile
		return nil, fmt.Errorf("load kubeconfig (tried %q): %w",
			firstNonEmpty(path, os.Getenv(clientcmd.RecommendedConfigPathEnvVar), hint), err)
	}

	active, err := resolveActiveContext(raw, preferredContext)
	if err != nil {
		return nil, err
	}

	return &Registry{
		path:          path,
		raw:           raw,
		activeContext: active,
		clients:       make(map[string]*Client),
	}, nil
}

// ActiveContext returns the context name used for DefaultClient.
func (r *Registry) ActiveContext() string {
	return r.activeContext
}

// Contexts returns all kubeconfig contexts. The active one has Current=true.
func (r *Registry) Contexts() []ContextInfo {
	names := make([]string, 0, len(r.raw.Contexts))
	for name := range r.raw.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]ContextInfo, 0, len(names))
	for _, name := range names {
		ctx := r.raw.Contexts[name]
		info := ContextInfo{
			Name:    name,
			Current: name == r.activeContext,
		}
		if ctx != nil {
			info.Cluster = ctx.Cluster
			info.User = ctx.AuthInfo
			info.Namespace = ctx.Namespace
		}
		out = append(out, info)
	}
	return out
}

// HasActiveContext reports whether a default cluster context was resolved.
func (r *Registry) HasActiveContext() bool {
	return r.activeContext != ""
}

// DefaultClient returns (and caches) a client for the active context.
// Returns a clear error when no active context is set (multi-context kubeconfig
// without current-context and without --context).
func (r *Registry) DefaultClient() (*Client, error) {
	if r.activeContext == "" {
		return nil, fmt.Errorf(
			"no active context; pass --context or query { contexts { name } } (available: %s)",
			strings.Join(contextNames(r.raw), ", "),
		)
	}
	return r.Client(r.activeContext)
}

// Client returns a cached ClusterReader client for the given context name,
// creating it on first use. One kubernetes clientset per context.
func (r *Registry) Client(contextName string) (*Client, error) {
	if contextName == "" {
		return nil, fmt.Errorf("context name is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.clients[contextName]; ok {
		return c, nil
	}
	if _, ok := r.raw.Contexts[contextName]; !ok {
		return nil, fmt.Errorf("context %q not found (available: %s)",
			contextName, strings.Join(contextNames(r.raw), ", "))
	}

	cfg, err := clientcmd.NewDefaultClientConfig(*r.raw, &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("rest config for context %q: %w", contextName, err)
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client for context %q: %w", contextName, err)
	}

	c := &Client{cs: cs, context: contextName}
	if mc, err := metricsclient.NewForConfig(cfg); err == nil {
		c.metrics = mc
	}
	r.clients[contextName] = c
	return c, nil
}

// resolveActiveContext picks the default context for cluster queries.
// Empty string means "no default": server may still list contexts.
func resolveActiveContext(raw *clientcmdapi.Config, preferred string) (string, error) {
	names := contextNames(raw)
	if len(names) == 0 {
		return "", fmt.Errorf("kubeconfig has no contexts")
	}

	if preferred != "" {
		if _, ok := raw.Contexts[preferred]; !ok {
			return "", fmt.Errorf("context %q not found (available: %s)",
				preferred, strings.Join(names, ", "))
		}
		return preferred, nil
	}

	if raw.CurrentContext != "" {
		if _, ok := raw.Contexts[raw.CurrentContext]; !ok {
			return "", fmt.Errorf("current-context %q not found in kubeconfig (available: %s)",
				raw.CurrentContext, strings.Join(names, ", "))
		}
		return raw.CurrentContext, nil
	}

	// No current-context in the file (common with some multi-file setups).
	if len(names) == 1 {
		return names[0], nil
	}

	// Multiple contexts, none selected — start without a default client.
	return "", nil
}

func contextNames(raw *clientcmdapi.Config) []string {
	names := make([]string, 0, len(raw.Contexts))
	for name := range raw.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
