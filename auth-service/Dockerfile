# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2 AS build

WORKDIR /workspace

# Set environment variables to allow fetching from private repositories
ENV GONOSUMDB=*
ENV GOPRIVATE=github.com/open-edge-platform/*
ENV GOPROXY=direct

# Configure git to use the GitHub token
# Use build secret for GITHUB_TOKEN
RUN --mount=type=secret,id=GITHUB_TOKEN,env=GITHUB_TOKEN \
    git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

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
