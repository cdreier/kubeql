package kube

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMapPod_LastTerminatedReason(t *testing.T) {
	finished := metav1.NewTime(time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC))
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: "prod"},
		Spec:       corev1.PodSpec{NodeName: "n1"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					Image:        "app:1",
					Ready:        false,
					RestartCount: 3,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "CrashLoopBackOff",
							Message: "back-off 5m0s restarting failed container=app",
						},
					},
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:     "OOMKilled",
							ExitCode:   137,
							Message:    "killed by oom",
							FinishedAt: finished,
						},
					},
				},
			},
		},
	}

	got := mapPod(p)
	if got.Restarts != 3 {
		t.Fatalf("restarts: got %d want 3", got.Restarts)
	}
	if got.LastRestartReason != "OOMKilled" {
		t.Fatalf("lastRestartReason: got %q want OOMKilled", got.LastRestartReason)
	}
	if got.LastRestartAt == nil || !got.LastRestartAt.Equal(finished.Time) {
		t.Fatalf("lastRestartAt: got %v want %v", got.LastRestartAt, finished.Time)
	}
	if len(got.Containers) != 1 {
		t.Fatalf("containers: got %d want 1", len(got.Containers))
	}
	c := got.Containers[0]
	if c.State != "Waiting" || c.Reason != "CrashLoopBackOff" {
		t.Fatalf("state/reason: got %s/%s", c.State, c.Reason)
	}
	if c.LastTerminated == nil || c.LastTerminated.Reason != "OOMKilled" || c.LastTerminated.ExitCode != 137 {
		t.Fatalf("lastTerminated: %#v", c.LastTerminated)
	}
}

func TestMapEvent_SourceAndCount(t *testing.T) {
	ts := metav1.NewTime(time.Date(2026, 3, 18, 12, 1, 0, 0, time.UTC))
	e := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "api-1.18", Namespace: "prod"},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Name:      "api-1",
			Namespace: "prod",
		},
		Type:           "Warning",
		Reason:         "BackOff",
		Message:        "Back-off restarting failed container app",
		Count:          12,
		FirstTimestamp: ts,
		LastTimestamp:  ts,
		Source:         corev1.EventSource{Component: "kubelet", Host: "n1"},
	}
	got := mapEvent(e)
	if got.Reason != "BackOff" || got.Type != "Warning" || got.Count != 12 {
		t.Fatalf("event: %#v", got)
	}
	if got.Source != "kubelet/n1" {
		t.Fatalf("source: got %q", got.Source)
	}
	if got.InvolvedName != "api-1" {
		t.Fatalf("involved: %q", got.InvolvedName)
	}
}

func TestClientListPodEvents(t *testing.T) {
	cs := fake.NewSimpleClientset(&corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "api-1.1", Namespace: "prod"},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Name:      "api-1",
			Namespace: "prod",
		},
		Type:    "Warning",
		Reason:  "Unhealthy",
		Message: "Liveness probe failed",
		Count:   2,
	})
	c := NewClientFromInterface(cs)
	got, err := c.ListPodEvents(context.Background(), "prod", "api-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Reason != "Unhealthy" {
		t.Fatalf("got %#v", got)
	}
}
