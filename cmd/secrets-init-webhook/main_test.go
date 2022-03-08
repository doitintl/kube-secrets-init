package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/doitintl/kube-secrets-init/cmd/secrets-init-webhook/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"
)

type MockRegistry struct {
	Image v1.Config
}

//nolint:lll
func (r *MockRegistry) GetImageConfig(_ context.Context, _ kubernetes.Interface, _ string, _ *corev1.Container, _ *corev1.PodSpec) (*v1.Config, error) {
	return &r.Image, nil
}

//nolint:funlen
func Test_mutatingWebhook_mutateContainers(t *testing.T) {
	type fields struct {
		k8sClient  kubernetes.Interface
		registry   registry.ImageRegistry
		provider   string
		image      string
		pullPolicy string
		volumeName string
		volumePath string
	}
	type args struct {
		containers []corev1.Container
		podSpec    *corev1.PodSpec
		ns         string
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		mutated          bool
		wantErr          bool
		wantedContainers []corev1.Container
	}{
		{
			name: "mutate container with command, no args",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(),
				registry: &MockRegistry{
					Image: v1.Config{},
				},
				provider:   "aws",
				image:      secretsInitImage,
				volumeName: binVolumeName,
				volumePath: binVolumePath,
				pullPolicy: string(corev1.PullIfNotPresent),
			},
			args: args{
				containers: []corev1.Container{
					{
						Name:    "TestContainer",
						Image:   "test-image",
						Command: []string{"echo"},
						Args:    nil,
						Env: []corev1.EnvVar{
							{
								Name:  "topsecret",
								Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/topsecret",
							},
						},
					},
				},
			},
			wantedContainers: []corev1.Container{
				{
					Name:         "TestContainer",
					Image:        "test-image",
					Command:      []string{fmt.Sprintf("%s/secrets-init", binVolumePath)},
					Args:         []string{"--provider=aws", "echo"},
					VolumeMounts: []corev1.VolumeMount{{Name: binVolumeName, MountPath: binVolumePath}},
					Env: []corev1.EnvVar{
						{
							Name:  "topsecret",
							Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/topsecret",
						},
					},
				},
			},
			mutated: true,
		},
		{
			name: "mutate container with command and secret reference",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeSecret("test-ns", "test-secret", map[string][]byte{
						"password": []byte("arn:aws:secretsmanager:us-east-1:123456789012:secret:test/topsecret"),
					}),
				),
				registry: &MockRegistry{
					Image: v1.Config{},
				},
				provider:   "aws",
				image:      secretsInitImage,
				volumeName: binVolumeName,
				volumePath: binVolumePath,
				pullPolicy: string(corev1.PullIfNotPresent),
			},
			args: args{
				ns: "test-ns",
				containers: []corev1.Container{
					{
						Name:    "TestContainer",
						Image:   "test-image",
						Command: []string{"echo"},
						Args:    nil,
						Env: []corev1.EnvVar{
							{
								Name: "topsecret",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										Key:                  "password",
										LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
									},
								},
							},
						},
					},
				},
			},
			wantedContainers: []corev1.Container{
				{
					Name:         "TestContainer",
					Image:        "test-image",
					Command:      []string{fmt.Sprintf("%s/secrets-init", binVolumePath)},
					Args:         []string{"--provider=aws", "echo"},
					VolumeMounts: []corev1.VolumeMount{{Name: binVolumeName, MountPath: binVolumePath}},
					Env: []corev1.EnvVar{
						{
							Name:  "topsecret",
							Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/topsecret",
						},
					},
				},
			},
			mutated: true,
		},
		{
			name: "mutate container with command and args",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(),
				registry: &MockRegistry{
					Image: v1.Config{},
				},
				provider:   "google",
				image:      secretsInitImage,
				volumeName: binVolumeName,
				volumePath: binVolumePath,
				pullPolicy: string(corev1.PullIfNotPresent),
			},
			args: args{
				containers: []corev1.Container{
					{
						Name:    "TestContainer",
						Image:   "test-image",
						Command: []string{"echo"},
						Args:    []string{"test"},
						Env: []corev1.EnvVar{
							{
								Name:  "topsecret",
								Value: "gcp:secretmanager:topsecret",
							},
						},
					},
				},
			},
			wantedContainers: []corev1.Container{
				{
					Name:         "TestContainer",
					Image:        "test-image",
					Command:      []string{fmt.Sprintf("%s/secrets-init", binVolumePath)},
					Args:         []string{"--provider=google", "echo", "test"},
					VolumeMounts: []corev1.VolumeMount{{Name: binVolumeName, MountPath: binVolumePath}},
					Env: []corev1.EnvVar{
						{
							Name:  "topsecret",
							Value: "gcp:secretmanager:topsecret",
						},
					},
				},
			},
			mutated: true,
		},
		{
			name: "mutate container with args, no command",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(),
				registry: &MockRegistry{
					Image: v1.Config{
						Entrypoint: []string{"/bin/zsh"},
					},
				},
				provider:   "aws",
				image:      secretsInitImage,
				volumeName: binVolumeName,
				volumePath: binVolumePath,
				pullPolicy: string(corev1.PullIfNotPresent),
			},
			args: args{
				containers: []corev1.Container{
					{
						Name:  "TestContainer",
						Image: "test-image",
						Args:  []string{"-c", "echo test"},
						Env: []corev1.EnvVar{
							{
								Name:  "topsecret",
								Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/topsecret",
							},
						},
					},
				},
			},
			wantedContainers: []corev1.Container{
				{
					Name:         "TestContainer",
					Image:        "test-image",
					Command:      []string{fmt.Sprintf("%s/secrets-init", binVolumePath)},
					Args:         []string{"--provider=aws", "/bin/zsh", "-c", "echo test"},
					VolumeMounts: []corev1.VolumeMount{{Name: binVolumeName, MountPath: binVolumePath}},
					Env: []corev1.EnvVar{
						{
							Name:  "topsecret",
							Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/topsecret",
						},
					},
				},
			},
			mutated: true,
		},
		{
			name: "mutate container with no container-command, no entrypoint",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(),
				registry: &MockRegistry{
					Image: v1.Config{
						Cmd: []string{"test-cmd"},
					},
				},
				provider:   "aws",
				image:      secretsInitImage,
				volumeName: binVolumeName,
				volumePath: binVolumePath,
				pullPolicy: string(corev1.PullIfNotPresent),
			},
			args: args{
				containers: []corev1.Container{
					{
						Name:  "TestContainer",
						Image: "test-image",
						Env: []corev1.EnvVar{
							{
								Name:  "topsecret",
								Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/topsecret",
							},
						},
					},
				},
			},
			wantedContainers: []corev1.Container{
				{
					Name:         "TestContainer",
					Image:        "test-image",
					Command:      []string{fmt.Sprintf("%s/secrets-init", binVolumePath)},
					Args:         []string{"--provider=aws", "test-cmd"},
					VolumeMounts: []corev1.VolumeMount{{Name: binVolumeName, MountPath: binVolumePath}},
					Env: []corev1.EnvVar{
						{
							Name:  "topsecret",
							Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/topsecret",
						},
					},
				},
			},
			mutated: true,
		},
		{
			name: "not mutate container without secrets with correct prefix",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(),
				registry: &MockRegistry{
					Image: v1.Config{},
				},
				image:      secretsInitImage,
				volumeName: binVolumeName,
				volumePath: binVolumePath,
				pullPolicy: string(corev1.PullIfNotPresent),
			},
			args: args{
				containers: []corev1.Container{
					{
						Name:    "TestContainer",
						Image:   "test-image",
						Command: []string{"/bin/bash"},
						Env: []corev1.EnvVar{
							{
								Name:  "non-secret",
								Value: "hello world",
							},
						},
					},
				},
			},
			wantedContainers: []corev1.Container{
				{
					Name:    "TestContainer",
					Image:   "test-image",
					Command: []string{"/bin/bash"},
					Env: []corev1.EnvVar{
						{
							Name:  "non-secret",
							Value: "hello world",
						},
					},
				},
			},
			mutated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mutatingWebhook{
				k8sClient:  tt.fields.k8sClient,
				registry:   tt.fields.registry,
				provider:   tt.fields.provider,
				image:      tt.fields.image,
				volumeName: tt.fields.volumeName,
				volumePath: tt.fields.volumePath,
				pullPolicy: tt.fields.pullPolicy,
			}
			got, err := mw.mutateContainers(tt.args.containers, tt.args.podSpec, tt.args.ns)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutatingWebhook.mutateContainers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.mutated {
				t.Errorf("mutatingWebhook.mutateContainers() = %v, want %v", got, tt.mutated)
			}
		})
	}
}

