# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2-bullseye AS builder

ENV GO111MODULE=on
ARG MAKE_TARGET=go-build

# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG FTP_PROXY
ARG SOCKS_PROXY
ARG NO_PROXY
ARG USER_HOME

COPY . /app
COPY vendor/ /go/src/github.com/open-edge-platform/orch-utils/tenancy-manager/vendor/

WORKDIR /app
ENV GOPRIVATE=github.com/open-edge-platform/*
ENV http_proxy=${HTTP_PROXY}
ENV https_proxy=${HTTPS_PROXY}
RUN apt-get update -y && apt-get -y install netcat-openbsd=1.217-3 --no-install-recommends
RUN mkdir -p build/bin && CGO_ENABLED=0 DOCKER_LABEL_REPO_URL=${REPO_URL} DOCKER_LABEL_VERSION=${VERSION} DOCKER_LABEL_REVISION=${REVISION} DOCKER_LABEL_BUILD_DATE=${BUILD_DATE} go build -o build/bin ./...

FROM gcr.io/distroless/static-debian12:nonroot
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
