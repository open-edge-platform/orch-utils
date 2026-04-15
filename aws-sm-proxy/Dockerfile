# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.2@sha256:5f3787b7f902c07c7ec4f3aa91a301a3eda8133aa32661a3b3a3a86ab3a68a36 AS build

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

FROM gcr.io/distroless/static-debian11:nonroot@sha256:63ebe035fbdd056ed682e6a87b286d07d3f05f12cb46f26b2b44fc10fc4a59ed

COPY --from=build /workspace/build/app /app

ENTRYPOINT [ "/app" ]
