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

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	kauth "github.com/google/go-containerregistry/pkg/authn/kubernetes"
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
	}

	for _, imagePullSecret := range podSpec.ImagePullSecrets {
		containerInfo.ImagePullSecrets = append(containerInfo.ImagePullSecrets, imagePullSecret.Name)
	}

	keychain, err := r.getKeychain(ctx, client, containerInfo)
	if err != nil {
		return nil, err
	}

	imageConfig, err := getImageConfig(ctx, keychain, container.Image, r.registrySkipVerify)
	if imageConfig != nil {
		r.imageCache.Put(container.Image, imageConfig)
	}

	return imageConfig, err
}

func (r *Registry) getKeychain(ctx context.Context, client kubernetes.Interface, container containerInfo) (authn.Keychain, error) {
	opts := k8schain.Options{
		Namespace:          container.Namespace,
		ServiceAccountName: container.ServiceAccountName,
		ImagePullSecrets:   container.ImagePullSecrets,
	}

	var keychain authn.Keychain
	var err error

	// TODO: allow reorganizing auth chains
	// currently cloud keychains have precedence
	keychain, err = k8schain.New(ctx, client, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8schain authentication: %w", err)
	}

	if r.defaultImagePullSecretNamespace != "" && r.defaultImagePullSecret != "" {
		opts := kauth.Options{
			Namespace:        r.defaultImagePullSecretNamespace,
			ImagePullSecrets: []string{r.defaultImagePullSecret},
		}

		defaultKeychain, err := kauth.New(ctx, client, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create default chain authentication: %w", err)
		}

		keychain = authn.NewMultiKeychain(keychain, defaultKeychain)
	}

	return keychain, nil
}

// getImageConfig download image blob from registry
func getImageConfig(ctx context.Context, keychain authn.Keychain, imageRef string, registrySkipVerify bool) (*v1.Config, error) {
	options := []remote.Option{
		remote.WithAuthFromKeychain(keychain),
		remote.WithContext(ctx),
	}

	if registrySkipVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
		}
		options = append(options, remote.WithTransport(tr))
	}

	ref, err := name.ParseReference(imageRef)
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
}
