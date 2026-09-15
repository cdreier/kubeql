package kube

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
)

// Groups that ship with kube-apiserver (and metrics-server). Everything else
// is treated as an extension API / CRD (Flux, cert-manager, Gateway API, …).
var builtinAPIGroups = map[string]struct{}{
	"":                             {},
	"admissionregistration.k8s.io": {},
	"apiextensions.k8s.io":         {},
	"apiregistration.k8s.io":       {},
	"apps":                         {},
	"authentication.k8s.io":        {},
	"authorization.k8s.io":         {},
	"autoscaling":                  {},
	"batch":                        {},
	"certificates.k8s.io":          {},
	"coordination.k8s.io":          {},
	"discovery.k8s.io":             {},
	"events.k8s.io":                {},
	"flowcontrol.apiserver.k8s.io": {},
	"internal.apiserver.k8s.io":    {},
	"metrics.k8s.io":               {},
	"custom.metrics.k8s.io":        {},
	"external.metrics.k8s.io":      {},
	"networking.k8s.io":            {},
	"node.k8s.io":                  {},
	"policy":                       {},
	"rbac.authorization.k8s.io":    {},
	"resource.k8s.io":              {},
	"scheduling.k8s.io":            {},
	"storage.k8s.io":               {},
	"storagemigration.k8s.io":      {},
}

func isBuiltinAPIGroup(group string) bool {
	_, ok := builtinAPIGroups[group]
	return ok
}

func (c *Client) ListAPIResources(ctx context.Context) ([]APIResource, error) {
	_ = ctx
	if c.cs == nil {
		return []APIResource{}, nil
	}
	lists, err := c.cs.Discovery().ServerPreferredResources()
	if len(lists) == 0 && err != nil {
		return nil, err
	}
	return parseAPIResourceLists(lists), nil
}

func parseAPIResourceLists(lists []*metav1.APIResourceList) []APIResource {
	out := make([]APIResource, 0)
	for _, list := range lists {
		if list == nil {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		if isBuiltinAPIGroup(gv.Group) {
			continue
		}
		for _, r := range list.APIResources {
			if strings.Contains(r.Name, "/") {
				continue
			}
			if !hasVerb(r.Verbs, "list") || !hasVerb(r.Verbs, "get") {
				continue
			}
			out = append(out, APIResource{
				Group:      gv.Group,
				Version:    gv.Version,
				Kind:       r.Kind,
				Resource:   r.Name,
				Namespaced: r.Namespaced,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Resource < out[j].Resource
	})
	return out
}

func hasVerb(verbs []string, want string) bool {
	for _, v := range verbs {
		if v == want || v == "*" {
			return true
		}
	}
	return false
}

func (c *Client) ListCustomResources(ctx context.Context, group, version, resource, namespace string) ([]CustomResource, error) {
	ri, err := c.customResourceInterface(group, version, resource, namespace)
	if err != nil {
		return []CustomResource{}, nil
	}
	list, err := ri.List(ctx, metav1.ListOptions{})
	if err != nil {
		if skipCustomResourceError(err) {
			return []CustomResource{}, nil
		}
		return nil, err
	}
	out := make([]CustomResource, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapUnstructured(&list.Items[i], resource))
	}
	return out, nil
}

func (c *Client) GetCustomResource(ctx context.Context, group, version, resource, namespace, name string) (*CustomResource, error) {
	ri, err := c.customResourceInterface(group, version, resource, namespace)
	if err != nil {
		return nil, nil
	}
	u, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) || skipCustomResourceError(err) {
			return nil, nil
		}
		return nil, err
	}
	mapped := mapUnstructured(u, resource)
	return &mapped, nil
}

func (c *Client) CustomResourceYAML(ctx context.Context, group, version, resource, namespace, name string) (string, error) {
	ri, err := c.customResourceInterface(group, version, resource, namespace)
	if err != nil {
		return "", err
	}
	u, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return unstructuredYAML(u)
}

func (c *Client) customResourceInterface(group, version, resource, namespace string) (dynamic.ResourceInterface, error) {
	if c.dyn == nil {
		return nil, errNoDynamicClient
	}
	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	n := c.dyn.Resource(gvr)
	if namespace != "" {
		return n.Namespace(namespace), nil
	}
	return n, nil
}

var errNoDynamicClient = fmt.Errorf("dynamic kubernetes client is not configured")

func skipCustomResourceError(err error) bool {
	return apierrors.IsForbidden(err) ||
		apierrors.IsNotFound(err) ||
		apierrors.IsMethodNotSupported(err) ||
		apierrors.IsServiceUnavailable(err)
}

func mapUnstructured(u *unstructured.Unstructured, resource string) CustomResource {
	gvk := u.GroupVersionKind()
	ready, reason, message := conditionStatus(u.Object)
	var created *time.Time
	if t := u.GetCreationTimestamp(); !t.IsZero() {
		tt := t.Time
		created = &tt
	}
	return CustomResource{
		APIVersion: u.GetAPIVersion(),
		Kind:       u.GetKind(),
		Group:      gvk.Group,
		Version:    gvk.Version,
		Resource:   resource,
		Name:       u.GetName(),
		Namespace:  u.GetNamespace(),
		Labels:     copyMap(u.GetLabels()),
		Ready:      ready,
		Reason:     reason,
		Message:    message,
		CreatedAt:  created,
	}
}

func conditionStatus(obj map[string]interface{}) (ready *bool, reason, message string) {
	raw, found, err := unstructured.NestedSlice(obj, "status", "conditions")
	if err != nil || !found {
		return nil, "", ""
	}
	byType := make(map[string]map[string]string, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "" {
			continue
		}
		entry := map[string]string{}
		if s, ok := m["status"].(string); ok {
			entry["status"] = s
		}
		if s, ok := m["reason"].(string); ok {
			entry["reason"] = s
		}
		if s, ok := m["message"].(string); ok {
			entry["message"] = s
		}
		byType[t] = entry
	}
	var chosen map[string]string
	for _, t := range []string{"Ready", "Healthy", "Available"} {
		if c, ok := byType[t]; ok {
			chosen = c
			break
		}
	}
	if chosen == nil {
		return nil, "", ""
	}
	switch strings.ToLower(chosen["status"]) {
	case "true":
		v := true
		ready = &v
	case "false":
		v := false
		ready = &v
	}
	return ready, chosen["reason"], chosen["message"]
}

func unstructuredYAML(u *unstructured.Unstructured) (string, error) {
	obj := u.DeepCopy().Object
	unstructured.RemoveNestedField(obj, "metadata", "managedFields")
	b, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
