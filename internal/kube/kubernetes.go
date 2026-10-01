package kube

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/yaml"
)

// Client is a ClusterReader backed by client-go for a single kubeconfig context.
type Client struct {
	cs      kubernetes.Interface
	dyn     dynamic.Interface       // optional; nil when the rest config cannot build a dynamic client
	metrics metricsclient.Interface // optional; nil when metrics-server is unused/unavailable
	context string
}

// Context returns the kubeconfig context name this client was built for.
func (c *Client) Context() string {
	return c.context
}

// NewClient builds a Kubernetes client from kubeconfig path and optional context.
// Prefer OpenRegistry when you also need to list contexts or hold one client per context.
// Empty kubeconfig uses the default loading rules (KUBECONFIG, ~/.kube/config).
func NewClient(kubeconfig, contextName string) (*Client, error) {
	reg, err := OpenRegistry(kubeconfig, contextName)
	if err != nil {
		return nil, err
	}
	return reg.DefaultClient()
}

// NewClientFromInterface is useful for tests with fake clientsets.
func NewClientFromInterface(cs kubernetes.Interface) *Client {
	return &Client{cs: cs}
}

// expandPath resolves leading ~/ to the user home directory.
// Go and client-go do not expand ~; that is a shell feature only.
func expandPath(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	if p == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func (c *Client) ListNamespaces(ctx context.Context) ([]Namespace, error) {
	list, err := c.cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]Namespace, 0, len(list.Items))
	for _, item := range list.Items {
		out = append(out, Namespace{Name: item.Name})
	}
	return out, nil
}

func (c *Client) GetNamespace(ctx context.Context, name string) (*Namespace, error) {
	ns, err := c.cs.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &Namespace{Name: ns.Name}, nil
}

func (c *Client) ListDeployments(ctx context.Context, namespace, labelSelector string) ([]Deployment, error) {
	list, err := c.cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Deployment, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapDeployment(&list.Items[i]))
	}
	return out, nil
}

func (c *Client) GetDeployment(ctx context.Context, namespace, name string) (*Deployment, error) {
	d, err := c.cs.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	mapped := mapDeployment(d)
	return &mapped, nil
}

func (c *Client) ListCronJobs(ctx context.Context, namespace, labelSelector string) ([]CronJob, error) {
	list, err := c.cs.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	out := make([]CronJob, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapCronJob(&list.Items[i]))
	}
	return out, nil
}

func (c *Client) GetCronJob(ctx context.Context, namespace, name string) (*CronJob, error) {
	cj, err := c.cs.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	mapped := mapCronJob(cj)
	return &mapped, nil
}

func (c *Client) ListJobs(ctx context.Context, namespace, labelSelector string) ([]Job, error) {
	list, err := c.cs.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Job, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapJob(&list.Items[i]))
	}
	return out, nil
}

func (c *Client) GetJob(ctx context.Context, namespace, name string) (*Job, error) {
	j, err := c.cs.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	mapped := mapJob(j)
	return &mapped, nil
}

func (c *Client) ListPods(ctx context.Context, namespace, labelSelector string) ([]Pod, error) {
	list, err := c.cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Pod, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapPod(&list.Items[i]))
	}
	c.attachListMetrics(ctx, namespace, labelSelector, out)
	return out, nil
}

func (c *Client) GetPod(ctx context.Context, namespace, name string) (*Pod, error) {
	p, err := c.cs.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	mapped := mapPod(p)
	c.attachGetMetrics(ctx, namespace, name, &mapped)
	return &mapped, nil
}

func (c *Client) ListConfigMaps(ctx context.Context, namespace string) ([]ConfigMap, error) {
	list, err := c.cs.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]ConfigMap, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapConfigMap(&list.Items[i]))
	}
	return out, nil
}

func (c *Client) GetConfigMap(ctx context.Context, namespace, name string) (*ConfigMap, error) {
	cm, err := c.cs.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	mapped := mapConfigMap(cm)
	return &mapped, nil
}

