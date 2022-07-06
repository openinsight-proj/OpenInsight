include ./Makefile.Common

BUILD_OTELCOL=builder
OTELCOL=./dist/otelcol-contrib


# Images management
REGISTRY_SERVER_ADDRESS?=ghcr.io
REGISTRY?=ghcr.io
REGISTRY_USER_NAME?=
REGISTRY_PASSWORD?=
TAG ?= latest

# Build parameters
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BUILD_ARCH ?= linux/$(GOARCH)

TOOLS_MOD_DIR := ./internal/tools
.PHONY: install-tools
install-tools:
	GO111MODULE=on GOPROXY=https://goproxy.cn,direct  go install go.opentelemetry.io/collector/cmd/builder@v0.54.0

.PHONY: build-otelcol
build-otelcol:
	CGO_ENABLED=0 $(BUILD_OTELCOL) --output-path=dist --config=builder/otelcol-builder.yaml

.PHONY: run-otelcol
run-otelcol:
	$(OTELCOL) --config configs/otelcol-contrib.yaml

.PHONY: build-otelcol-docker
build-otelcol-docker:
	echo "Building otelcol for arch = $(BUILD_ARCH)"
	export DOCKER_CLI_EXPERIMENTAL=enabled ;\
	docker buildx create --use --platform=$(BUILD_ARCH) --name otelcol-multi-platform-builder ;\
	docker buildx build \
    			--builder otelcol-multi-platform-builder \
    			--platform $(BUILD_ARCH) \
    			--tag $(REGISTRY)/openinsight:$(TAG)  \
    			--tag $(REGISTRY)/openinsight:latest  \
    			-f ./Dockerfile \
    			.

.PHONY: build-push-otelcol-docker
build-push-otelcol-docker:
	echo "Building otelcol for arch = $(BUILD_ARCH)"
	echo "login ${REGISTRY_SERVER_ADDRESS}"
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u ${REGISTRY_USER_NAME} -p "${REGISTRY_PASSWORD}" ${REGISTRY_SERVER_ADDRESS}
endif
	export DOCKER_CLI_EXPERIMENTAL=enabled ;\
	docker buildx create --use --platform=$(BUILD_ARCH) --name otelcol-multi-platform-builder ;\
	docker buildx build \
    			--builder otelcol-multi-platform-builder \
    			--platform $(BUILD_ARCH) \
    			--tag $(REGISTRY)/openinsight:$(TAG)  \
    			--tag $(REGISTRY)/openinsight:latest  \
    			-f ./Dockerfile \
    			--push \
    			.

.PHONY: run-otelcol-docker
run-otelcol-docker: build-otelcol-docker
	docker-compose up collector

