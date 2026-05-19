# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.3@sha256:313faae491b410a35402c05d35e7518ae99103d957308e940e1ae2cfa0aac29b AS builder
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
FROM gcr.io/distroless/base-debian12:nonroot@sha256:7a75a36f4bec82a7542c64195e402907486f9a4dd2f8797a976aa0cf31cfb470
#FROM gcr.io/distroless/base-debian12:debug

WORKDIR /

# binaries
COPY --from=builder /src/main /opt/intel/aws_management

EXPOSE 8080/tcp
ENTRYPOINT ["/opt/intel/aws_management"]
