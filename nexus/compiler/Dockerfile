# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

ARG BUILDER_TAG=nexus/compiler/builder/amd64:latest

# hadolint ignore=DL3006
FROM $BUILDER_TAG

# Set environment variable for Go cache
ENV GOCACHE=/home/appuser/.cache/go-build

# Copy sources and binaries with correct ownership.
ADD --chown=appuser:appuser compiler.tar /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler
COPY --chown=appuser:appuser cmd/nexus-openapi-gen /go/bin/nexus-openapi-gen
COPY --chown=appuser:appuser cmd/gqlgen /go/bin/gqlgen

RUN mkdir -p /home/appuser/.cache/go-build \
    && chown -R appuser:appuser /home/appuser/.cache

USER root
RUN chown -R appuser:appuser /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler
WORKDIR /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler
RUN mv .git kube-openapi gqlgen nexus install-validator common-library ..
USER appuser

RUN git config --global --add safe.directory '*' \
 && git config --global --add url."https://github.com/".insteadOf "git@github.com:"

RUN make init_submodules

RUN mkdir -p /go/src/nexustmp
WORKDIR /go/src/nexustmp
RUN --mount=type=cache,target="/home/appuser/.cache/go-build" go mod init && \
    /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/scripts/pin_deps.sh && \
    /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/scripts/pin_graphql_build_version.sh && \
    go mod download || go mod download

WORKDIR /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/_deps/github.com/kubernetes/code-generator
RUN --mount=type=cache,target="/home/appuser/.cache/go-build" go mod download || go mod download

USER root
WORKDIR /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/install-validator
RUN --mount=type=cache,target=/home/appuser/.cache/go-build \
    chown -R appuser:appuser /home/appuser/.cache/go-build && \
    su appuser -c 'CGO_ENABLED=0 go build --trimpath -o install.so ./cmd/validate'
USER appuser

WORKDIR /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler
RUN --mount=type=cache,target="/home/appuser/.cache/go-build" go mod download || go mod download

CMD ["make", "preparser"]