func (c *Client) ListSecrets(ctx context.Context, namespace string) ([]Secret, error) {
	list, err := c.cs.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]Secret, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapSecret(&list.Items[i]))
	}
	return out, nil
}

func (c *Client) GetSecret(ctx context.Context, namespace, name string) (*Secret, error) {
	sec, err := c.cs.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	mapped := mapSecret(sec)
	return &mapped, nil
}

func (c *Client) ConfigMapYAML(ctx context.Context, namespace, name string) (string, error) {
	cm, err := c.cs.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return toYAML(cm)
}

func (c *Client) SecretYAML(ctx context.Context, namespace, name string) (string, error) {
	sec, err := c.cs.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return toYAML(sec)
}

func mapConfigMap(cm *corev1.ConfigMap) ConfigMap {
	data := map[string]string{}
	for k, v := range cm.Data {
		data[k] = v
	}
	for k, v := range cm.BinaryData {
		data[k] = decodeBytes(v)
	}
	return ConfigMap{
		Name:      cm.Name,
		Namespace: cm.Namespace,
		Data:      data,
	}
}

func mapSecret(s *corev1.Secret) Secret {
	data := map[string]string{}
	for k, v := range s.Data {
		data[k] = decodeBytes(v)
	}
	return Secret{
		Name:      s.Name,
		Namespace: s.Namespace,
		Type:      string(s.Type),
		Data:      data,
	}
}

func (c *Client) DeploymentYAML(ctx context.Context, namespace, name string) (string, error) {
	d, err := c.cs.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return toYAML(d)
}

func (c *Client) CronJobYAML(ctx context.Context, namespace, name string) (string, error) {
	cj, err := c.cs.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return toYAML(cj)
}

func (c *Client) JobYAML(ctx context.Context, namespace, name string) (string, error) {
	j, err := c.cs.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return toYAML(j)
}

func (c *Client) PodYAML(ctx context.Context, namespace, name string) (string, error) {
	p, err := c.cs.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return toYAML(p)
}

func (c *Client) StreamPodLogs(ctx context.Context, opts LogOptions) (io.ReadCloser, error) {
	req := c.cs.CoreV1().Pods(opts.Namespace).GetLogs(opts.Pod, &corev1.PodLogOptions{
		Container:  opts.Container,
		Follow:     opts.Follow,
		TailLines:  opts.TailLines,
		Timestamps: true,
	})
	return req.Stream(ctx)
}

