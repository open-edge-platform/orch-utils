# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.1-bullseye AS builder

COPY . /app
WORKDIR /app/openapi-generator
RUN go mod tidy && go mod download && GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/openapi-generator .

FROM gcr.io/nsx-sm/photon:4.0
WORKDIR /bin
COPY --from=builder /app/openapi-generator/bin/openapi-generator .
USER 65532:65532
