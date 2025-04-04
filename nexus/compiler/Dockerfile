# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

ARG BUILDER_TAG

# hadolint ignore=DL3006
FROM $BUILDER_TAG
ADD compiler.tar /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler
COPY cmd/nexus-openapi-gen /go/bin/nexus-openapi-gen
COPY cmd/gqlgen /go/bin/gqlgen

WORKDIR /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler
RUN mv .git kube-openapi gqlgen nexus install-validator common-library  ..
RUN git config --global --add safe.directory '*' && git config --global --add url."https://github.com/".insteadOf "git@github.com:"
RUN make init_submodules

ENV GOCACHE=/root/.cache/go-build

RUN mkdir /go/src/nexustmp
WORKDIR /go/src/nexustmp
RUN --mount=type=cache,target="/root/.cache/go-build" go mod init && \
    /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/scripts/pin_deps.sh && \
    /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/scripts/pin_graphql_build_version.sh && \
    go mod download || go mod download

WORKDIR /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/_deps/github.com/kubernetes/code-generator
RUN --mount=type=cache,target="/root/.cache/go-build"  go mod download  || go mod download

WORKDIR /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/install-validator
RUN --mount=type=cache,target="/root/.cache/go-build"  CGO_ENABLED=0 go build --trimpath -o install.so ./cmd/validate

WORKDIR /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler
RUN --mount=type=cache,target="/root/.cache/go-build" go mod download  || go mod download

CMD ["make", "preparser"]
