{{- define "COMPONENT1.fullname" -}}
{{- printf "%s-%s-%s" .Chart.Name .Values.COMPONENT1.name  .Chart.Version  }}
{{- end -}}
{{- define "COMPONENT2.fullname" -}}
{{- printf "%s-%s-%s" .Chart.Name .Values.COMPONENT2.name  .Chart.Version  }}
{{- end -}}
{{- define "COMPONENT3.fullname" -}}
{{- printf "%s-%s-%s" .Chart.Name .Values.COMPONENT3.name  .Chart.Version  }}
{{- end -}}

{{- define "COMPONENT1.svcname" -}}
{{- printf "%s-%s" .Chart.Name .Values.COMPONENT1.name  }}
{{- end -}}
{{- define "COMPONENT2.svcname" -}}
{{- printf "%s-%s" .Chart.Name .Values.COMPONENT2.name  }}
{{- end -}}
{{- define "COMPONENT3.svcname" -}}
{{- printf "%s-%s" .Chart.Name .Values.COMPONENT3.name  }}
{{- end -}}

{{- define "COMPONENT1.name" -}}
{{- default .Chart.Name .Values.COMPONENT1.name  | trimSuffix "-" -}}
{{- end -}}
{{- define "COMPONENT2.name" -}}
{{- default .Chart.Name .Values.COMPONENT2.name  | trimSuffix "-" -}}
{{- end -}}
{{- define "COMPONENT3.name" -}}
{{- default .Chart.Name .Values.COMPONENT3.name  | trimSuffix "-" -}}
{{- end -}}

{{- define "APP.name" -}}
{{- default .Chart.Name .Values.nameOverride  | trimSuffix "-" -}}
{{- end -}}

{{- define "APP.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_"  | trimSuffix "-" -}}
{{- end -}}

{{- define "APP.version" -}}
{{- default .Chart.Version .Values.nameOverride  | trimSuffix "-" -}}
{{- end -}}

{{- define "APP.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride  | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version  | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "nsPrefix" -}}
{{- if .Values.global.mode.standard -}}
{{- .Values.global.env  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.namespace.prefix  | trimSuffix "-" -}}
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
{{- .Values.global.env  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.host.prefix  | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

# {{- define "postgresUser" -}}
# {{- default "postgres" .Values.global.service.postgres.user  | trimSuffix "-" -}}
# {{- end -}}
# {{- define "postgresPassword" -}}
# {{- default "postgres" .Values.global.service.postgres.password  | trimSuffix "-" -}}
# {{- end -}}
{{- define "postgresHost" -}}
{{- if .Values.global.service.postgres.customHost -}}
{{- .Values.global.service.postgres.customHost  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.service.postgres.svcName -}}.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.infrastructure.postgres }}
{{- end -}}
{{- end -}}
{{- define "postgresPort" -}}
{{- default "5432" .Values.global.service.postgres.svcPort  | trimSuffix "-" -}}
{{- end -}}
{{- define "postgresSVCAddr" -}}
{{ template "postgresHost" . }}:{{ template "postgresPort" . }}
{{- end -}}


{{- define "rabbitmqHost" -}}
{{- if .Values.global.service.rabbitmq.customHost -}}
{{- .Values.global.service.rabbitmq.customHost  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.service.rabbitmq.svcName -}}.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.infrastructure.rabbitmq }}
{{- end -}}
{{- end -}}
{{- define "rabbitmqPort" -}}
{{- default "5672" .Values.global.service.rabbitmq.svcPort  | trimSuffix "-" -}}
{{- end -}}
{{- define "rabbitmqSVCAddr" -}}
{{ template "rabbitmqHost" . }}:{{ template "rabbitmqPort" . }}
{{- end -}}



{{- define "workspaceHost" -}}
{{ template "hostSuffix" . }}.{{- .Values.global.host.baseUrl -}}
{{- end -}}

{{- define "sessmsHost" -}}
sessms-grpc.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.sessms }}
{{- end -}}
{{- define "sessmsSVCAddr" -}}
{{ template "SESSMS.host" . }}:6000
{{- end -}}




















{{- define "CONSUL_HOST" -}}
{{- if .Values.global.service.consul.customHost -}}
{{- .Values.global.service.consul.customHost  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.service.consul.svcName -}}.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.infrastructure.consul }}
{{- end -}}
{{- end -}}
{{- define "CONSUL.svcAddr" -}}
{{ template "CONSUL_HOST" . }}:8500
{{- end -}}








