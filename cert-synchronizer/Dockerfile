# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.1@sha256:595c7847cff97c9a9e76f015083c481d26078f961c9c8dca3923132f51fe12f1 AS builder
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
RUN go get github.com/mitchellh/go-ps@v1.0.0
RUN go get github.com/aws/aws-sdk-go-v2/aws@v1.36.6
RUN go get github.com/aws/aws-sdk-go-v2/config@v1.29.18
RUN go get github.com/aws/aws-sdk-go-v2/credentials@v1.17.71
RUN go get github.com/aws/aws-sdk-go-v2/credentials/stscreds@v1.17.71
RUN go get github.com/aws/aws-sdk-go-v2/service/sts@v1.34.1
RUN go get github.com/aws/aws-sdk-go-v2/service/acm@v1.33.1
RUN go get github.com/aws/aws-sdk-go-v2/service/acm/types@v1.33.1
RUN go get github.com/aws/aws-sdk-go-v2/service/route53@v1.53.1
RUN go get github.com/aws/aws-sdk-go-v2/service/route53/types@v1.53.1

ARG GIT_COMMIT
RUN echo "Using commitid $GIT_COMMIT"
RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-X main.Commit=${GIT_COMMIT}" -tags RELEASE ./cmd/main.go


#FROM gcr.io/distroless/base-debian12
FROM gcr.io/distroless/base-debian12:nonroot@sha256:8b9f2e503e55aff85b79d6b22c7a63a65170e8698ae80de680e3f5ea600977bf
#FROM gcr.io/distroless/base-debian12:debug

WORKDIR /

# binaries
COPY --from=builder /src/main /opt/intel/aws_management

EXPOSE 8080/tcp
ENTRYPOINT ["/opt/intel/aws_management"]
