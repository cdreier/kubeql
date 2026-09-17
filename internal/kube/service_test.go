package kube

import (
	"context"
	"testing"
)

func sampleCluster() *Fake {
	f := NewFake()
	f.Namespaces = []Namespace{
		{Name: "default"},
		{Name: "prod"},
		{Name: "kube-system"},
	}
	f.Deployments = []Deployment{
		{
			Name: "api", Namespace: "prod", Replicas: 2, ReadyReplicas: 2,
			Status: "2/2", Labels: map[string]string{"app": "api"},
			Selector: map[string]string{"app": "api"},
		},
		{
			Name: "web", Namespace: "default", Replicas: 1, ReadyReplicas: 1,
			Status: "1/1", Labels: map[string]string{"app": "web"},
			Selector: map[string]string{"app": "web"},
		},
	}
	f.Pods = []Pod{
		{
			Name: "api-1", Namespace: "prod", Phase: "Running", Ready: true, Restarts: 2,
			Labels: map[string]string{"app": "api"},
		},
		{
			Name: "api-2", Namespace: "prod", Phase: "Running", Ready: false, Restarts: 0,
			Labels: map[string]string{"app": "api"},
		},
		{
			Name: "web-1", Namespace: "default", Phase: "Running", Ready: true, Restarts: 1,
			Labels: map[string]string{"app": "web"},
		},
	}
	return f
}

const testCtx = "test"

func TestListNamespaces_HasDeployment(t *testing.T) {
	svc := NewService(sampleCluster())
	name := "api"
	list, err := svc.ListNamespaces(context.Background(), testCtx, &NamespaceFilter{
		HasDeployment: &DeploymentFilter{NameContains: &name},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "prod" {
		t.Fatalf("expected only prod, got %#v", list)
	}
}

func TestListNamespaces_HasPod(t *testing.T) {
	svc := NewService(sampleCluster())
	name := "web"
	list, err := svc.ListNamespaces(context.Background(), testCtx, &NamespaceFilter{
		HasPod: &PodFilter{NameContains: &name},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "default" {
		t.Fatalf("expected only default, got %#v", list)
	}
}

func TestListDeployments_HasPodReady(t *testing.T) {
	svc := NewService(sampleCluster())
	ready := false
	list, err := svc.ListDeployments(context.Background(), testCtx, "prod", &DeploymentFilter{
		HasPod: &PodFilter{Ready: &ready},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "api" {
		t.Fatalf("expected api (has not-ready pod), got %#v", list)
	}
}

func TestPodsForDeployment(t *testing.T) {
	svc := NewService(sampleCluster())
	d, err := svc.GetDeployment(context.Background(), testCtx, "prod", "api")
	if err != nil || d == nil {
		t.Fatalf("deployment: %v %#v", err, d)
	}
	pods, err := svc.PodsForDeployment(context.Background(), testCtx, *d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pods) != 2 {
		t.Fatalf("expected 2 pods, got %d", len(pods))
	}
}

func TestDeploymentRestarts(t *testing.T) {
	svc := NewService(sampleCluster())
	d, _ := svc.GetDeployment(context.Background(), testCtx, "prod", "api")
	n, err := svc.DeploymentRestarts(context.Background(), testCtx, *d)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 restarts, got %d", n)
	}
}

func TestConfigMapsForDeployment_MissingAndFound(t *testing.T) {
	f := sampleCluster()
	f.Deployments[0].ConfigMapRefs = []ObjectRef{
		{Name: "app-config", Via: []string{"envFrom"}},
		{Name: "gone", Via: []string{"volume:cfg"}},
	}
	f.ConfigMaps = []ConfigMap{{
		Name: "app-config", Namespace: "prod",
		Data: map[string]string{"LOG": "info"},
	}}
	svc := NewService(f)
	d, _ := svc.GetDeployment(context.Background(), testCtx, "prod", "api")
	cms, err := svc.ConfigMapsForDeployment(context.Background(), testCtx, *d)
	if err != nil {
		t.Fatal(err)
	}
	if len(cms) != 2 {
		t.Fatalf("got %d configmaps", len(cms))
	}
	if cms[0].Name != "app-config" || cms[0].Missing || cms[0].Data["LOG"] != "info" {
		t.Fatalf("found map: %#v", cms[0])
	}
	if cms[1].Name != "gone" || !cms[1].Missing {
		t.Fatalf("missing map: %#v", cms[1])
	}
}

func TestListCronJobs_AndJobs(t *testing.T) {
	f := sampleCluster()
	f.CronJobs = []CronJob{
		{Name: "backup", Namespace: "prod", Schedule: "0 * * * *", Status: "Idle"},
		{Name: "nightly", Namespace: "default", Schedule: "0 2 * * *", Suspend: true, Status: "Suspended"},
	}
	f.Jobs = []Job{
		{
			Name: "backup-1", Namespace: "prod", Status: "Complete", Succeeded: 1, Completions: 1,
			OwnerKind: "CronJob", OwnerName: "backup",
			Selector: map[string]string{"job-name": "backup-1"},
		},
		{
			Name: "other-1", Namespace: "prod", Status: "Complete",
			OwnerKind: "CronJob", OwnerName: "other",
			Selector: map[string]string{"job-name": "other-1"},
		},
	}
	f.Pods = append(f.Pods, Pod{
		Name: "backup-1-pod", Namespace: "prod", Phase: "Succeeded", Ready: false,
		Labels: map[string]string{"job-name": "backup-1"},
	})
	svc := NewService(f)

	list, err := svc.ListCronJobs(context.Background(), testCtx, "prod", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "backup" {
		t.Fatalf("expected backup in prod, got %#v", list)
	}

	suspended := true
	list, err = svc.ListCronJobs(context.Background(), testCtx, "", &CronJobFilter{Suspended: &suspended})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "nightly" {
		t.Fatalf("expected nightly suspended, got %#v", list)
	}

	cj, err := svc.GetCronJob(context.Background(), testCtx, "prod", "backup")
	if err != nil || cj == nil {
		t.Fatalf("cronjob: %v %#v", err, cj)
	}
	jobs, err := svc.JobsForCronJob(context.Background(), testCtx, *cj)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].Name != "backup-1" {
		t.Fatalf("expected backup-1, got %#v", jobs)
	}

	pods, err := svc.PodsForCronJob(context.Background(), testCtx, *cj, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pods) != 1 || pods[0].Name != "backup-1-pod" {
		t.Fatalf("expected backup-1-pod, got %#v", pods)
	}
}

func TestListPods_NameContains(t *testing.T) {
	svc := NewService(sampleCluster())
	needle := "web"
	pods, err := svc.ListPods(context.Background(), testCtx, "", &PodFilter{NameContains: &needle})
	if err != nil {
		t.Fatal(err)
	}
	if len(pods) != 1 || pods[0].Name != "web-1" {
		t.Fatalf("got %#v", pods)
	}
}