# {{- define "MONGO_USER" -}}
# {{- default "root" .Values.global.service.mongo.user  | trimSuffix "-" -}}
# {{- end -}}
# {{- define "MONGO_PASSWORD" -}}
# {{- default "password" .Values.global.service.mongo.password  | trimSuffix "-" -}}
# {{- end -}}
{{- define "MONGO_HOST" -}}
{{- if .Values.global.service.mongo.customHost -}}
{{- .Values.global.service.mongo.customHost  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.service.mongo.svcName -}}.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.infrastructure.mongo }}
{{- end -}}
{{- end -}}
{{- define "MONGO_PORT" -}}
{{- default "27017" .Values.global.service.mongo.svcPort  | trimSuffix "-" -}}
{{- end -}}
{{- define "MONGO.svcAddr" -}}
{{ template "MONGO_HOST" . }}:{{ template "MONGO_PORT" . }}
{{- end -}}

{{- define "REDIS_HOST" -}}
{{- if .Values.global.service.redis.customHost -}}
{{- .Values.global.service.redis.customHost  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.service.redis.svcName -}}.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.infrastructure.redis }}
{{- end -}}
{{- end -}}
{{- define "REDIS_PORT" -}}
{{- default "6379" .Values.global.service.redis.svcPort  | trimSuffix "-" -}}
{{- end -}}
{{- define "REDIS.svcAddr" -}}
{{ template "REDIS_HOST" . }}:{{ template "REDIS_PORT" . }}
{{- end -}}

# {{- define "RABBITMQ_USER" -}}
# {{- default "guest" .Values.global.service.rabbitmq.user  | trimSuffix "-" -}}
# {{- end -}}
# {{- define "RABBITMQ_PASSWORD" -}}
# {{- default "guest" .Values.global.service.rabbitmq.password  | trimSuffix "-" -}}
# {{- end -}}
{{- define "RABBITMQ_HOST" -}}
{{- if .Values.global.service.rabbitmq.customHost -}}
{{- .Values.global.service.rabbitmq.customHost  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.service.rabbitmq.svcName -}}.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.infrastructure.rabbitmq }}
{{- end -}}
{{- end -}}
{{- define "RABBITMQ_PORT" -}}
{{- default "5672" .Values.global.service.rabbitmq.svcPort  | trimSuffix "-" -}}
{{- end -}}
{{- define "RABBITMQ.svcAddr" -}}
{{ template "RABBITMQ_HOST" . }}:{{ template "RABBITMQ_PORT" . }}
{{- end -}}

# {{- define "MINIO_ACCESS" -}}
# {{- default "minio_access" .Values.global.service.minio.access  | trimSuffix "-" -}}
# {{- end -}}
# {{- define "MINIO_SECRET" -}}
# {{- default "Tes9ting" .Values.global.service.minio.secret  | trimSuffix "-" -}}
# {{- end -}}
{{- define "MINIO_HOST" -}}
{{- if .Values.global.service.minio.customHost -}}
{{- .Values.global.service.minio.customHost  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.service.minio.svcName -}}.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.infrastructure.minio }}
{{- end -}}
{{- end -}}
{{- define "MINIO_PORT" -}}
{{- default "9000" .Values.global.service.minio.svcPort  | trimSuffix "-" -}}
{{- end -}}
{{- define "MINIO.svcAddr" -}}
{{ template "MINIO_HOST" . }}:{{ template "MINIO_PORT" . }}
{{- end -}}

{{- define "ELASTICSEARCH_HOST" -}}
{{- if .Values.global.service.elasticsearch.customHost -}}
{{- .Values.global.service.elasticsearch.customHost  | trimSuffix "-" -}}
{{- else -}}
{{- .Values.global.service.elasticsearch.svcName -}}.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.infrastructure.elasticsearch }}
{{- end -}}
{{- end -}}
{{- define "ELASTICSEARCH_PORT" -}}
{{- default "9200" .Values.global.service.elasticsearch.svcPort  | trimSuffix "-" -}}
{{- end -}}
{{- define "ELASTICSEARCH.svcAddr" -}}
{{ template "ELASTICSEARCH_HOST" . }}:{{ template "ELASTICSEARCH_PORT" . }}
{{- end -}}





{{- define "WORKSPACE.svcAddr" -}}
workspace-be.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.workspace }}:9302
{{- end -}}
{{- define "WORKSPACE.host" -}}
{{ template "hostSuffix" . }}.{{- .Values.global.host.baseUrl -}}
{{- end -}}
{{- define "WORKSPACE.url" -}}
https://{{ template "WORKSPACE.host" . }}
{{- end -}}

