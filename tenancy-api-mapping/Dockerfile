# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.1-bullseye AS builder

COPY cmd/jobapp /app
COPY go.mod /app/go.mod
COPY .gitconfig .netrc .ssh /root/
COPY .netrc /root

RUN mkdir -p /mnt/shared/openapispecs

WORKDIR /app
ENV GOPRIVATE=github.com/open-edge-platform/*
RUN set -x && \
    go mod tidy && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/jobapp

FROM gcr.io/distroless/static-debian12:nonroot
# Run as non-privileged user
USER nobody
COPY apimappingconfigcrs /apimappingconfigcrs
COPY openapispecs /openapispecs

COPY --from=builder /app/bin/jobapp /usr/local/bin/jobapp
COPY --from=builder /mnt/shared/openapispecs /mnt/shared/openapispecs
WORKDIR /bin
# Copy the kubectl to runner
COPY --from=bitnami/kubectl:1.31.1 /opt/bitnami/kubectl/bin/kubectl kubectl

ENTRYPOINT ["jobapp"]
