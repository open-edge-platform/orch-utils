# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.3@sha256:cc9a5d7a008cfe2cbc7ffc752b0d6636ad30fc16e4a648d2e4aac00fd8b25ca3 AS builder

ENV GO111MODULE=on

# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

ARG HTTP_PROXY
ARG HTTPS_PROXY

COPY go.mod go.sum /app/
WORKDIR /app
ENV GOPRIVATE=github.com/open-edge-platform/*
ENV http_proxy=${HTTP_PROXY}
ENV https_proxy=${HTTPS_PROXY}

RUN go mod download

COPY . /app
RUN mkdir -p build/bin && \
    CGO_ENABLED=0 \
    DOCKER_LABEL_REPO_URL=${REPO_URL} \
    DOCKER_LABEL_VERSION=${VERSION} \
    DOCKER_LABEL_REVISION=${REVISION} \
    DOCKER_LABEL_BUILD_DATE=${BUILD_DATE} \
    go build -o build/bin/tenancy-manager ./cmd

FROM gcr.io/distroless/static-debian12:nonroot@sha256:d093aa3e30dbadd3efe1310db061a14da60299baff8450a17fe0ccc514a16639
# Run as non-privileged user
USER nobody

ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

LABEL org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${REPO_URL} \
      org.opencontainers.image.revision=${REVISION} \
      org.opencontainers.image.created=${BUILD_DATE}

COPY --from=builder /app/build/bin/tenancy-manager /usr/local/bin/tenancy-manager

ENTRYPOINT ["tenancy-manager"]