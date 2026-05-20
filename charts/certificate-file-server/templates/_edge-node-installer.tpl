{{- define "certificate-file-server.edge-node-installer" -}}
{{- $config := .Values.infraConfig | default dict -}}
{{- $gatewayCert := "" -}}
{{- $bootsCert := "" -}}
{{- if .Values.orchSecretName -}}
{{- $secret := (lookup "v1" "Secret" "orch-gateway" .Values.orchSecretName) -}}
{{- if $secret -}}
{{- $gatewayCert = index $secret.data "ca.crt" -}}
{{- end -}}
{{- end -}}
{{- if .Values.orchBootsSecretName -}}
{{- $bootsSecret := (lookup "v1" "Secret" "orch-gateway" .Values.orchBootsSecretName) -}}
{{- if $bootsSecret -}}
{{- $bootsCert = index $bootsSecret.data "ca.crt" -}}
{{- end -}}
{{- end -}}
#!/bin/bash

# SPDX-FileCopyrightText: (C) 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -eu
{{- if index $config "enProxyHTTP" }}
grep -qF "http_proxy" /etc/environment || echo http_proxy={{ index $config "enProxyHTTP" }} >> /etc/environment
grep -qF "https_proxy" /etc/environment || echo https_proxy={{ index $config "enProxyHTTPS" | default (index $config "enProxyHTTP") }} >> /etc/environment
{{- end }}
{{- if index $config "enProxyFTP" }}
grep -qF "ftp_proxy" /etc/environment || echo ftp_proxy={{ index $config "enProxyFTP" }} >> /etc/environment
{{- end }}
{{- if index $config "enProxySocks" }}
grep -qF "socks_proxy" /etc/environment || echo socks_proxy={{ index $config "enProxySocks" }} >> /etc/environment
{{- end }}
{{- if index $config "enProxyNoProxy" }}
grep -qF "no_proxy" /etc/environment || echo no_proxy={{ index $config "enProxyNoProxy" }} >> /etc/environment
{{- end }}
. /etc/environment
{{- if index $config "enProxyHTTP" }}
export http_proxy https_proxy
{{- end }}
{{- if index $config "enProxyFTP" }}
export ftp_proxy
{{- end }}
{{- if index $config "enProxySocks" }}
export socks_proxy
{{- end }}
{{- if index $config "enProxyNoProxy" }}
export no_proxy
{{- end }}

# TODO: Investigate and find the root cause of the issue with the HOME environment variable not found.
# Set the HOME environment variable if not already set
export HOME=/root

SETUP_STATUS_FILENAME="install_pkgs_status"
STATUS_FILENAME=".success_install_status"
SCRIPT_DIR=$(pwd)
touch "$SCRIPT_DIR/$SETUP_STATUS_FILENAME"
touch "$SCRIPT_DIR/$STATUS_FILENAME"

# Write agent variables to /etc/edge-node/node/agent_variables
# These are sourced and used by both agents and install functions
mkdir -p /etc/edge-node/node
cat <<EOF > /etc/edge-node/node/agent_variables
CLUSTER_ORCH_URL={{ index $config "orchCluster" }}
HW_INVENTORY_URL={{ index $config "orchInfra" }}
NODE_ONBOARDING_ENABLED=true
NODE_ONBOARDING_URL={{ index $config "orchInfra" }}
NODE_ONBOARDING_HEARTBEAT=10s
NODE_ACCESS_URL={{ index $config "orchKeycloak" }}
NODE_RS_URL={{ index $config "orchRelease" }}
PLATFORM_MANAGEABILITY_URL={{ index $config "orchDeviceManager" }}
NODE_SERVICE_CLIENTS={{ index $config "enServiceClients" }}
NODE_OUTBOUND_CLIENTS={{ index $config "enOutboundClients" }}
NODE_METRICS_ENABLED={{ index $config "enMetricsEnabled" | default "false" }}
PLATFORM_MANAGEABILITY_METRICS={{ index $config "enMetricsEnabled" | default "false" }}
NODE_TOKEN_CLIENTS={{ index $config "enTokenClients" }}
FILE_RS_ROOT={{ index $config "enFilesRsRoot" }}
DEB_PACKAGES_REPO={{ index $config "enDebianPackagesRepo" }}
RS_TYPE={{ index $config "rsType" | default "no-auth" }}
MANIFEST_REGISTRY={{ index $config "registryService" }}
MANIFEST_REPO={{ index $config "enManifestRepo" }}
MANIFEST_TAG={{ index $config "enAgentManifestTag" }}
OM_SVC={{ index $config "omSvc" }}
OM_STREAM_SVC={{ index $config "omStreamSvc" }}
CADDY_APT_PROXY_URL=$(echo "{{ index $config "orchFileServer" }}" | cut -d: -f1)
CADDY_APT_PROXY_PORT=$(echo "{{ index $config "orchFileServer" }}" | cut -d: -f2)
REGISTRY_URL=$(echo "{{ index $config "orchRegistry" }}" | cut -d: -f1)
CADDY_REGISTRY_PROXY_URL=$(echo "{{ index $config "orchRegistry" }}" | cut -d: -f1)
CADDY_REGISTRY_PROXY_PORT=$(echo "{{ index $config "orchRegistry" }}" | cut -d: -f2)
RPS_ADDRESS=$(echo "{{ index $config "orchRPSHost" }}" | cut -d: -f1)
KEYCLOAK_FQDN=$(echo "{{ index $config "orchKeycloak" }}" | cut -d: -f1)
RELEASE_FQDN=$(echo "{{ index $config "orchRelease" }}" | cut -d: -f1)
# Observability endpoints for logs and metrics, used if observability enabled.
OBSERVABILITY_LOGGING_URL=$(echo "{{ index $config "orchPlatformObsLogs" }}" | cut -d: -f1)
OBSERVABILITY_LOGGING_PORT=$(echo "{{ index $config "orchPlatformObsLogs" }}" | cut -d: -f2)
OBSERVABILITY_METRICS_URL=$(echo "{{ index $config "orchPlatformObsMetrics" }}" | cut -d: -f1)
OBSERVABILITY_METRICS_PORT=$(echo "{{ index $config "orchPlatformObsMetrics" }}" | cut -d: -f2)
EOF
. /etc/edge-node/node/agent_variables

