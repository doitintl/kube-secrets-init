module github.com/doitintl/kube-secrets-init

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/google/go-containerregistry v0.8.1-0.20220110151055-a61fd0a8e2bb
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20220219142810-1571d7fdc46e
	github.com/google/go-containerregistry/pkg/authn/kubernetes v0.0.0-20220128225446-c63684ed5f15
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/slok/kubewebhook/v2 v2.0.0
	github.com/urfave/cli v1.22.4
	k8s.io/api v0.23.3
	k8s.io/apimachinery v0.23.3
	k8s.io/client-go v0.23.3
	sigs.k8s.io/controller-runtime v0.11.1
)

replace github.com/doitintl/kube-secrets-init/cmd/secrets-init-webhook/registry => ./cmd/secrets-init-webhook/registry
