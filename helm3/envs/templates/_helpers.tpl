{{- define "APP.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "APP.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "APP.version" -}}
{{- default .Chart.Version .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "APP.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}


# {{- define "APP.namespace" -}}
# {{ .Values.global.nsPrefix }}-{{- .Chart.Keywords | toString |  regexFind "[a-zA-Z0-9].*[a-zA-Z0-9]" -}}
# {{- end -}}