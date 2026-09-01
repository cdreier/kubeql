package kube

import "testing"

func TestMatchPod(t *testing.T) {
	p := Pod{Name: "api-abc", Phase: "Running", Ready: true}

	if !MatchPod(p, nil) {
		t.Fatal("nil filter should match")
	}

	name := "API"
	if !MatchPod(p, &PodFilter{NameContains: &name}) {
		t.Fatal("expected case-insensitive name match")
	}

	phase := "Pending"
	if MatchPod(p, &PodFilter{Phase: &phase}) {
		t.Fatal("phase mismatch should not match")
	}

	ready := false
	if MatchPod(p, &PodFilter{Ready: &ready}) {
		t.Fatal("ready mismatch should not match")
	}
}

func TestMatchDeployment(t *testing.T) {
	d := Deployment{Name: "frontend"}
	needle := "end"
	if !MatchDeployment(d, &DeploymentFilter{NameContains: &needle}) {
		t.Fatal("expected substring match")
	}
	other := "backend"
	if MatchDeployment(d, &DeploymentFilter{NameContains: &other}) {
		t.Fatal("unexpected match")
	}
}

func TestSelectorToString(t *testing.T) {
	s := SelectorToString(map[string]string{"app": "api"})
	if s != "app=api" {
		t.Fatalf("got %q", s)
	}
}
