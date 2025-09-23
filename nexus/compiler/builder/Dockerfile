# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#graphql's and compiler's builder/Dockerfile should be in sync
FROM golang:1.25.1-bookworm

# Create a non-root user with home directory
RUN useradd -m -d /home/appuser appuser

# Set environment variables
ENV GOPATH=/go
ENV PATH=$PATH:/usr/local/go/bin:/go/bin
ENV GO111MODULE=on
ENV GOPROXY=""
ENV GOFLAGS="-buildvcs=false"
ENV HOME=/home/appuser

# Configure git and install necessary packages
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
# hadolint ignore=DL3008
RUN git config --global --add safe.directory '*' \
    && echo "deb http://deb.debian.org/debian unstable main" | tee -a /etc/apt/sources.list.d/unstable.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends -t unstable wget \
    && apt-get install -y --no-install-recommends \
        make=4.4.1-2 \
        coreutils=9.7-3 \
        openssh-client \
    && curl -Lo /usr/local/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.7.1/jq-linux-amd64 \
    && chmod +x /usr/local/bin/jq \
    && go install github.com/onsi/ginkgo/ginkgo@v1.16.0 \
    && go install github.com/onsi/gomega/...@v1.18.0 \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && go install github.com/mikefarah/yq/v4@v4.45.1 \
    #&& go install -buildvcs=false k8s.io/kube-openapi/cmd/openapi-gen@v0.0.0-20250304201544-e5f78fe3ede9
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v1.64.6 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create .cache dir and add write permissions for .cache and /go dirs
RUN mkdir /.cache && chmod -R a+w /.cache /go \
    && chown -R appuser:appuser /.cache /go /home/appuser

# Switch to non-root user
USER appuser
