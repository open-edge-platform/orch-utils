# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2 AS build

WORKDIR /workspace

# Copy go.mod, go.sum files and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy sources to the working directory
COPY aws-sm-proxy/ /workspace/aws-sm-proxy/

ARG org_oci_version=unknown
ARG org_oci_source=unknown
ARG org_oci_revision=unknown
ARG org_oci_created=unknown

LABEL org.opencontainers.image.version=$org_oci_version \
      org.opencontainers.image.source=$org_oci_source \
      org.opencontainers.image.revision=$org_oci_revision \
      org.opencontainers.image.created=$org_oci_created

RUN CGO_ENABLED=0 \
  GOARCH=amd64 \
  GOOS=linux \
  go \
  build \
  -ldflags="-s -w -extldflags=-static -X 'main.Version=$org_oci_version' -X 'main.Revision=$org_oci_revision'" \
  -o ./build/app \
  ./aws-sm-proxy/cmd/proxy

FROM gcr.io/distroless/static-debian11:nonroot

COPY --from=build /workspace/build/app /app

ENTRYPOINT [ "/app" ]