func (c *Client) ListPodEvents(ctx context.Context, namespace, podName string) ([]Event, error) {
	list, err := c.cs.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod,involvedObject.namespace=%s", podName, namespace),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapEvent(&list.Items[i]))
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := eventSortTime(out[i]), eventSortTime(out[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func mapEvent(e *corev1.Event) Event {
	first := metaTimePtr(&e.FirstTimestamp)
	last := metaTimePtr(&e.LastTimestamp)
	if last == nil {
		last = microTimePtr(e.EventTime)
	}
	if first == nil {
		first = last
	}
	return Event{
		Name:         e.Name,
		Namespace:    e.Namespace,
		InvolvedName: e.InvolvedObject.Name,
		Type:         e.Type,
		Reason:       e.Reason,
		Message:      e.Message,
		Count:        eventCount(e),
		FirstSeen:    first,
		LastSeen:     last,
		Source:       eventSource(e),
	}
}

func eventCount(e *corev1.Event) int32 {
	if e.Series != nil && e.Series.Count > 0 {
		return e.Series.Count
	}
	if e.Count > 0 {
		return e.Count
	}
	return 1
}

func eventSource(e *corev1.Event) string {
	src := e.Source.Component
	if e.Source.Host != "" {
		if src != "" {
			src += "/" + e.Source.Host
		} else {
			src = e.Source.Host
		}
	}
	if src == "" {
		src = e.ReportingController
	}
	return src
}

func eventSortTime(e Event) time.Time {
	if e.LastSeen != nil {
		return *e.LastSeen
	}
	if e.FirstSeen != nil {
		return *e.FirstSeen
	}
	return time.Time{}
}

func microTimePtr(t metav1.MicroTime) *time.Time {
	if t.IsZero() {
		return nil
	}
	tt := t.Time
	return &tt
}

func mapDeployment(d *appsv1.Deployment) Deployment {
	var replicas int32
	if d.Spec.Replicas != nil {
		replicas = *d.Spec.Replicas
	}
	selector := map[string]string{}
	if d.Spec.Selector != nil {
		for k, v := range d.Spec.Selector.MatchLabels {
			selector[k] = v
		}
	}
	cms, secrets := collectPodConfigAndSecrets(d.Spec.Template.Spec)
	return Deployment{
		Name:              d.Name,
		Namespace:         d.Namespace,
		Replicas:          replicas,
		ReadyReplicas:     d.Status.ReadyReplicas,
		AvailableReplicas: d.Status.AvailableReplicas,
		Status:            deploymentStatus(d),
		Labels:            copyMap(d.Labels),
		Selector:          selector,
		ConfigMapRefs:     cms,
		SecretRefs:        secrets,
	}
}

func deploymentStatus(d *appsv1.Deployment) string {
	return fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, ptrInt32(d.Spec.Replicas))
}

func mapCronJob(cj *batchv1.CronJob) CronJob {
	suspend := false
	if cj.Spec.Suspend != nil {
		suspend = *cj.Spec.Suspend
	}
	tz := ""
	if cj.Spec.TimeZone != nil {
		tz = *cj.Spec.TimeZone
	}
	cms, secrets := collectPodConfigAndSecrets(cj.Spec.JobTemplate.Spec.Template.Spec)
	policy := string(cj.Spec.ConcurrencyPolicy)
	if policy == "" {
		policy = "Allow"
	}
	return CronJob{
		Name:               cj.Name,
		Namespace:          cj.Namespace,
		Schedule:           cj.Spec.Schedule,
		TimeZone:           tz,
		Suspend:            suspend,
		ConcurrencyPolicy:  policy,
		LastScheduleTime:   metaTimePtr(cj.Status.LastScheduleTime),
		LastSuccessfulTime: metaTimePtr(cj.Status.LastSuccessfulTime),
		Active:             int32(len(cj.Status.Active)),
		Status:             cronJobStatus(cj),
		Labels:             copyMap(cj.Labels),
		ConfigMapRefs:      cms,
		SecretRefs:         secrets,
	}
}

func cronJobStatus(cj *batchv1.CronJob) string {
	if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
		return "Suspended"
	}
	if n := len(cj.Status.Active); n > 0 {
		return fmt.Sprintf("Active %d", n)
	}
	return "Idle"
}

func mapJob(j *batchv1.Job) Job {
	completions := int32(1)
	if j.Spec.Completions != nil {
		completions = *j.Spec.Completions
	}
	selector := map[string]string{}
	if j.Spec.Selector != nil {
		for k, v := range j.Spec.Selector.MatchLabels {
			selector[k] = v
		}
	}
	ownerKind, ownerName := controllerOwner(j.OwnerReferences)
	cms, secrets := collectPodConfigAndSecrets(j.Spec.Template.Spec)
	return Job{
		Name:           j.Name,
		Namespace:      j.Namespace,
		Completions:    completions,
		Succeeded:      j.Status.Succeeded,
		Failed:         j.Status.Failed,
		Active:         j.Status.Active,
		Status:         jobStatus(j),
		StartTime:      metaTimePtr(j.Status.StartTime),
		CompletionTime: metaTimePtr(j.Status.CompletionTime),
		Labels:         copyMap(j.Labels),
		Selector:       selector,
		OwnerKind:      ownerKind,
		OwnerName:      ownerName,
		ConfigMapRefs:  cms,
		SecretRefs:     secrets,
	}
}

func jobStatus(j *batchv1.Job) string {
	for _, c := range j.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return "Complete"
		}
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return "Failed"
		}
	}
	if j.Status.Active > 0 {
		return "Running"
	}
	return "Pending"
}

