package kube

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseAPIResourceLists_SkipsBuiltinAndSubresources(t *testing.T) {
	lists := []*metav1.APIResourceList{
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: []string{"get", "list"}},
			},
		},
		{
			GroupVersion: "kustomize.toolkit.fluxcd.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "kustomizations", Kind: "Kustomization", Namespaced: true, Verbs: []string{"get", "list", "watch"}},
				{Name: "kustomizations/status", Kind: "Kustomization", Namespaced: true, Verbs: []string{"get"}},
			},
		},
		{
			GroupVersion: "cert-manager.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "clusterissuers", Kind: "ClusterIssuer", Namespaced: false, Verbs: []string{"get", "list"}},
				{Name: "certificates", Kind: "Certificate", Namespaced: true, Verbs: []string{"create"}},
			},
		},
	}
	got := parseAPIResourceLists(lists)
	if len(got) != 2 {
		t.Fatalf("got %d resources, want 2: %#v", len(got), got)
	}
	if got[0].Kind != "ClusterIssuer" || got[0].Namespaced {
		t.Fatalf("first: %#v", got[0])
	}
	if got[1].Kind != "Kustomization" || got[1].Resource != "kustomizations" {
		t.Fatalf("second: %#v", got[1])
	}
}

func TestConditionStatus_Ready(t *testing.T) {
	obj := map[string]interface{}{
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":    "Ready",
					"status":  "False",
					"reason":  "BuildFailed",
					"message": "kustomize build failed",
				},
			},
		},
	}
	ready, reason, message := conditionStatus(obj)
	if ready == nil || *ready {
		t.Fatalf("ready=%v, want false", ready)
	}
	if reason != "BuildFailed" || message != "kustomize build failed" {
		t.Fatalf("reason=%q message=%q", reason, message)
	}
}

func TestConditionStatus_Unknown(t *testing.T) {
	ready, reason, message := conditionStatus(map[string]interface{}{})
	if ready != nil || reason != "" || message != "" {
		t.Fatalf("got ready=%v reason=%q message=%q", ready, reason, message)
	}
}

func TestListCustomResources_KindContains(t *testing.T) {
	f := NewFake()
	f.APIResources = []APIResource{
		{Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Kind: "Kustomization", Resource: "kustomizations", Namespaced: true},
		{Group: "helm.toolkit.fluxcd.io", Version: "v2", Kind: "HelmRelease", Resource: "helmreleases", Namespaced: true},
		{Group: "cert-manager.io", Version: "v1", Kind: "Certificate", Resource: "certificates", Namespaced: true},
	}
	f.CustomResources = []CustomResource{
		{Name: "apps", Namespace: "prod", Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Kind: "Kustomization", Resource: "kustomizations"},
		{Name: "web", Namespace: "prod", Group: "helm.toolkit.fluxcd.io", Version: "v2", Kind: "HelmRelease", Resource: "helmreleases"},
		{Name: "tls", Namespace: "prod", Group: "cert-manager.io", Version: "v1", Kind: "Certificate", Resource: "certificates"},
		{Name: "other", Namespace: "default", Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Kind: "Kustomization", Resource: "kustomizations"},
	}
	svc := NewService(f)
	needle := "flux"
	list, err := svc.ListCustomResources(context.Background(), testCtx, "prod", &CustomResourceFilter{
		KindContains: &needle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d, want 2 flux CRs in prod: %#v", len(list), list)
	}
	if list[0].Kind != "HelmRelease" || list[1].Kind != "Kustomization" {
		t.Fatalf("sort: %#v %#v", list[0], list[1])
	}
}

func TestMatchAPIResource_KindContains(t *testing.T) {
	a := APIResource{Group: "kustomize.toolkit.fluxcd.io", Kind: "Kustomization", Resource: "kustomizations"}
	needle := "kustom"
	if !MatchAPIResource(a, &CustomResourceFilter{KindContains: &needle}) {
		t.Fatal("expected match on kind")
	}
	other := "cert"
	if MatchAPIResource(a, &CustomResourceFilter{KindContains: &other}) {
		t.Fatal("did not expect match")
	}
}
