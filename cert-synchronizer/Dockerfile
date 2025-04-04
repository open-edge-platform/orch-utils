# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2 AS builder
WORKDIR /src


COPY ./go.mod .
COPY ./go.sum .
RUN mkdir -p ./cmd
RUN mkdir -p ./pkg
RUN mkdir -p ./pkg/apiserver
RUN mkdir -p ./pkg/buildflags
RUN mkdir -p ./pkg/log
COPY ./cmd/main.go ./cmd/main.go
COPY ./pkg/apiserver/*.go ./pkg/apiserver/
COPY ./pkg/buildflags/*.go ./pkg/buildflags/
COPY ./pkg/log/*.go ./pkg/log/

RUN go mod download
RUN go get github.com/mitchellh/go-ps
RUN go get github.com/aws/aws-sdk-go-v2/aws
RUN go get github.com/aws/aws-sdk-go-v2/config
RUN go get github.com/aws/aws-sdk-go-v2/credentials
RUN go get github.com/aws/aws-sdk-go-v2/credentials/stscreds
RUN go get github.com/aws/aws-sdk-go-v2/service/sts
RUN go get github.com/aws/aws-sdk-go-v2/service/acm
RUN go get github.com/aws/aws-sdk-go-v2/service/acm/types
RUN go get github.com/aws/aws-sdk-go-v2/service/route53
RUN go get github.com/aws/aws-sdk-go-v2/service/route53/types

ARG GIT_COMMIT
RUN echo "Using commitid $GIT_COMMIT"
RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-X main.Commit=${GIT_COMMIT}" -tags RELEASE ./cmd/main.go


#FROM gcr.io/distroless/base-debian12
FROM gcr.io/distroless/base-debian12:nonroot
#FROM gcr.io/distroless/base-debian12:debug

WORKDIR /

# binaries
COPY --from=builder /src/main /opt/intel/aws_management

EXPOSE 8080/tcp
ENTRYPOINT ["/opt/intel/aws_management"]

