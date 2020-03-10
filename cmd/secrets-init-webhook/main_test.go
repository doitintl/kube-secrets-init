package main

import (
	"fmt"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/doitintl/kube-secrets-init/cmd/secrets-init-webhook/registry"
	imagev1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type MockRegistry struct {
	Image imagev1.ImageConfig
}

//nolint:lll
func (r *MockRegistry) GetImageConfig(_ kubernetes.Interface, _ string, _ *corev1.Container, _ *corev1.PodSpec) (*imagev1.ImageConfig, error) {
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
					Image: imagev1.ImageConfig{},
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
					Command:      []string{fmt.Sprintf("%s/secrets-init --provider=%s", binVolumePath, "aws")},
					Args:         []string{"echo"},
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
					Image: imagev1.ImageConfig{},
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
					Command:      []string{fmt.Sprintf("%s/secrets-init --provider=%s", binVolumePath, "google")},
					Args:         []string{"echo", "test"},
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
					Image: imagev1.ImageConfig{
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
					Command:      []string{fmt.Sprintf("%s/secrets-init --provider=%s", binVolumePath, "aws")},
					Args:         []string{"/bin/zsh", "-c", "echo test"},
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
					Image: imagev1.ImageConfig{
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
					Command:      []string{fmt.Sprintf("%s/secrets-init --provider=%s", binVolumePath, "aws")},
					Args:         []string{"test-cmd"},
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
					Image: imagev1.ImageConfig{},
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
			if !cmp.Equal(tt.args.containers, tt.wantedContainers) {
				t.Errorf("mutatingWebhook.mutateContainers() = diff %v", cmp.Diff(tt.args.containers, tt.wantedContainers))
			}
		})
	}
}
