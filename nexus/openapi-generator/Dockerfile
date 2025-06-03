# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2-bullseye@sha256:f50ff25f8331682b44c1582974eb9e620fcb08052fc6ed434f93ca24636fc4d6 AS builder

COPY . /app
WORKDIR /app/openapi-generator
RUN go mod tidy && go mod download && GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/openapi-generator .

FROM photon:5.0@sha256:777bfe878bb8f0590907f817676f7485a302e70da2cbabb03612b3bcdaf4671f
WORKDIR /bin
COPY --from=builder /app/openapi-generator/bin/openapi-generator .
USER 65532:65532
