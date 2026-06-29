{{/* Expand the name of the chart. */}}
{{- define "ical-filter-proxy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Fully qualified app name. */}}
{{- define "ical-filter-proxy.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "ical-filter-proxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "ical-filter-proxy.labels" -}}
helm.sh/chart: {{ include "ical-filter-proxy.chart" . }}
{{ include "ical-filter-proxy.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "ical-filter-proxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ical-filter-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: {{ include "ical-filter-proxy.name" . }}
{{- end }}

{{/* Checksum of the config so pods roll when the config changes. */}}
{{- define "ical-filter-proxy.configChecksum" -}}
{{- toYaml .Values.config | sha256sum }}
{{- end }}
