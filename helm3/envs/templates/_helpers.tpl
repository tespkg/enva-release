## 
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

{{- define "nsPrefix" -}}
{{- if .Values.global.mode.standard -}}
{{- .Values.global.env | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.namespace.prefix | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "nsPrefixMid" -}}
{{ template "nsPrefix" . }}-{{- .Values.global.namespace.mid -}}
{{- end -}}

{{- define "APP.namespace" -}}
{{ template "nsPrefixMid" . }}-{{- .Chart.Keywords | toString |  regexFind "[a-zA-Z0-9].*[a-zA-Z0-9]" -}}
{{- end -}}

{{- define "hostSuffix" -}}
{{- if .Values.global.mode.standard -}}
{{- .Values.global.env | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.host.prefix | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "GATEWAY.addr" -}}
{{- printf "%s/%s" .Values.global.service.gateway.namespace .Values.global.service.gateway.name  -}}
{{- end -}}