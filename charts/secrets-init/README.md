# secrets-init Helm Chart

A Helm Chart the installs DoIT International's [kube-secrets-init](https://github.com/doitintl/kube-secrets-init) application into a Kubernetes cluster.

Please consult the applicatoin's documentation for specifics regarding behavior, etc.

## Prerequisites

Before you can deploy the Helm chart, you'll need to perform two tasks to make sure the application has what it needs within the Kubernetes cluster.

1. Use `scripts/webhook-retrieve-ca-bundle.sh` to obtain the Kubernetes server CA bundle. This will be set in the variable `webhook.caBundle` in this chart and supplied to the application. Set this in the values file however you see fit before installing the Chart.
2. Use `scripts/webhook-create-signed-certificate.sh` to configure a `Secret` within the cluster containing the necessary certificate data for the application itself.

Don't attempt to install the chart before accomplishing both of these tasks.

When running the `webhook-create-signed-certificate.sh` script, be sure to set `namespace` on line 46:

```bash
[ -z ${namespace} ] && namespace=default # OVERRIDE THIS IF CHANGING
```

## Parameters

| Parameter            | Description                                                                | Default                      | Required |
| -------------------- | -------------------------------------------------------------------------- | ---------------------------- | -------- |
| `deployment.enabled` | Enable or disable the `Deployment`.                                        | `true`                       | No       |
| `image.repository`   | Location of the image to pull.                                             | `doitintl/kube-secrets-init` | No       |
| `image.pullPolicy`   | Image pull policy.                                                         | `IfNotPresent`               | No       |
| `image.tag`          | Image tag.                                                                 | `0.2.11`                     | No       |
| `logLevel`           | Log level flag passed to the application.                                  | `debug`                      | No       |
| `provider`           | Can be either `aws` or `google`. Must provide one or the other.            | `nil`                        | Yes      |
| `rbac.create`        | Whether or not to create `ServiceAccount` and `ClusterRole/Binding`.       | `true`                       | No       |
| `replicaCount`       | Replica count for the `Deployment`.                                        | `1`                          | No       |
| `service.port`       | Port to configure for the `Service`                                        | `443`                        | No       |
| `service.targetPort` | Target port on `Pods`.                                                     | `8443`                       | No       |
| `webhook.caBundle`   | Kubernetes API CA bundle to provide to the `MutatingWebhookConfiguration`. | `nil`                        | Yes      |
| `webhook.domain`     | The domain to pass to the `MutatingWebhookConfiguration` configuration.    | `secrets-init.doit-intl.com` | No       |
