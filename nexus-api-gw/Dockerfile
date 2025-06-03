# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2-bullseye@sha256:f50ff25f8331682b44c1582974eb9e620fcb08052fc6ed434f93ca24636fc4d6 AS builder

ARG API_GW_COMPONENT_NAME
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG FTP_PROXY
ARG SOCKS_PROXY
ARG NO_PROXY
ARG USER_HOME

COPY . /app
COPY vendor/ /go/src/github.com/open-edge-platform/orch-utils/nexus-api-gw/vendor/

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
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/${API_GW_COMPONENT_NAME}

FROM gcr.io/distroless/static-debian12:nonroot@sha256:188ddfb9e497f861177352057cb21913d840ecae6c843d39e00d44fa64daa51c
# Run as non-privileged user
USER nobody

WORKDIR /bin
COPY --from=builder /app/bin/api-gw /usr/local/bin/api-gw
# Copy policy
COPY --from=builder /app/pkg/auth/authz/authz.rego /usr/local/authz.rego

# copy static api-doc file
COPY --from=builder /app/static/openapispecs/combined/combined_spec.yaml /static/openapispecs/combined/combined_spec.yaml
ENTRYPOINT ["api-gw"]
