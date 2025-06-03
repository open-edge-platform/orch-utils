# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2@sha256:30baaea08c5d1e858329c50f29fe381e9b7d7bced11a0f5f1f69a1504cdfbf5e AS build

WORKDIR /workspace

# Copy go.mod, go.sum files and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy just the necessary source files
COPY secrets/ /workspace/secrets/
COPY internal/ /workspace/internal/

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
  ./secrets/cmd/config

FROM gcr.io/distroless/static-debian11:nonroot@sha256:63ebe035fbdd056ed682e6a87b286d07d3f05f12cb46f26b2b44fc10fc4a59ed

COPY --from=build /workspace/build/app /app

ENTRYPOINT [ "/app" ]
