# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.3@sha256:efaccb5b497e90df3ebe5216cc25cd9f98e73874e2d638b56e38d4a3f098c41c AS builder
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
RUN	apt-get update && apt-get -y install --no-install-recommends -o APT::Keep-Downloaded-Packages=false build-essential=12.12 sudo=1.9.16p2-3+deb13u1 \
	&& apt-get clean \
	&& rm -rf /var/lib/apt/lists/* \
	&& go install github.com/magefile/mage@v1.15.0 \
	&& mage binary:build

FROM gcr.io/distroless/base:nonroot@sha256:fb282f8ed3057f71dbfe3ea0f5fa7e961415dafe4761c23948a9d4628c6166fe

COPY --from=builder /usr/src/keycloak-tenant-controller/bin/* /bin/
COPY ./images/entrypoint.sh /oep/keycloak-tenant-controller/
COPY --from=busybox:1.36.0-uclibc /bin/sh /bin/sh

WORKDIR /oep/keycloak-tenant-controller/
ENTRYPOINT ["/oep/keycloak-tenant-controller/entrypoint.sh"]
