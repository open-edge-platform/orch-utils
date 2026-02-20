# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.5@sha256:8bbd14091f2c61916134fa6aeb8f76b18693fcb29a39ec6d8be9242c0a7e9260 AS builder

COPY . /app
WORKDIR /app

ENV GOPRIVATE=github.com/open-edge-platform/*
RUN set -x && \
    go mod tidy && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/tenantinit ./cmd/tenant-init

FROM gcr.io/distroless/static-debian12:nonroot

# Run as non-privileged user
USER nobody
COPY vendor/ /go/src/github.com/open-edge-platform/orch-utils/tenancy-init/vendor/

COPY --from=builder /app/bin/tenantinit /usr/local/bin/tenantinit
WORKDIR /bin

# Copy the kubectl to runner
# hadolint ignore=DL3022
COPY --from=alpine/kubectl:1.34.1 /usr/local/bin/kubectl kubectl

# No HEALTHCHECK needed for job containers
# trivy:ignore:DS026
ENTRYPOINT ["tenantinit"]
