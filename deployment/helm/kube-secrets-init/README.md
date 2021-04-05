# kube-secrets-init

![version: 0.4.0](https://img.shields.io/badge/version-0.4.0-informational?style=flat-square) ![type: application](https://img.shields.io/badge/type-application-informational?style=flat-square) ![app version: 0.2.14](https://img.shields.io/badge/app%20version-0.2.14-informational?style=flat-square) ![kube version: >=1.16.0-0](https://img.shields.io/badge/kube%20version->=1.16.0--0-informational?style=flat-square) [![artifact hub](https://img.shields.io/badge/artifact%20hub-kube--secrets--init-informational?style=flat-square)](https://artifacthub.io/packages/helm/sagikazarmark/kube-secrets-init)

kube-secrets-init is a Kubernetes mutating admission webhook, that mutates any Pod that is using specially prefixed environment variables, directly or from Kubernetes as Secret or ConfigMap.

**Homepage:** <https://github.com/doitintl/kube-secrets-init>

## TL;DR;

```bash
helm repo add skm https://charts.sagikazarmark.dev
helm install --generate-name --wait skm/kube-secrets-init
```

## Configuration

### Certificate

Webhooks require CA and TLS certificates to secure communication between the API server and the webhook.
The chart offers the following options for certificate provisioning:

#### Generate (default)

The default option is to let Helm generate the CA and TLS certificates during installation.

This will renew the certificates on each deployment.

```yaml
certificate:
  generate: true
```

#### Using cert-manager

If have [cert-manager](https://cert-manager.io/) installed in your cluster, you can use it to automatically provision and inject certificates.

```yaml
certificate:
  useCertManager: true
  generate: false
```

## About GKE Private Clusters

When Google configure the control plane for private clusters, they automatically configure VPC peering between your Kubernetes cluster’s network in a separate Google managed project.

The auto-generated rules **only** open ports 10250 and 443 between masters and nodes. This means that to use the webhook component with a GKE private cluster, you must configure an additional firewall rule to allow your masters CIDR to access your webhook pod using the port 8443.

You can read more information on how to add firewall rules for the GKE control plane nodes in the [GKE docs](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules).

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| replicaCount | int | `1` | Number of Pods to launch. |
| image.repository | string | `"doitintl/kube-secrets-init"` | Repository to pull the container image from. |
| image.pullPolicy | string | `"IfNotPresent"` | Image [pull policy](https://kubernetes.io/docs/concepts/containers/images/#updating-images) |
| image.tag | string | `""` | Overrides the image tag (default is the chart appVersion). |
| imagePullSecrets | list | `[]` | Image [pull secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret) |
| nameOverride | string | `""` | Provide a name in place of the chart name for `app:` labels. |
| fullnameOverride | string | `""` | Provide a name to substitute for the full names of resources. |
| provider | string | `""` | One of the supported secret providers:   - `google` (Google Cloud Secrets Manager)   - `aws` (AWS Secrets Manager and SSM Parameter Store) |
| defaultImagePullSecret | string | `""` | Fallback secret name to use when no image pull secret is found in a Pod. |
| defaultImagePullSecretNamespace | string | `""` | Namespace of the fallback secret name. |
| certificate.useCertManager | bool | `false` | Use jetstack/cert-manager for creating the necessary certificates. This is usually preferred as cert-manager automatically renews certificates. Mutually exclusive with `generate`. |
| certificate.generate | bool | `true` | Generate the necessary certificates during chart install. Mutually exclusive with `useCertManager`. |
| certificate.secretName | string | `""` | The name of the secret to use. If not set and useCertManager or generate is true, a name is generated using the fullname template. |
| serviceAccount.create | bool | `true` | Whether a service account should be created. |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account. |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template. |
| rbac.create | bool | `true` | Specifies whether RBAC resources should be created. If disabled, the operator is responsible for creating the necessary resources based on the templates. |
| podAnnotations | object | `{}` | Custom annotations for a Pod. |
| podSecurityContext | object | `{}` | Pod [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workloads-resources/pod-v1/#security-context) for details. |
| securityContext | object | `{}` | Container [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-container). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workloads-resources/container/#security-context) for details. |
| serviceMonitor.enabled | bool | `false` | Enable Prometheus ServiceMonitor. |
| serviceMonitor.interval | string | `"30s"` | Interval at which metrics should be scraped. |
| resources | object | No requests or limits. | Container resource [requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workloads-resources/container/#resources) for details. |
| autoscaling | object | Disabled by default. | Autoscaling configuration (see [values.yaml](values.yaml) for details). |
| nodeSelector | object | `{}` | [Node selector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) configuration. |
| tolerations | list | `[]` | [Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) for node taints. See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workloads-resources/pod-v1/#scheduling) for details. |
| affinity | object | `{}` | [Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) configuration. See the [API reference](https://kubernetes.io/docs/reference/kubernetes-api/workloads-resources/pod-v1/#scheduling) for details. |
| namespaceSelector | object | `kube-system` namespace is excluded. | [Namespace selector](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-namespaceselector) for the mutating webhook configuration. |
| objectSelector | object | Exclude objects labeled with `kube-init-secrets.doit-intl.com/mutate: skip`. | [Object selector](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-objectselector) for the mutating webhook configuration. |

## Attributions

Forked from [banzaicloud-stable/kube-secrets-init](https://github.com/banzaicloud/banzai-charts/tree/cd93b7049885033c36f5e9551bb39fda7361f835/kube-secrets-init).
