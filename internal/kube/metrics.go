package kube

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func (c *Client) attachListMetrics(ctx context.Context, namespace, labelSelector string, pods []Pod) {
	if c.metrics == nil || len(pods) == 0 {
		return
	}
	list, err := c.metrics.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return
	}
	byName := make(map[string]metricsv1beta1.PodMetrics, len(list.Items))
	for i := range list.Items {
		byName[list.Items[i].Name] = list.Items[i]
	}
	for i := range pods {
		if m, ok := byName[pods[i].Name]; ok {
			cpu, mem := sumPodMetrics(&m)
			pods[i].CPUUsage = cpu
			pods[i].MemoryUsage = mem
		}
	}
}

func (c *Client) attachGetMetrics(ctx context.Context, namespace, name string, pod *Pod) {
	if c.metrics == nil || pod == nil {
		return
	}
	m, err := c.metrics.MetricsV1beta1().PodMetricses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return
	}
	cpu, mem := sumPodMetrics(m)
	pod.CPUUsage = cpu
	pod.MemoryUsage = mem
}

func sumPodMetrics(m *metricsv1beta1.PodMetrics) (cpu, mem *string) {
	if m == nil {
		return nil, nil
	}
	var cpuQ, memQ resource.Quantity
	var hasCPU, hasMem bool
	for _, c := range m.Containers {
		if q, ok := c.Usage[corev1.ResourceCPU]; ok {
			cpuQ.Add(q)
			hasCPU = true
		}
		if q, ok := c.Usage[corev1.ResourceMemory]; ok {
			memQ.Add(q)
			hasMem = true
		}
	}
	if hasCPU {
		s := cpuQ.String()
		cpu = &s
	}
	if hasMem {
		s := memQ.String()
		mem = &s
	}
	return cpu, mem
}

func podResourceLimits(p *corev1.Pod) (cpu, mem *string) {
	var cpuQ, memQ resource.Quantity
	var hasCPU, hasMem bool
	for _, c := range p.Spec.Containers {
		if q, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
			cpuQ.Add(q)
			hasCPU = true
		}
		if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
			memQ.Add(q)
			hasMem = true
		}
	}
	if hasCPU {
		s := cpuQ.String()
		cpu = &s
	}
	if hasMem {
		s := memQ.String()
		mem = &s
	}
	return cpu, mem
}
