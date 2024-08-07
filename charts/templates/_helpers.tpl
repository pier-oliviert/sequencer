{{/*
Expand the name of the chart.
*/}}
{{- define "operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- default .Chart.Name .Values.nameOverride }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "operator.labels" -}}
helm.sh/chart: {{ include "operator.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ include "operator.name" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "operator.serviceAccountName" -}}
{{- if .Values.serviceAccount }}
{{- default (include "operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- include "operator.fullname" . }}-controller-manager
{{- end }}
{{- end }}

{{- define "operator.externalDNS.image" -}}
{{- printf "%s:%s" .Values.externalDNS.image.repository (default (printf "v%s" .Chart.AppVersion) .Values.externalDNS.image.tag) }}
{{- end }}

{{/*
Provider name, Keeps backward compatibility on provider
*/}}
{{- define "operator.externalDNS.providerName" -}}
{{- if eq (typeOf .Values.externalDNS.provider) "string" }}
{{- .Values.externalDNS.provider }}
{{- else }}
{{- .Values.externalDNS.provider.name }}
{{- end }}
{{- end }}
