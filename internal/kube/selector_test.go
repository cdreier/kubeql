package kube

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPodSelector(t *testing.T) {
	cases := []struct {
		name string
		in   *metav1.LabelSelector
		want string
	}{
		{"nil selects nothing", nil, ""},
		{"empty selects nothing", &metav1.LabelSelector{}, ""},
		{"matchLabels", &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}}, "app=api"},
		{"matchExpressions In", &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"api", "web"}},
		}}, "app in (api,web)"},
		{"matchLabels + Exists", &metav1.LabelSelector{
			MatchLabels:      map[string]string{"app": "api"},
			MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "tier", Operator: metav1.LabelSelectorOpExists}},
		}, "app=api,tier"},
		{"invalid operator selects nothing", &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "app", Operator: "Bogus", Values: []string{"api"}},
		}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := podSelector(tc.in); got != tc.want {
				t.Fatalf("podSelector = %q, want %q", got, tc.want)
			}
		})
	}
}

func selectorTestPods() []*corev1.Pod {
	pod := func(name string, lbls map[string]string, restarts int32) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns", Labels: lbls},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "c", RestartCount: restarts},
			}},
		}
	}
	return []*corev1.Pod{
		pod("api-1", map[string]string{"app": "api", "tier": "backend"}, 1),
		pod("api-2", map[string]string{"app": "api"}, 2),
		pod("db-1", map[string]string{"app": "db"}, 5),
		pod("job-1", map[string]string{"batch.kubernetes.io/controller-uid": "uid-1"}, 0),
	}
}

func newSelectorTestService(t *testing.T, objs ...*corev1.Pod) (*Service, *fake.Clientset) {
	t.Helper()
	cs := fake.NewSimpleClientset()
	for _, p := range objs {
		if _, err := cs.CoreV1().Pods(p.Namespace).Create(context.Background(), p, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	return NewService(NewClientFromInterface(cs)), cs
}

func podNames(pods []Pod) map[string]bool {
	out := map[string]bool{}
	for _, p := range pods {
		out[p.Name] = true
	}
	return out
}

func TestPodsForDeployment_MatchExpressions(t *testing.T) {
	ctx := context.Background()
	svc, cs := newSelectorTestService(t, selectorTestPods()...)
	deps := []*appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "ns"},
			Spec: appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
				{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"api"}},
			}}},
		},
		{
			// Not accepted by a real API server; must not fall back to "all pods".
			ObjectMeta: metav1.ObjectMeta{Name: "noselector", Namespace: "ns"},
		},
	}
	for _, d := range deps {
		if _, err := cs.AppsV1().Deployments("ns").Create(ctx, d, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	d, err := svc.GetDeployment(ctx, "", "ns", "api")
	if err != nil || d == nil {
		t.Fatalf("get deployment: %v %v", d, err)
	}
	pods, err := svc.PodsForDeployment(ctx, "", *d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := podNames(pods); len(got) != 2 || !got["api-1"] || !got["api-2"] {
		t.Fatalf("expected api-1, api-2; got %v", got)
	}
	restarts, err := svc.DeploymentRestarts(ctx, "", *d)
	if err != nil {
		t.Fatal(err)
	}
	if restarts != 3 {
		t.Fatalf("restarts = %d, want 3 (db-1 must not be counted)", restarts)
	}

	none, err := svc.GetDeployment(ctx, "", "ns", "noselector")
	if err != nil || none == nil {
		t.Fatalf("get deployment: %v %v", none, err)
	}
	pods, err = svc.PodsForDeployment(ctx, "", *none, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pods) != 0 {
		t.Fatalf("deployment without selector must select no pods, got %v", podNames(pods))
	}
}

func TestPodsForJob_Selector(t *testing.T) {
	ctx := context.Background()
	svc, cs := newSelectorTestService(t, selectorTestPods()...)
	jobs := []*batchv1.Job{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "job", Namespace: "ns"},
			Spec: batchv1.JobSpec{Selector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
				{Key: "batch.kubernetes.io/controller-uid", Operator: metav1.LabelSelectorOpIn, Values: []string{"uid-1"}},
			}}},
		},
		{ObjectMeta: metav1.ObjectMeta{Name: "noselector", Namespace: "ns"}},
	}
	for _, j := range jobs {
		if _, err := cs.BatchV1().Jobs("ns").Create(ctx, j, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	j, err := svc.GetJob(ctx, "", "ns", "job")
	if err != nil || j == nil {
		t.Fatalf("get job: %v %v", j, err)
	}
	pods, err := svc.PodsForJob(ctx, "", *j, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := podNames(pods); len(got) != 1 || !got["job-1"] {
		t.Fatalf("expected job-1; got %v", got)
	}

	none, err := svc.GetJob(ctx, "", "ns", "noselector")
	if err != nil || none == nil {
		t.Fatalf("get job: %v %v", none, err)
	}
	pods, err = svc.PodsForJob(ctx, "", *none, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pods) != 0 {
		t.Fatalf("job without selector must select no pods, got %v", podNames(pods))
	}
}

func TestPodsForDeployment_FilterLabelSelector(t *testing.T) {
	f := NewFake()
	f.Pods = []Pod{
		{Name: "api-1", Namespace: "ns", Labels: map[string]string{"app": "api", "tier": "backend"}},
		{Name: "api-2", Namespace: "ns", Labels: map[string]string{"app": "api"}},
		{Name: "db-1", Namespace: "ns", Labels: map[string]string{"app": "db"}},
		{Name: "other", Namespace: "ns", Labels: map[string]string{"app": "other", "tier": "backend"}},
	}
	svc := NewService(f)
	d := Deployment{Name: "mixed", Namespace: "ns", Selector: "app in (api,db)"}

	cases := []struct {
		selector string
		want     []string
	}{
		{"", []string{"api-1", "api-2", "db-1"}},
		{"tier=backend", []string{"api-1"}},
		{"tier!=backend", []string{"api-2", "db-1"}},
		{"app in (api)", []string{"api-1", "api-2"}},
		{"app notin (api)", []string{"db-1"}},
		{"tier", []string{"api-1"}},
		{"!tier", []string{"api-2", "db-1"}},
		{"app=api,tier", []string{"api-1"}},
	}
	for _, tc := range cases {
		t.Run(tc.selector, func(t *testing.T) {
			sel := tc.selector
			pods, err := svc.PodsForDeployment(context.Background(), testCtx, d, &PodFilter{LabelSelector: &sel})
			if err != nil {
				t.Fatal(err)
			}
			got := podNames(pods)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for _, name := range tc.want {
				if !got[name] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}

	bad := "app in ("
	if _, err := svc.PodsForDeployment(context.Background(), testCtx, d, &PodFilter{LabelSelector: &bad}); err == nil {
		t.Fatal("expected error for invalid labelSelector")
	}
	if _, err := svc.ListPods(context.Background(), testCtx, "ns", &PodFilter{LabelSelector: &bad}); err == nil {
		t.Fatal("ListPods must reject an invalid labelSelector instead of returning nothing")
	}
	if _, err := svc.ListDeployments(context.Background(), testCtx, "ns", &DeploymentFilter{LabelSelector: &bad}); err == nil {
		t.Fatal("ListDeployments must reject an invalid labelSelector")
	}
	if _, err := svc.ListCronJobs(context.Background(), testCtx, "ns", &CronJobFilter{LabelSelector: &bad}); err == nil {
		t.Fatal("ListCronJobs must reject an invalid labelSelector")
	}
}
