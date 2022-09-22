include ./Makefile.Common

BUILD_OTELCOL=builder
OTELCOL=./cmd/otelcol-contrib


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

# ALL_MODULES includes ./* dirs (excludes . dir and example with go code)
ALL_MODULES := $(shell find . -type f -name "go.mod" -exec dirname {} \; | sort | egrep  '^./' )

.DEFAULT_GOAL := all

all-modules:
	@echo $(ALL_MODULES) | tr ' ' '\n' | sort

TOOLS_MOD_DIR := ./internal/tools
.PHONY: install-tools
install-tools:
	GO111MODULE=on GOPROXY=https://goproxy.cn,direct  go install go.opentelemetry.io/collector/cmd/builder@v0.55.0

# Build the Collector executable.
.PHONY: build-otelcol
build-otelcol:
	$(BUILD_OTELCOL) --output-path=cmd/ --config=builder/otelcol-builder.yaml

.PHONY: insight-darwin
insight-darwin:
	CGO_ENABLED=0 GOOS=darwin GOPROXY=https://goproxy.cn,direct make build-otelcol

.PHONY: insight-linux
insight-linux:
	CGO_ENABLED=0 GOOS=linux GOPROXY=https://goproxy.cn,direct make build-otelcol

.PHONY: run-otelcol
run-otelcol:
	$(OTELCOL) --config configs/otelcol-contrib.yaml

.PHONY: build-otelcol-docker
build-otelcol-docker: insight-linux
	docker build --tag $(REGISTRY)/openinsight:$(TAG)  \
    			--tag $(REGISTRY)/openinsight:latest  \
    			-f ./Dockerfile \
    			.

.PHONY: build-otelcol-docker-multiarch
build-otelcol-docker-multiarch:
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
run-otelcol-demo: build-otelcol-docker
	docker-compose -f  examples/demo/docker-compose.yaml up

.PHONY: add-tag
add-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Adding tag ${TAG}"
	@git tag -a ${TAG} -s -m "Version ${TAG}"
	@set -e; for dir in $(ALL_MODULES); do \
	  (echo Adding tag "$${dir:2}/$${TAG}" && \
	 	git tag -a "$${dir:2}/$${TAG}" -s -m "Version ${dir:2}/${TAG}" ); \
	done

.PHONY: push-tag
push-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Pushing tag ${TAG}"
	@git push git@github.com:openinsight-proj/OpenInsight.git  ${TAG}
	@set -e; for dir in $(ALL_MODULES); do \
	  (echo Pushing tag "$${dir:2}/$${TAG}" && \
	 	git push git@github.com:openinsight-proj/OpenInsight.git  "$${dir:2}/$${TAG}"); \
	done

.PHONY: delete-tag
delete-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Deleting tag ${TAG}"
	@git tag -d ${TAG}
	@set -e; for dir in $(ALL_MODULES); do \
	  (echo Deleting tag "$${dir:2}/$${TAG}" && \
	 	git tag -d "$${dir:2}/$${TAG}" ); \
	done


