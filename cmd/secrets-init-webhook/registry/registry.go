// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// ImageRegistry is a docker registry
type ImageRegistry interface {
	GetImageConfig(
		ctx context.Context,
		client kubernetes.Interface,
		namespace string,
		container *corev1.Container,
		podSpec *corev1.PodSpec,
	) (*v1.Config, error)
}

// Registry impl
type Registry struct {
	imageCache                      ImageCache
	registrySkipVerify              bool
	dockerConfigJSONKey             string
	defaultImagePullSecret          string
	defaultImagePullSecretNamespace string
}

// NewRegistry creates and initializes registry
func NewRegistry(skipVerify bool, configJSONKey, imagePullSecret, imagePullSecretNamespace string) ImageRegistry {
	return &Registry{
		imageCache:                      NewInMemoryImageCache(),
		registrySkipVerify:              skipVerify,
		dockerConfigJSONKey:             configJSONKey,
		defaultImagePullSecret:          imagePullSecret,
		defaultImagePullSecretNamespace: imagePullSecretNamespace,
	}
}

//nolint:lll
// GetImageConfig returns entrypoint and command of container
func (r *Registry) GetImageConfig(ctx context.Context, client kubernetes.Interface, namespace string, container *corev1.Container, podSpec *corev1.PodSpec) (*v1.Config, error) {
	if imageConfig := r.imageCache.Get(container.Image); imageConfig != nil {
		return imageConfig, nil
	}

	containerInfo := containerInfo{
		Namespace:          namespace,
		ServiceAccountName: podSpec.ServiceAccountName,
		Image:              container.Image,
	}

	for _, imagePullSecret := range podSpec.ImagePullSecrets {
		containerInfo.ImagePullSecrets = append(containerInfo.ImagePullSecrets, imagePullSecret.Name)
	}

	if len(containerInfo.ImagePullSecrets) == 0 &&
		r.defaultImagePullSecretNamespace != "" && r.defaultImagePullSecret != "" {
		containerInfo.Namespace = r.defaultImagePullSecretNamespace
		containerInfo.ImagePullSecrets = []string{r.defaultImagePullSecret}

		// TODO: check service account for image pull secrets
		containerInfo.ServiceAccountName = ""
	}

	imageConfig, err := getImageConfig(ctx, client, containerInfo, r.registrySkipVerify)
	if imageConfig != nil {
		r.imageCache.Put(container.Image, imageConfig)
	}

	return imageConfig, err
}

// getImageConfig download image blob from registry
func getImageConfig(ctx context.Context, client kubernetes.Interface, container containerInfo, registrySkipVerify bool) (*v1.Config, error) {
	chainOpts := k8schain.Options{
		Namespace:          container.Namespace,
		ServiceAccountName: container.ServiceAccountName,
		ImagePullSecrets:   container.ImagePullSecrets,
	}

	authChain, err := k8schain.New(
		ctx,
		client,
		chainOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8schain authentication: %w", err)
	}

	options := []remote.Option{
		remote.WithAuthFromKeychain(authChain),
	}

	if registrySkipVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
		}
		options = append(options, remote.WithTransport(tr))
	}

	ref, err := name.ParseReference(container.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference: %w", err)
	}

	descriptor, err := remote.Get(ref, options...)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch image descriptor: %w", err)
	}

	image, err := descriptor.Image()
	if err != nil {
		return nil, fmt.Errorf("cannot convert image descriptor to v1.Image: %w", err)
	}

	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("cannot extract config file of image: %w", err)
	}

	return &configFile.Config, nil
}

// containerInfo keeps information retrieved from POD based container definition
type containerInfo struct {
	Namespace          string
	ImagePullSecrets   []string
	ServiceAccountName string
	Image              string
}
