# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

# Use a full-featured base image to build Squid
FROM debian:trixie as build-stage

# Install build dependencies
RUN apt-get update && apt-get install -y --fix-missing \
    squid=6.13-1

# show squid version
RUN echo "Squid version:" && squid -v

# Build from source is currently disabled because it's much slower than installing using apt.
# However, build from source is our long term solution as it includes important security fixes.
# ARG SQUID_VERSION=6.10
# RUN apt-get update && apt-get install -y --fix-missing build-essential libssl-dev wget && \
#     wget https://www.squid-cache.org/Versions/v6/squid-${SQUID_VERSION}.tar.gz && \
#     tar xzf squid-${SQUID_VERSION}.tar.gz && \
#     cd squid-${SQUID_VERSION} && \
#     ./configure --prefix=/usr --localstatedir=/var --libexecdir=/usr/lib/squid --datadir=/usr/share/squid --sysconfdir=/etc/squid --with-openssl --enable-ssl-crtd && \
#     make && \
#     make install

# Find and store the dynamic dependencies of Squid
RUN mkdir /deps
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN ldd /usr/sbin/squid | tr -s '[:space:]' '\n' | grep '^/' | \
    xargs -I {} cp -v --parents {} /deps

# Use Distroless as the final base image
FROM gcr.io/distroless/base-debian12:nonroot

# Copy Squid binaries and configuration from the build stage
COPY --from=build-stage /usr/sbin/squid /usr/sbin/squid
COPY --from=build-stage /usr/lib/squid /usr/lib/squid
COPY --from=build-stage /usr/share/squid/icons /usr/share/squid/icons
COPY --from=build-stage /usr/share/squid /usr/share/squid
COPY --from=build-stage /usr/share/squid/errors/templates /usr/share/squid/errors/templates

# Copy the dependencies
COPY --from=build-stage /deps /

# Expose the Squid port
EXPOSE 3128

# Run Squid in the foreground, with minimal logging
CMD ["/usr/sbin/squid", "-N", "-d1"]