// helper function - make K8s Secret
//nolint:unparam
func makeSecret(namespace, name string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
}

// helper function - make K8s ConfigMap
//nolint:unparam
func makeConfigMap(namespace, name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

//nolint:funlen
func Test_mutatingWebhook_lookForEnvFrom(t *testing.T) {
	type fields struct {
		k8sClient  kubernetes.Interface
		provider   string
		image      string
		pullPolicy string
		volumeName string
		volumePath string
	}
	type args struct {
		envFrom []corev1.EnvFromSource
		ns      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []corev1.EnvVar
		wantErr bool
	}{
		{
			name: "get value from secret",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeSecret("test-ns", "test-secret", map[string][]byte{
						"password": []byte("arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret"),
					}),
				),
			},
			args: args{
				ns: "test-ns",
				envFrom: []corev1.EnvFromSource{
					{
						SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"}},
					},
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "password",
					Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
				},
			},
		},
		{
			name: "get value from secret, ignore non-cloud secret",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeSecret("test-ns", "test-secret", map[string][]byte{
						"password": []byte("arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret"),
						"text":     []byte("ignore me"),
					}),
				),
			},
			args: args{
				ns: "test-ns",
				envFrom: []corev1.EnvFromSource{
					{
						SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"}},
					},
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "password",
					Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
				},
			},
		},
		{
			name: "get value from configmap",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeConfigMap("test-ns", "test-secret", map[string]string{
						"password": "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
					}),
				),
			},
			args: args{
				ns: "test-ns",
				envFrom: []corev1.EnvFromSource{
					{
						ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"}},
					},
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "password",
					Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
				},
			},
		},
		{
			name: "get value from configmap, ignore non-cloud configmap",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeConfigMap("test-ns", "test-secret", map[string]string{
						"password": "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
						"text":     "ignore me",
					}),
				),
			},
			args: args{
				ns: "test-ns",
				envFrom: []corev1.EnvFromSource{
					{
						ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"}},
					},
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "password",
					Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
				},
			},
		},
	}
	//nolint:dupl
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mutatingWebhook{
				k8sClient:  tt.fields.k8sClient,
				provider:   tt.fields.provider,
				image:      tt.fields.image,
				pullPolicy: tt.fields.pullPolicy,
				volumeName: tt.fields.volumeName,
				volumePath: tt.fields.volumePath,
			}
			got, err := mw.lookForEnvFrom(tt.args.envFrom, tt.args.ns)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutatingWebhook.lookForEnvFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutatingWebhook.lookForEnvFrom() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:funlen
