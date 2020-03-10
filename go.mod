module github.com/doitintl/kube-secrets-init

go 1.13

require (
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-cmp v0.3.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/opencontainers/image-spec v1.0.1
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/prometheus/client_golang v1.0.0
	github.com/sirupsen/logrus v1.4.2
	github.com/slok/kubewebhook v0.3.0
	github.com/urfave/cli v1.22.1
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.3.0
)

replace github.com/doitintl/kube-secrets-init/cmd/secrets-init-webhook/registry => ./cmd/secrets-init-webhook/registry

replace k8s.io/api => k8s.io/api v0.0.0-20181213150558-05914d821849

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93

replace k8s.io/client-go => k8s.io/client-go v10.0.0+incompatible
