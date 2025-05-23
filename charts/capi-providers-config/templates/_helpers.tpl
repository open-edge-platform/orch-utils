# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0


{{/*
Defines reusable metrics service configurations for Helm charts.
Usage: {{ include "metrics.servicetemplate" (dict "name" <define name> "namespace" <define namespace> ) }}
*/}}
{{- define "metricsservicetemplate" -}}
apiVersion: v1
kind: Service
metadata:
    name: capi-metrics-svc-{{ .name }}
    namespace: {{ .namespace }}
    labels:
        app: capi-metrics-svc-{{ .name }}
spec:
    type: ClusterIP
    ports:
    - name: metrics
      port: 8080
      targetPort: metrics
      protocol: TCP
{{- end -}}

{{- define "metricsservicemonitortemplate" -}}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
    name: capi-metrics-svc-monitor-{{ .name }}
    namespace: {{ .namespace }}
spec:
    endpoints:
    - port: metrics
      scheme: http
      path: /metrics
    namespaceSelector:
      matchNames:
      - {{ .namespace }}
    selector:
      matchExpressions:
      - key: prometheus.io/service-monitor
        operator: NotIn
        values:
        - "false"
      matchLabels:
        app: capi-metrics-svc-{{ .name }}
{{- end -}}