{{- define "API.host" -}}
api.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "API.url" -}}
https://api.{{ template "WORKSPACE.host" . }}
{{- end -}}

{{- define "MEERASPACE.allowedOrigins" -}}
{{- printf "%s.%s" "*" .Values.global.host.baseUrl -}}
{{- end -}}



{{- define "MEETING.url" -}}
{{- if eq .Values.global.env "dev" -}}
{{- printf "%s.%s.%s" "https://meeting" "dev" .Values.global.host.baseUrl -}}
{{- else -}}
{{- printf "%s.%s.%s" "https://meeting" "public" .Values.global.host.baseUrl -}}
{{- end -}}
{{- end -}}

{{- define "envsHTTPAddr" -}}
envs.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.envs }}:9112
{{- end -}}
{{- define "envsHost" -}}
{{- .Values.global.host.svcName.envs  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "envsUrl" -}}
https://{{ template "envsHost" . }}
{{- end -}}

{{- define "ssoHTTPAddr" -}}
sso-be.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.sso }}:5557
{{- end -}}
{{- define "SSO.host" -}}
{{- .Values.global.host.svcName.sso  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "SSO.url" -}}
https://{{ template "SSO.host" . }}
{{- end -}}

{{- define "LICENSE.svcAddr" -}}
license.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.license }}:6000
{{- end -}}
{{- define "LICENSE.host" -}}
{{- .Values.global.host.svcName.license -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "LICENSE.url" -}}
https://{{ template "LICENSE.host" . }}
{{- end -}}

{{- define "ACCESSCONTROL.svcAddr" -}}
accesscontrol-be.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.accesscontrol }}:7001
{{- end -}}
{{- define "ACCESSCONTROL.host" -}}
{{- .Values.global.host.svcName.accesscontrol  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "ACCESSCONTROL.url" -}}
https://{{ template "ACCESSCONTROL.host" . }}
{{- end -}}
{{- define "ACCESSCONTROL.svc" -}}
accesscontrol-be.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.accesscontrol }}
{{- end -}}

{{- define "MESSAGEPUSHER.svcAddr" -}}
message-pusher.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.messagepusher }}:6565
{{- end -}}

{{- define "MEERAFS.svcAddr" -}}
meerafs.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.meerafs }}:9011
{{- end -}}


{{- define "PROFILE.graphqlAddr" -}}
profile-be-graphql.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.profile }}:9301
{{- end -}}
{{- define "PROFILE.restAddr" -}}
profile-be-rest.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.profile }}:9302
{{- end -}}
{{- define "PROFILE.grpcAddr" -}}
profile-be-grpc.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.profile }}:50051
{{- end -}}
{{- define "PROFILE.host" -}}
{{- .Values.global.host.svcName.profile  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "PROFILE.url" -}}
https://{{ template "PROFILE.host" . }}
{{- end -}}  

{{- define "CONFIGURATOR.svcAddr" -}}
configurator-be.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.configurator }}:9302
{{- end -}}
{{- define "CONFIGURATOR.grpcAddr" -}}
configurator-be.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.configurator }}:9301
{{- end -}}
{{- define "CONFIGURATOR.host" -}}
{{- .Values.global.host.svcName.configurator -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "CONFIGURATOR.url" -}}
https://{{ template "CONFIGURATOR.host" . }}
{{- end -}}
{{- define "CONFIGURATOR.svc" -}}
configurator-be.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.configurator }}
{{- end -}}


{{- define "HCM.graphqlAddr" -}}
hcm-be-graphql.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.hcm }}:9301
{{- end -}}
{{- define "HCM.host" -}}
{{- .Values.global.namespace.suffix.meeraApp.hcm -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "HCM.url" -}}
https://{{ template "HCM.host" . }}
{{- end -}}

{{- define "CSR.host" -}}
{{- .Values.global.host.svcName.csr -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "CSR.url" -}}
https://{{ template "CSR.host" . }}
{{- end -}}

{{- define "EDGE.host" -}}
{{- .Values.global.host.svcName.edge -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "EDGE.url" -}}
https://{{ template "EDGE.host" . }}
{{- end -}}



{{- define "MAP.host" -}}
{{- .Values.global.host.svcName.map -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "MAP.url" -}}
https://{{ template "MAP.host" . }}
{{- end -}}

