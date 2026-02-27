# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.7@sha256:931c889bca758a82fcbfcb1b6ed6ca1de30783e9e52e6093ad50060735cb99be AS build

WORKDIR /workspace

# Copy go.mod, go.sum files and download deps
COPY ./go.mod ./go.sum ./
RUN go mod download -x

# Copy sources to the working directory
COPY ./ .

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
      ./cmd/auth-service

FROM scratch

COPY --from=build /workspace/build/app /app

ENTRYPOINT [ "/app" ]
