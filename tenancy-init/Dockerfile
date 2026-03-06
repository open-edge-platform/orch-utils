# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.7@sha256:5a79b94c34c299ac0361fbb7c7fca6dc552e166b42341050323fa3ab137d7be9 AS builder

COPY . /app
WORKDIR /app

ENV GOPRIVATE=github.com/open-edge-platform/*
RUN set -x && \
    go mod tidy && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/tenantinit ./cmd/tenant-init

FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1

# Run as non-privileged user
USER nobody
COPY vendor/ /go/src/github.com/open-edge-platform/orch-utils/tenancy-init/vendor/

COPY --from=builder /app/bin/tenantinit /usr/local/bin/tenantinit
WORKDIR /bin

# Copy the kubectl to runner
# hadolint ignore=DL3022
COPY --from=alpine/kubectl:1.35.1 /usr/local/bin/kubectl kubectl

# No HEALTHCHECK needed for job containers
# trivy:ignore:DS026
ENTRYPOINT ["tenantinit"]
