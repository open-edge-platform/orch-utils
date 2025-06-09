#!/bin/sh

# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -eux

NAME=${NAME:-}
IMAGE="file://${NAME}"
TITLE=${TITLE:-}
SKIP_CRD_INSTALLATION=${SKIP_CRD_INSTALLATION:-false}
GRAPHQL_ENABLED=${GRAPHQL_ENABLED:-false}

# User can pass the custom HTTP URL where the graphql plugin can be downloaded via internet
GRAPHQL_PATH=${GRAPHQL_PATH:-NA}

echo '
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  name: datamodels.nexus.com
spec:
  conversion:
    strategy: None
  group: nexus.com
  names:
    kind: Datamodel
    listKind: DatamodelList
    plural: datamodels
    shortNames:
    - datamodel
    singular: datamodel
  scope: Cluster
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          metadata:
            type: object
          spec:
            properties:
              name:
                type: string
              url:
                type: string
              title:
                type: string
                default: "Datamodel APIs"
              enableGraphql:
                type: boolean
                default: false
              graphqlPath:
                type: string
                default: ""
            type: object
    served: true
    storage: true
' | kubectl apply -f -

# This is to support older way of installing datamodel from local folder
if [ "$SKIP_CRD_INSTALLATION" = "false" ]; then
    kubectl apply -f /crds --recursive
    [ "$GRAPHQL_PATH" != "NA" ] && GRAPHQL_ENABLED=true
    if [ -f /build/server ]; then
        GRAPHQL_ENABLED=true
    fi
fi

# Install clusterroles.
kubectl apply -f /clusterroles/clusterrole.yaml

echo "$NAME"
# We will create datamodel object
if [ -n "$NAME" ] && [ -n "$IMAGE" ]; then
  if [ -n "$TITLE" ]; then
    echo '
      apiVersion: nexus.com/v1
      kind: Datamodel
      metadata:
        name: '"$NAME"'
      spec:
        name: '"$NAME"'
        url: '"$IMAGE"'
        title: '"$TITLE"'
        enableGraphql: '"$GRAPHQL_ENABLED"'' | kubectl apply -f -
  else
    echo '
    apiVersion: nexus.com/v1
    kind: Datamodel
    metadata:
      name: '"$NAME"'
    spec:
      name: '"$NAME"'
      url: '"$IMAGE"'
      enableGraphql: '"$GRAPHQL_ENABLED"'' | kubectl apply -f -
  fi
fi
