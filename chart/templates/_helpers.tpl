{{/*
Expand the name of the chart.
*/}}
{{- define "kube-secrets-init.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a fully qualified app name.
*/}}
{{- define "kube-secrets-init.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "kube-secrets-init.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels.
*/}}
{{- define "kube-secrets-init.labels" -}}
helm.sh/chart: {{ include "kube-secrets-init.chart" . }}
{{ include "kube-secrets-init.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app: secrets-init-webhook
{{- end -}}

{{/*
Selector labels.
*/}}
{{- define "kube-secrets-init.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kube-secrets-init.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
ServiceAccount name.
*/}}
{{- define "kube-secrets-init.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "kube-secrets-init.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Name of the TLS Secret that backs the webhook. Produced by cert-manager or supplied by the user.
*/}}
{{- define "kube-secrets-init.tlsSecretName" -}}
{{- if .Values.tls_existingSecret -}}
{{- .Values.tls_existingSecret -}}
{{- else -}}
{{- printf "%s-tls" (include "kube-secrets-init.fullname" .) -}}
{{- end -}}
{{- end -}}

{{/*
Name of the cert-manager Certificate resource.
*/}}
{{- define "kube-secrets-init.certificateName" -}}
{{- printf "%s-cert" (include "kube-secrets-init.fullname" .) -}}
{{- end -}}

{{/*
Name of the (optional) in-release self-signed Issuer.
*/}}
{{- define "kube-secrets-init.issuerName" -}}
{{- printf "%s-selfsigned" (include "kube-secrets-init.fullname" .) -}}
{{- end -}}

{{/*
MutatingWebhookConfiguration name.
*/}}
{{- define "kube-secrets-init.webhookName" -}}
{{- default (include "kube-secrets-init.fullname" .) .Values.webhook.name -}}
{{- end -}}
