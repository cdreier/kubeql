package kube

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleKubeconfig = `
apiVersion: v1
kind: Config
clusters:
- name: cluster-a
  cluster:
    server: https://a.example.com
- name: cluster-b
  cluster:
    server: https://b.example.com
contexts:
- name: ctx-a
  context:
    cluster: cluster-a
    user: user-a
    namespace: ns-a
- name: ctx-b
  context:
    cluster: cluster-b
    user: user-b
users:
- name: user-a
  user:
    token: token-a
- name: user-b
  user:
    token: token-b
current-context: ctx-a
`

const sampleKubeconfigNoCurrent = `
apiVersion: v1
kind: Config
clusters:
- name: cluster-a
  cluster:
    server: https://a.example.com
- name: cluster-b
  cluster:
    server: https://b.example.com
contexts:
- name: ctx-a
  context:
    cluster: cluster-a
    user: user-a
- name: ctx-b
  context:
    cluster: cluster-b
    user: user-b
users:
- name: user-a
  user:
    token: token-a
- name: user-b
  user:
    token: token-b
`

const sampleKubeconfigSingleNoCurrent = `
apiVersion: v1
kind: Config
clusters:
- name: cluster-a
  cluster:
    server: https://a.example.com
contexts:
- name: only-ctx
  context:
    cluster: cluster-a
    user: user-a
users:
- name: user-a
  user:
    token: token-a
`

func writeKubeconfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenRegistry_ListsContextsAndActive(t *testing.T) {
	path := writeKubeconfig(t, sampleKubeconfig)
	reg, err := OpenRegistry(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if reg.ActiveContext() != "ctx-a" {
		t.Fatalf("active=%q", reg.ActiveContext())
	}
	ctxs := reg.Contexts()
	if len(ctxs) != 2 {
		t.Fatalf("got %d contexts: %#v", len(ctxs), ctxs)
	}
	// Sorted by name.
	if ctxs[0].Name != "ctx-a" || !ctxs[0].Current {
		t.Fatalf("first: %#v", ctxs[0])
	}
	if ctxs[0].Cluster != "cluster-a" || ctxs[0].User != "user-a" || ctxs[0].Namespace != "ns-a" {
		t.Fatalf("ctx-a fields: %#v", ctxs[0])
	}
	if ctxs[1].Name != "ctx-b" || ctxs[1].Current {
		t.Fatalf("second: %#v", ctxs[1])
	}
}

func TestOpenRegistry_PreferredContext(t *testing.T) {
	path := writeKubeconfig(t, sampleKubeconfig)
	reg, err := OpenRegistry(path, "ctx-b")
	if err != nil {
		t.Fatal(err)
	}
	if reg.ActiveContext() != "ctx-b" {
		t.Fatalf("active=%q", reg.ActiveContext())
	}
	for _, c := range reg.Contexts() {
		if c.Name == "ctx-b" && !c.Current {
			t.Fatal("ctx-b should be current")
		}
		if c.Name == "ctx-a" && c.Current {
			t.Fatal("ctx-a should not be current when preferred is ctx-b")
		}
	}
}

func TestOpenRegistry_NoCurrentMultipleContexts(t *testing.T) {
	path := writeKubeconfig(t, sampleKubeconfigNoCurrent)
	reg, err := OpenRegistry(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if reg.HasActiveContext() {
		t.Fatalf("expected no active context, got %q", reg.ActiveContext())
	}
	ctxs := reg.Contexts()
	if len(ctxs) != 2 {
		t.Fatalf("got %d contexts", len(ctxs))
	}
	for _, c := range ctxs {
		if c.Current {
			t.Fatalf("no context should be current: %#v", c)
		}
	}
	_, err = reg.DefaultClient()
	if err == nil {
		t.Fatal("DefaultClient should fail without active context")
	}
	if !strings.Contains(err.Error(), "no active context") {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.Contains(err.Error(), "ctx-a") || !strings.Contains(err.Error(), "ctx-b") {
		t.Fatalf("error should list available contexts: %v", err)
	}

	svc, err := NewServiceFromRegistry(reg)
	if err != nil {
		t.Fatal(err)
	}
	if len(svc.ListContexts()) != 2 {
		t.Fatal("ListContexts should still work")
	}
	_, err = svc.ListNamespaces(context.Background(), "", nil)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("empty context should fail: %v", err)
	}
	// Explicit context still works without a global active context.
	// Client creation succeeds (token/server from sample config); no live API call yet.
	if _, err := reg.Client("ctx-a"); err != nil {
		t.Fatalf("Client(ctx-a): %v", err)
	}
}

func TestOpenRegistry_NoCurrentSingleContext(t *testing.T) {
	path := writeKubeconfig(t, sampleKubeconfigSingleNoCurrent)
	reg, err := OpenRegistry(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if reg.ActiveContext() != "only-ctx" {
		t.Fatalf("active=%q", reg.ActiveContext())
	}
}

func TestOpenRegistry_UnknownPreferred(t *testing.T) {
	path := writeKubeconfig(t, sampleKubeconfig)
	_, err := OpenRegistry(path, "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestRegistry_ClientCachedPerContext(t *testing.T) {
	path := writeKubeconfig(t, sampleKubeconfig)
	reg, err := OpenRegistry(path, "")
	if err != nil {
		t.Fatal(err)
	}
	a1, err := reg.Client("ctx-a")
	if err != nil {
		t.Fatal(err)
	}
	a2, err := reg.Client("ctx-a")
	if err != nil {
		t.Fatal(err)
	}
	if a1 != a2 {
		t.Fatal("expected same client instance for same context")
	}
	if a1.Context() != "ctx-a" {
		t.Fatalf("context name=%q", a1.Context())
	}
	b, err := reg.Client("ctx-b")
	if err != nil {
		t.Fatal(err)
	}
	if b == a1 {
		t.Fatal("expected different client per context")
	}
	if b.Context() != "ctx-b" {
		t.Fatalf("context name=%q", b.Context())
	}
}

func TestService_ListContexts(t *testing.T) {
	path := writeKubeconfig(t, sampleKubeconfig)
	reg, err := OpenRegistry(path, "ctx-b")
	if err != nil {
		t.Fatal(err)
	}
	// Avoid DefaultClient network — attach registry with a fake reader.
	svc := &Service{Reader: NewFake(), Registry: reg}
	list := svc.ListContexts()
	if len(list) != 2 {
		t.Fatalf("got %d", len(list))
	}
	var current string
	for _, c := range list {
		if c.Current {
			current = c.Name
		}
	}
	if current != "ctx-b" {
		t.Fatalf("current=%q", current)
	}
}

func TestService_ListContexts_NoRegistry(t *testing.T) {
	svc := NewService(NewFake())
	if got := svc.ListContexts(); len(got) != 0 {
		t.Fatalf("expected empty, got %#v", got)
	}
}
