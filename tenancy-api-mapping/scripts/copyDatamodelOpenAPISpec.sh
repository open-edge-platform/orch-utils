#!/usr/bin/env bash

set -eu -o pipefail

# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Variables
JSON_FILE_PATH="../tenancy-datamodel/build/openapi/edge-orchestrator.intel.com.json"
OUTPUT_PATH="openapispecs/generated/orch-utils.tenancy-datamodel.openapi.yaml"

# Function to log messages
log() {
  echo "$(date +"%Y-%m-%d %H:%M:%S"): $1"
}

# Add headers to output
echo "---
# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
" > "$OUTPUT_PATH"

# Convert JSON to YAML and save it to the specified location
log "Converting JSON to YAML."
yq -P < "$JSON_FILE_PATH" >> "$OUTPUT_PATH"

# Remove PATCH operations from all paths
log "Removing PATCH operations from all paths."
yq eval 'del(.paths.[].patch)' -i $OUTPUT_PATH

# Add BearerAuth security to the OpenAPI YAML specification
log "Adding BearerAuth security to the OpenAPI YAML specification."
yq eval '.components.securitySchemes.BearerAuth = {"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}' -i $OUTPUT_PATH
yq eval '.security = [{"BearerAuth": []}]' -i $OUTPUT_PATH

# Replace the servers field in the OpenAPI YAML specification
log "Replacing the servers field in the OpenAPI YAML specification."
yq eval '.servers = [{"url": "{apiRoot}", "variables": {"apiRoot": {"default": "https://<multitenancy-gateway-host>"}}}]' -i $OUTPUT_PATH