func controllerOwner(owners []metav1.OwnerReference) (kind, name string) {
	for _, o := range owners {
		if o.Controller != nil && *o.Controller {
			return o.Kind, o.Name
		}
	}
	if len(owners) > 0 {
		return owners[0].Kind, owners[0].Name
	}
	return "", ""
}

func metaTimePtr(t *metav1.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	tt := t.Time
	return &tt
}

func mapPod(p *corev1.Pod) Pod {
	var restarts int32
	var lastRestart *time.Time
	var lastRestartReason string
	containers := make([]Container, 0, len(p.Status.ContainerStatuses))
	for _, cs := range p.Status.ContainerStatuses {
		restarts += cs.RestartCount
		lastTerm := mapTerminated(cs.LastTerminationState.Terminated)
		if lastTerm != nil {
			if lastTerm.FinishedAt != nil && (lastRestart == nil || lastTerm.FinishedAt.After(*lastRestart)) {
				lastRestart = lastTerm.FinishedAt
				lastRestartReason = lastTerm.Reason
			} else if lastRestart == nil && lastTerm.Reason != "" {
				lastRestartReason = lastTerm.Reason
			}
		}
		state, reason, message := containerStateParts(cs.State)
		containers = append(containers, Container{
			Name:           cs.Name,
			Image:          cs.Image,
			Ready:          cs.Ready,
			RestartCount:   cs.RestartCount,
			State:          state,
			Reason:         reason,
			Message:        message,
			LastTerminated: lastTerm,
		})
	}
	// Include containers that have no status yet (from spec).
	if len(containers) == 0 {
		for _, c := range p.Spec.Containers {
			containers = append(containers, Container{
				Name:  c.Name,
				Image: c.Image,
				State: "Unknown",
			})
		}
	}
	cpuLimit, memLimit := podResourceLimits(p)
	var createdAt *time.Time
	if !p.CreationTimestamp.IsZero() {
		t := p.CreationTimestamp.Time
		createdAt = &t
	}
	return Pod{
		Name:              p.Name,
		Namespace:         p.Namespace,
		Phase:             string(p.Status.Phase),
		Ready:             isPodReady(p),
		Restarts:          restarts,
		LastRestartAt:     lastRestart,
		LastRestartReason: lastRestartReason,
		CreatedAt:         createdAt,
		NodeName:          p.Spec.NodeName,
		Labels:            copyMap(p.Labels),
		Containers:        containers,
		CPULimit:          cpuLimit,
		MemoryLimit:       memLimit,
	}
}

func isPodReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func containerStateParts(s corev1.ContainerState) (state, reason, message string) {
	switch {
	case s.Running != nil:
		return "Running", "", ""
	case s.Waiting != nil:
		return "Waiting", s.Waiting.Reason, s.Waiting.Message
	case s.Terminated != nil:
		return "Terminated", s.Terminated.Reason, s.Terminated.Message
	default:
		return "Unknown", "", ""
	}
}

func mapTerminated(t *corev1.ContainerStateTerminated) *ContainerTermination {
	if t == nil {
		return nil
	}
	return &ContainerTermination{
		Reason:     t.Reason,
		ExitCode:   t.ExitCode,
		Signal:     t.Signal,
		Message:    t.Message,
		StartedAt:  metaTimePtr(&t.StartedAt),
		FinishedAt: metaTimePtr(&t.FinishedAt),
	}
}

func toYAML(obj runtime.Object) (string, error) {
	// Drop managed fields noise for a cleaner viewer experience.
	if m, ok := obj.(metav1.Object); ok {
		m.SetManagedFields(nil)
	}
	b, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func ptrInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}