func Test_mutatingWebhook_lookForValueFrom(t *testing.T) {
	type fields struct {
		k8sClient  kubernetes.Interface
		provider   string
		image      string
		pullPolicy string
		volumeName string
		volumePath string
	}
	type args struct {
		envVar corev1.EnvVar
		ns     string //nolint:gofmt
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *corev1.EnvVar
		wantErr bool
	}{
		{
			name: "get value from secret",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeSecret("test-ns", "test-secret", map[string][]byte{
						"password": []byte("arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret"),
					}),
				),
			},
			args: args{
				ns: "test-ns",
				envVar: corev1.EnvVar{
					Name: "PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							Key:                  "password",
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						},
					},
				},
			},
			want: &corev1.EnvVar{
				Name:  "PASSWORD",
				Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
			},
		},
		{
			name: "get value from secret, ignore non-cloud secret",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeSecret("test-ns", "test-secret", map[string][]byte{
						"password": []byte("arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret"),
						"text":     []byte("ignore me"),
					}),
				),
			},
			args: args{
				ns: "test-ns",
				envVar: corev1.EnvVar{
					Name: "PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							Key:                  "password",
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						},
					},
				},
			},
			want: &corev1.EnvVar{

				Name:  "PASSWORD",
				Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
			},
		},
		{
			name: "get value from configmap",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeConfigMap("test-ns", "test-secret", map[string]string{
						"password": "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
					}),
				),
			},
			args: args{
				ns: "test-ns",
				envVar: corev1.EnvVar{
					Name: "PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							Key:                  "password",
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						},
					},
				},
			},
			want: &corev1.EnvVar{

				Name:  "PASSWORD",
				Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
			},
		},
		{
			name: "get value from configmap, ignore non-cloud configmap",
			fields: fields{
				k8sClient: fake.NewSimpleClientset(
					makeConfigMap("test-ns", "test-secret", map[string]string{
						"password": "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
						"text":     "ignore me",
					}),
				),
			},
			args: args{
				ns: "test-ns",
				envVar: corev1.EnvVar{
					Name: "PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							Key:                  "password",
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						},
					},
				},
			},
			want: &corev1.EnvVar{

				Name:  "PASSWORD",
				Value: "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/secret",
			},
		},
	}
	//nolint:dupl
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mutatingWebhook{
				k8sClient:  tt.fields.k8sClient,
				provider:   tt.fields.provider,
				image:      tt.fields.image,
				pullPolicy: tt.fields.pullPolicy,
				volumeName: tt.fields.volumeName,
				volumePath: tt.fields.volumePath,
			}
			got, err := mw.lookForValueFrom(tt.args.envVar, tt.args.ns)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutatingWebhook.lookForEnvFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutatingWebhook.lookForEnvFrom() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isNewImage(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  bool
	}{
		{
			name:  "new version",
			image: "test:1.0.0",
			want:  true,
		},
		{
			name:  "new version exact match",
			image: "test:0.4.0",
			want:  true,
		},
		{
			name:  "assumed latest",
			image: "test",
			want:  true,
		},
		{
			name:  "explicit latest",
			image: "test:latest",
			want:  true,
		},
		{
			name:  "unexpected version",
			image: "test:wtf",
			want:  false,
		},
		{
			name:  "old version",
			image: "test:0.1.0",
			want:  false,
		},
		{
			name:  "old version with prefix",
			image: "test:v0.2.9",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewImage(tt.image); got != tt.want {
				t.Errorf("isNewImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
