package kube

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestCollectPodConfigAndSecrets(t *testing.T) {
	opt := true
	spec := corev1.PodSpec{
		ImagePullSecrets: []corev1.LocalObjectReference{{Name: "regcred"}},
		Containers: []corev1.Container{{
			Name: "app",
			Env: []corev1.EnvVar{
				{
					Name: "DB_URL",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "db"},
							Key:                  "url",
						},
					},
				},
			},
			EnvFrom: []corev1.EnvFromSource{{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
				},
			}},
		}},
		Volumes: []corev1.Volume{
			{
				Name: "tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{SecretName: "tls-cert"},
				},
			},
			{
				Name: "cfg",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
						Optional:             &opt,
					},
				},
			},
		},
	}
	cms, secrets := collectPodConfigAndSecrets(spec)
	if len(cms) != 1 || cms[0].Name != "app-config" {
		t.Fatalf("configmaps: %#v", cms)
	}
	if !reflect.DeepEqual(cms[0].Via, []string{"envFrom", "volume:cfg"}) {
		t.Fatalf("configmap via: %#v", cms[0].Via)
	}
	gotSecrets := map[string][]string{}
	for _, s := range secrets {
		gotSecrets[s.Name] = s.Via
	}
	if !reflect.DeepEqual(gotSecrets["db"], []string{"env:DB_URL"}) {
		t.Fatalf("db secret via: %#v", gotSecrets["db"])
	}
	if !reflect.DeepEqual(gotSecrets["tls-cert"], []string{"volume:tls"}) {
		t.Fatalf("tls secret via: %#v", gotSecrets["tls-cert"])
	}
	if !reflect.DeepEqual(gotSecrets["regcred"], []string{"imagePull"}) {
		t.Fatalf("pull secret via: %#v", gotSecrets["regcred"])
	}
}
