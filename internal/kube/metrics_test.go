package kube

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func TestPodResourceLimits_SumsContainers(t *testing.T) {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "prod"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
				{
					Name: "sidecar",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}
	cpu, mem := podResourceLimits(p)
	if cpu == nil || *cpu != "300m" {
		t.Fatalf("cpu limit: got %v want 300m", deref(cpu))
	}
	if mem == nil || *mem != "192Mi" {
		t.Fatalf("memory limit: got %v want 192Mi", deref(mem))
	}
}

func TestPodResourceLimits_Unset(t *testing.T) {
	p := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
	}
	cpu, mem := podResourceLimits(p)
	if cpu != nil || mem != nil {
		t.Fatalf("expected nil limits, got cpu=%v mem=%v", deref(cpu), deref(mem))
	}
}

func TestSumPodMetrics(t *testing.T) {
	m := &metricsv1beta1.PodMetrics{
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name: "app",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
			},
			{
				Name: "sidecar",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("25m"),
					corev1.ResourceMemory: resource.MustParse("16Mi"),
				},
			},
		},
	}
	cpu, mem := sumPodMetrics(m)
	if cpu == nil || *cpu != "75m" {
		t.Fatalf("cpu usage: got %v want 75m", deref(cpu))
	}
	if mem == nil || *mem != "48Mi" {
		t.Fatalf("memory usage: got %v want 48Mi", deref(mem))
	}
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
