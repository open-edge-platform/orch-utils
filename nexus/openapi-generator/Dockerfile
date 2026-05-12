# SPDX-FileCopyrightText: (C) 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.3@sha256:2981696eed011d747340d7252620932677929cce7d2d539602f56a8d7e9b660b AS builder

COPY . /app
WORKDIR /app/openapi-generator
RUN go mod tidy && go mod download && GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/openapi-generator .

FROM photon:5.0@sha256:1c4f1a00f97a3d5276c6a4f6e1fe1e4372e5b5c4107e813030f716666dd2fb5a
WORKDIR /bin
COPY --from=builder /app/openapi-generator/bin/openapi-generator .
USER 65532:65532