{{- define "PDO.host" -}}
{{- .Values.global.host.svcName.pdo -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "PDO.url" -}}
https://{{ template "PDO.host" . }}
{{- end -}}

{{- define "PLANNER.host" -}}
{{- .Values.global.host.svcName.planner -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "PLANNER.url" -}}
https://{{ template "PLANNER.host" . }}
{{- end -}}

{{- define "TROVE.host" -}}
{{- .Values.global.host.svcName.trove -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "TROVE.url" -}}
https://{{ template "TROVE.host" . }}
{{- end -}}

{{- define "AMC.host" -}}
{{- .Values.global.host.svcName.amc -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "AMC.url" -}}
https://{{ template "AMC.host" . }}
{{- end -}}

{{- define "CRM.host" -}}
{{- .Values.global.host.svcName.crm -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "CRM.url" -}}
https://{{ template "CRM.host" . }}
{{- end -}}

{{- define "LANG.host" -}}
{{- .Values.global.host.svcName.lang -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "LANG.url" -}}
https://{{ template "LANG.host" . }}
{{- end -}}

{{- define "ORGANIZATION.host" -}}
{{- .Values.global.host.svcName.organization -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "ORGANIZATION.url" -}}
https://{{ template "ORGANIZATION.host" . }}
{{- end -}}

{- define "REACH.svcAddr" -}}
reach-be.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.reach }}:9302
{{- define "REACH.host" -}}
{{- .Values.global.host.svcName.reach -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "REACH.url" -}}
https://{{ template "REACH.host" . }}
{{- end -}}

{{- define "GATEWAY.addr" -}}
{{- printf "%s/%s" .Values.global.service.gateway.namespace .Values.global.service.gateway.name  -}}
{{- end -}}

{{- define "BI.host" -}}
{{- .Values.global.host.svcName.bi  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "BI.url" -}}
https://{{ template "BI.host" . }}
{{- end -}}

{{- define "CADRE.host" -}}
{{- .Values.global.host.svcName.cadre  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "CADRE.url" -}}
https://{{ template "CADRE.host" . }}
{{- end -}}

{{- define "OKR.host" -}}
{{- .Values.global.host.svcName.okr  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "OKR.url" -}}
https://{{ template "OKR.host" . }}
{{- end -}}

{{- define "MANAGER.host" -}}
{{- .Values.global.host.svcName.manager  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "MANAGER.url" -}}
https://{{ template "MANAGER.host" . }}
{{- end -}}

{{- define "PERMIT.host" -}}
{{- .Values.global.host.svcName.permit  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "PERMIT.url" -}}
https://{{ template "PERMIT.host" . }}
{{- end -}}

{{- define "I18N.host" -}}
{{- .Values.global.host.svcName.lang  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "I18N.url" -}}
https://{{ template "I18N.host" . }}
{{- end -}}

{{- define "PSA.host" -}}
{{- .Values.global.host.svcName.psa  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "PSA.url" -}}
https://{{ template "PSA.host" . }}
{{- end -}}

{{- define "REGULATION.host" -}}
{{- .Values.global.host.svcName.regulation  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "REGULATION.url" -}}
https://{{ template "REGULATION.host" . }}
{{- end -}}

{{- define "PEOPLEANALYTICS.host" -}}
{{- .Values.global.host.svcName.peopleanalytics  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "PEOPLEANALYTICS.url" -}}
https://{{ template "PEOPLEANALYTICS.host" . }}
{{- end -}}

{{- define "PULSE.host" -}}
{{- .Values.global.host.svcName.pulse  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "PULSE.url" -}}
https://{{ template "PULSE.host" . }}
{{- end -}}

{{- define "INTERNATIONAL.host" -}}
i18n.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "INTERNATIONAL.url" -}}
https://{{ template "INTERNATIONAL.host" . }}
{{- end -}}

{{- define "FILEMANAGER.host" -}}
filemanager-file-manager.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "FILEMANAGER.url" -}}
https://{{ template "FILEMANAGER.host" . }}
{{- end -}}

{{- define "SENDER.svcAddr" -}}
sender.{{ template "nsPrefixMid" . }}-{{.Values.global.namespace.suffix.meeraApp.sender }}:5557
{{- end -}}
{{- define "SENDER.host" -}}
{{- .Values.global.host.svcName.sender  -}}.{{ template "WORKSPACE.host" . }}
{{- end -}}
{{- define "SENDER.url" -}}
https://{{ template "SENDER.host" . }}
{{- end -}}