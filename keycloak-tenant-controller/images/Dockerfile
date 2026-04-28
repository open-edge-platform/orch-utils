# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.2@sha256:b54cbf583d390341599d7bcbc062425c081105cc5ef6d170ced98ef9d047c716 AS builder
ARG KTC_GIT_COMMIT

# Set proxy environment variables using build arguments
ARG http_proxy
ARG https_proxy
ARG no_proxy
ARG ftp_proxy
ARG socks_proxy

ENV HTTP_PROXY=$http_proxy
ENV HTTPS_PROXY=$http_proxy
ENV NO_PROXY=$http_proxy
ENV FTP_PROXY=$ftp_proxy
ENV SOCKS_PROXY=$socks_proxy

COPY . /usr/src/keycloak-tenant-controller/
WORKDIR /usr/src/keycloak-tenant-controller/

RUN echo "Using commit ID $KTC_GIT_COMMIT"
RUN go mod vendor
RUN mkdir -p ./bin \
    && CGO_ENABLED=0 go build -mod=vendor -trimpath \
    -ldflags="-w -s -buildid= -X 'main.Commit=${KTC_GIT_COMMIT}'" \
    -o ./bin/ktc ./cmd/

FROM gcr.io/distroless/base:nonroot@sha256:fb282f8ed3057f71dbfe3ea0f5fa7e961415dafe4761c23948a9d4628c6166fe

COPY --from=builder /usr/src/keycloak-tenant-controller/bin/* /bin/
COPY ./images/entrypoint.sh /oep/keycloak-tenant-controller/
COPY --from=busybox:1.36.0-uclibc /bin/sh /bin/sh

WORKDIR /oep/keycloak-tenant-controller/
ENTRYPOINT ["/oep/keycloak-tenant-controller/entrypoint.sh"]
