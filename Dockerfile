ARG BASE=golang:1.23-alpine3.20
FROM ${BASE} AS builder
WORKDIR /device-lpmp-go
RUN apk add --no-cache make git openssh-client ca-certificates tzdata && update-ca-certificates
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn
COPY go.mod go.sum ./
COPY vendor* ./
RUN if [ -d vendor ]; then \
      echo "==> using vendor"; \
    else \
      echo "==> no vendor, go mod download"; \
      go mod download -x; \
    fi
COPY . .
ARG ADD_BUILD_TAGS=""
ARG MAKE="make -e ADD_BUILD_TAGS=$ADD_BUILD_TAGS build"
RUN $MAKE
FROM alpine:3.20
LABEL license='SPDX-License-Identifier: Apache-2.0' \
      copyright='Copyright (c) 2019-2021: IOTech'
RUN apk add --no-cache dumb-init ca-certificates tzdata && update-ca-certificates
WORKDIR /
COPY --from=builder /device-lpmp-go/cmd/device-lpmp /device-lpmp
COPY --from=builder /device-lpmp-go/cmd/res /res
EXPOSE 59908
ENTRYPOINT ["dumb-init","--","/device-lpmp"]
CMD ["-cp=keeper.http://edgex-core-keeper:59890","--registry"]
