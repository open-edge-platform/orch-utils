# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.3@sha256:313faae491b410a35402c05d35e7518ae99103d957308e940e1ae2cfa0aac29b AS builder
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

FROM gcr.io/distroless/base:nonroot@sha256:a557d784ac275c287d2bdf3172f47bece8d2a0ef3c0fdefb712e95084a04a562

COPY --from=builder /usr/src/keycloak-tenant-controller/bin/* /bin/
COPY ./images/entrypoint.sh /oep/keycloak-tenant-controller/
COPY --from=busybox:1.36.0-uclibc /bin/sh /bin/sh

WORKDIR /oep/keycloak-tenant-controller/
ENTRYPOINT ["/oep/keycloak-tenant-controller/entrypoint.sh"]
