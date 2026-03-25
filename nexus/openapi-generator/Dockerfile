# SPDX-FileCopyrightText: (C) 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.8@sha256:dfae680962532eeea67ab297f1166c2c4e686edb9a8f05f9d02d96fc9191833e AS builder

COPY . /app
WORKDIR /app/openapi-generator
RUN go mod tidy && go mod download && GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/openapi-generator .

FROM photon:5.0@sha256:1c4f1a00f97a3d5276c6a4f6e1fe1e4372e5b5c4107e813030f716666dd2fb5a
WORKDIR /bin
COPY --from=builder /app/openapi-generator/bin/openapi-generator .
USER 65532:65532
