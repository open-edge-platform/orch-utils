# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2-bullseye AS builder

ARG API_GW_COMPONENT_NAME
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG FTP_PROXY
ARG SOCKS_PROXY
ARG NO_PROXY
ARG USER_HOME

COPY . /app
COPY .gitconfig .netrc .ssh /root/
COPY .netrc /root

ENV http_proxy=${HTTP_PROXY}
ENV https_proxy=${HTTPS_PROXY}
ENV ftp_proxy=${FTP_PROXY}
ENV socks_proxy=${SOCKS_PROXY}
ENV no_proxy=${NO_PROXY}

WORKDIR /app
ENV GOPRIVATE=github.com/open-edge-platform/*
RUN set -x && \
    apt-get update -y && \
    apt-get install -y --no-install-recommends netcat-openbsd=1.217-3 && \
    go mod tidy && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/${API_GW_COMPONENT_NAME}

FROM gcr.io/distroless/static-debian12:nonroot
# Run as non-privileged user
USER nobody

WORKDIR /bin
COPY --from=builder /app/bin/api-gw /usr/local/bin/api-gw
# Copy policy
COPY --from=builder /app/pkg/auth/authz/authz.rego /usr/local/authz.rego

# copy static api-doc file
COPY --from=builder /app/static/openapispecs/combined/combined_spec.yaml /static/openapispecs/combined/combined_spec.yaml
ENTRYPOINT ["api-gw"]
