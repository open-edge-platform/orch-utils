# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

#graphql's and compiler's builder/Dockerfile should be in sync
FROM golang:1.26.2@sha256:5f3787b7f902c07c7ec4f3aa91a301a3eda8133aa32661a3b3a3a86ab3a68a36

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
        make=4.4.1-3 \
        coreutils=9.7-3 \
    && curl -Lo /usr/local/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.8.1/jq-linux-amd64 \
    && echo "020468de7539ce70ef1bceaf7cde2e8c4f2ca6c3afb84642aabc5c97d9fc2a0d /usr/local/bin/jq" | sha256sum -c - \
    && chmod +x /usr/local/bin/jq \
    && go install github.com/onsi/ginkgo/ginkgo@v1.16.0 \
    && go install github.com/onsi/gomega/...@v1.38.2 \
    && go install golang.org/x/tools/cmd/goimports@v0.37.0 \
    && go install github.com/mikefarah/yq/v4@v4.45.1 \
    && go install -buildvcs=false k8s.io/kube-openapi/cmd/openapi-gen@v0.0.0-20250910181357-589584f1c912 \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/v1.58.2/install.sh -o /tmp/install-golangci-lint.sh \
    && echo "562223cd80a52743745ed3fc2d05927942f141b64ecc3423d0a17b312a036fe2 /tmp/install-golangci-lint.sh" | sha256sum -c - \
    && sh /tmp/install-golangci-lint.sh -b "$(go env GOPATH)/bin" v1.58.2 \
    && rm /tmp/install-golangci-lint.sh \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create .cache dir and add write permissions for .cache and /go dirs
RUN mkdir /.cache && chmod -R a+w /.cache /go \
    && chown -R appuser:appuser /.cache /go /home/appuser

# Switch to non-root user
USER appuser