# Decode CA certificates from base64
GATEWAY_CA_CERT_B64='{{ $gatewayCert }}'
{{- if $bootsCert }}
BOOTS_CA_CERT_B64='{{ $bootsCert }}'
{{- else }}
BOOTS_CA_CERT_B64=''
{{- end }}
CA_CERT=$(echo "$GATEWAY_CA_CERT_B64" | base64 -d)
if [ -n "${BOOTS_CA_CERT_B64:-}" ]; then
    BOOTS_CERT=$(echo "$BOOTS_CA_CERT_B64" | base64 -d)
    CA_PEM="${CA_CERT}
${BOOTS_CERT}"
else
    CA_PEM="$CA_CERT"
fi

# NTP Server
NTP_SERVER="{{ index $config "ntpServer" | default "ntp1.server.org,ntp2.server.org" }}"

# Firewall Rules  
FW_REQ_RULES='{{ index $config "firewallReqAllow" | default "[]" }}'
FW_CFG_RULES='{{ index $config "firewallCfgAllow" | toJson | default "[]" }}'

# Kernel configuration
KERNEL_CONFIG_OVER_COMMIT_MEMORY={{ index $config "systemConfigVmOverCommitMemory" | default "1" }}
KERNEL_CONFIG_KERNEL_PANIC={{ index $config "systemConfigKernelPanic" | default "10" }}
KERNEL_CONFIG_PANIC_ON_OOPS={{ index $config "systemConfigKernelPanicOnOops" | default "1" }}
KERNEL_CONFIG_MAX_USER_INSTANCE={{ index $config "systemConfigFsInotifyMaxUserInstances" | default "8192" }}

# Other constants
MANIFEST_FILE="ena-manifest.yaml"

# Agent versions (will be populated from manifest)
DEVICE_DISCOVERY_VERSION=""
NODE_AGENT_VERSION=""
PLATFORM_MANAGEABILITY_VERSION=""
PLATFORM_TELEMETRY_VERSION=""
CADDY_VERSION=""
APT_DISTRO=""

# Function to extract version from manifest YAML
get_version_from_manifest() {
    local package_name=$1
    local manifest_file=$2
    grep -A 1 "name: $package_name" "$manifest_file" | grep "version:" | awk '{print $2}' | head -1 || true
}

# Function to extract codename from manifest YAML
get_codename_from_manifest() {
    local manifest_file=$1
    grep "codename:" "$manifest_file" | awk '{print $2}' | head -1 || true
}

download_and_parse_manifest() {
    if grep -q "download_and_parse_manifest done" "$SCRIPT_DIR"/$STATUS_FILENAME; then
        echo "Skipping manifest download"
    else
        echo "Downloading agent manifest..."
        oras pull "${MANIFEST_REGISTRY}/${MANIFEST_REPO}:${MANIFEST_TAG}" -o "$SCRIPT_DIR"

        if [ ! -f "$SCRIPT_DIR/$MANIFEST_FILE" ]; then
            echo "ERROR: Failed to download manifest file"
            exit 1
        fi

        echo "download_and_parse_manifest done" | tee -a "$SCRIPT_DIR"/$STATUS_FILENAME
    fi
}

{{- $content := .Files.Get "scripts/edge-node-installer.sh" }}
{{- if $content }}
{{- $lines := splitList "\n" $content }}
{{- $foundMarker := false }}
{{- range $line := $lines }}
{{- if contains "FUNCTIONS START BELOW" $line }}
{{- $foundMarker = true }}
{{- else if $foundMarker }}
{{ $line }}
{{- end }}
{{- end }}
{{- else }}
# ERROR: Could not load script functions from scripts/edge-node-installer.sh
# Ensure the file exists in the chart directory
{{- end }}
{{- end -}}
