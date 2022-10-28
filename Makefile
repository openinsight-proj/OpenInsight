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
	GO111MODULE=on go install go.opentelemetry.io/collector/cmd/builder@v0.63.0

# Build the Collector executable.
.PHONY: build-openinsight
build-openinsight:
	$(BUILD_OTELCOL) --output-path=cmd/ --config=builder/otelcol-builder.yaml

.PHONY: openinsight-darwin
openinsight-darwin:
	GO111MODULE=on CGO_ENABLED=0 GOOS=darwin GOARCH=${GOARCH} make build-openinsight

.PHONY: openinsight-linux
openinsight-linux:
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} make build-openinsight

.PHONY: run-openinsight
run-openinsight:
	$(OTELCOL) --config configs/otelcol-contrib.yaml

.PHONY: build-openinsight-docker
build-openinsight-docker-multiarch: build-operator-crosscompile
	docker buildx build \
    			--builder openinsight-multi-platform-builder \
    			--platform linux/amd64,linux/arm64 \
    			--tag $(REGISTRY)/openinsight:$(TAG)  \
				--tag $(REGISTRY)/openinsight:latest \
    			-f ./Dockerfile \
    			--push \
    			.

.PHONY: build-openinsight-docker
build-openinsight-docker: openinsight-linux
	docker build --tag $(REGISTRY)/openinsight:$(TAG)  \
    			--tag $(REGISTRY)/openinsight:latest  \
    			-f ./Dockerfile.single \
    			.

.PHONY: run-otelcol-docker
run-openinsight-demo: build-openinsight-docker
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


