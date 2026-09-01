package kube

import (
	"fmt"
	"sort"
	"unicode"
	"unicode/utf8"

	corev1 "k8s.io/api/core/v1"
)

func decodeBytes(b []byte) string {
	if utf8.Valid(b) && isMostlyPrintable(string(b)) {
		return string(b)
	}
	return fmt.Sprintf("<binary %d bytes>", len(b))
}

func isMostlyPrintable(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func collectPodConfigAndSecrets(spec corev1.PodSpec) (cms, secrets []ObjectRef) {
	cmVia := map[string][]string{}
	secVia := map[string][]string{}
	add := func(m map[string][]string, name, via string) {
		if name == "" {
			return
		}
		m[name] = append(m[name], via)
	}

	collectFromContainers := func(containers []corev1.Container) {
		for _, c := range containers {
			for _, e := range c.Env {
				if e.ValueFrom == nil {
					continue
				}
				if r := e.ValueFrom.ConfigMapKeyRef; r != nil {
					add(cmVia, r.Name, "env:"+e.Name)
				}
				if r := e.ValueFrom.SecretKeyRef; r != nil {
					add(secVia, r.Name, "env:"+e.Name)
				}
			}
			for _, e := range c.EnvFrom {
				if e.ConfigMapRef != nil {
					add(cmVia, e.ConfigMapRef.Name, "envFrom")
				}
				if e.SecretRef != nil {
					add(secVia, e.SecretRef.Name, "envFrom")
				}
			}
		}
	}
	collectFromContainers(spec.InitContainers)
	collectFromContainers(spec.Containers)

	for _, v := range spec.Volumes {
		if v.ConfigMap != nil {
			add(cmVia, v.ConfigMap.Name, "volume:"+v.Name)
		}
		if v.Secret != nil {
			add(secVia, v.Secret.SecretName, "volume:"+v.Name)
		}
		if v.Projected != nil {
			for _, src := range v.Projected.Sources {
				if src.ConfigMap != nil {
					add(cmVia, src.ConfigMap.Name, "projected:"+v.Name)
				}
				if src.Secret != nil {
					add(secVia, src.Secret.Name, "projected:"+v.Name)
				}
			}
		}
	}
	for _, ips := range spec.ImagePullSecrets {
		add(secVia, ips.Name, "imagePull")
	}

	return refsFrom(cmVia), refsFrom(secVia)
}

func refsFrom(m map[string][]string) []ObjectRef {
	if len(m) == 0 {
		return nil
	}
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ObjectRef, 0, len(names))
	for _, name := range names {
		out = append(out, ObjectRef{Name: name, Via: uniq(m[name])})
	}
	return out
}

func uniq(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
