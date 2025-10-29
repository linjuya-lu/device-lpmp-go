.PHONY: build clean docker 

ENABLE_FULL_RELRO=true

ENABLE_PIE=true

MICROSERVICES=cmd/device-lpmp

ARCH ?= aarch64

ifeq ($(ARCH),aarch64)
  GOOS   ?= linux
  GOARCH ?= arm64
else ifeq ($(ARCH),arm64)
  GOOS   ?= linux
  GOARCH ?= arm64
else ifneq (,$(filter armv7l armhf arm,$(ARCH)))
  GOOS   ?= linux
  GOARCH ?= arm
  GOARM  ?= 7
else ifneq (,$(filter x86_64 amd64,$(ARCH)))
  GOOS   ?= linux
  GOARCH ?= amd64
else
  GOOS   ?= $(shell go env GOOS)
  GOARCH ?= $(shell go env GOARCH)
endif

export GOOS GOARCH GOARM

DOCKERS=docker_device_lpmp_go_arm64

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)
SDKVERSION=$(shell cat ./go.mod | grep 'github.com/edgexfoundry/device-sdk-go/v4 v' | sed 's/require//g' | awk '{print $$2}')

ifeq ($(ENABLE_FULL_RELRO), true)
	ENABLE_FULL_RELRO_GOFLAGS = -bindnow
	# -bindnow 仅对使用外部链接器时生效；当前 CGO=0 走内部链接器，等需要 FULL RELRO 再改用 -extldflags。
endif

GOFLAGS=-ldflags "-s -w -X github.com/edgexfoundry/device-lpmp-go.Version=$(VERSION) \
                  -X github.com/edgexfoundry/device-sdk-go/v4/internal/common.SDKVersion=$(SDKVERSION) \
                  $(ENABLE_FULL_RELRO_GOFLAGS)" \
                   -trimpath -mod=readonly

ifeq ($(ENABLE_PIE), true)
	GOFLAGS += -buildmode=pie
endif

build: $(MICROSERVICES)

tidy:
	go mod tidy

cmd/device-lpmp:
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) CGO_ENABLED=0 \
	go build -tags "$(ADD_BUILD_TAGS)" $(GOFLAGS) -o $@ ./cmd

clean:
	rm -f $(MICROSERVICES)

docker_device_lpmp_go_arm64:
	docker buildx build --platform linux/arm64 \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--label "git_sha=$(GIT_SHA)" \
		-t edgexfoundry/device-lpmp:$(GIT_SHA)-arm64 \
		-t edgexfoundry/device-lpmp:$(VERSION)-dev-arm64 \
		--load .

vendor:
	CGO_ENABLED=0 go mod vendor


