# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.2@sha256:2a2b4b5791cea8ae09caecba7bad0bd9631def96e5fe362e4a5e67009fe4ae61 AS builder
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

FROM gcr.io/distroless/base:nonroot@sha256:a696c7c8545ba9b2b2807ee60b8538d049622f0addd85aee8cec3ec1910de1f9

COPY --from=builder /usr/src/keycloak-tenant-controller/bin/* /bin/
COPY ./images/entrypoint.sh /oep/keycloak-tenant-controller/
COPY --from=busybox:1.36.0-uclibc /bin/sh /bin/sh

WORKDIR /oep/keycloak-tenant-controller/
ENTRYPOINT ["/oep/keycloak-tenant-controller/entrypoint.sh"]
