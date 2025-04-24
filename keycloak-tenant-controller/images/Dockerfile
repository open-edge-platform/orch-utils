# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2 AS builder
ARG KTC_GIT_COMMIT

# Set proxy environment variables using build arguments
ARG http_proxy
ARG https_proxy
ARG no_proxy
ARG ftp_proxy
ARG socks_proxy

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $http_proxy
ENV NO_PROXY $http_proxy
ENV FTP_PROXY $ftp_proxy
ENV SOCKS_PROXY $socks_proxy

COPY . /usr/src/keycloak-tenant-controller/
WORKDIR /usr/src/keycloak-tenant-controller/

RUN echo "Using commit ID $KTC_GIT_COMMIT"
RUN	apt-get update && apt-get -y install --no-install-recommends -o APT::Keep-Downloaded-Packages=false build-essential=12.* sudo=1.9.* \
	&& apt-get clean \
	&& rm -rf /var/lib/apt/lists/* \
	&& go install github.com/magefile/mage@latest \
	&& mage binary:build

# hadolint ignore=DL3006
FROM gcr.io/distroless/base

COPY --from=builder /usr/src/keycloak-tenant-controller/bin/* /bin/
COPY ./images/entrypoint.sh /oep/keycloak-tenant-controller/
COPY --from=busybox:1.36.0-uclibc /bin/sh /bin/sh

WORKDIR /oep/keycloak-tenant-controller/
ENTRYPOINT ["/oep/keycloak-tenant-controller/entrypoint.sh"]
