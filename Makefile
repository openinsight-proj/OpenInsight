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

FIND_MOD_ARGS=-type f -name "go.mod"
TO_MOD_DIR=dirname {} \; | sort | grep -E '^./'
# NONROOT_MODS includes ./* dirs (excludes . dir)
NONROOT_MODS := $(shell find . $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) )

RECEIVER_MODS_0 := $(shell find ./receiver/[a-k]* $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) )
RECEIVER_MODS_1 := $(shell find ./receiver/[l-z]* $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) )
RECEIVER_MODS := $(RECEIVER_MODS_0) $(RECEIVER_MODS_1)
PROCESSOR_MODS := $(shell find ./processor/* $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) )
EXPORTER_MODS := $(shell find ./exporter/* $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) )
EXTENSION_MODS := $(shell find ./extension/* $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) )
INTERNAL_MODS := $(shell find ./internal/* $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) )
OTHER_MODS := $(shell find . $(EX_COMPONENTS) $(EX_INTERNAL) $(FIND_MOD_ARGS) -exec $(TO_MOD_DIR) ) $(PWD)
ALL_MODS := $(RECEIVER_MODS) $(PROCESSOR_MODS) $(EXPORTER_MODS) $(EXTENSION_MODS) $(INTERNAL_MODS) $(OTHER_MODS)

.DEFAULT_GOAL := all

GROUP ?= all
FOR_GROUP_TARGET=for-$(GROUP)-target

all-modules:
	@echo $(ALL_MODULES) | tr ' ' '\n' | sort

TOOLS_MOD_DIR := ./internal/tools
.PHONY: install-builder-tools
install-builder-tools:
	GO111MODULE=on go install go.opentelemetry.io/collector/cmd/builder@latest

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
	docker build --tag $(REGISTRY)/openinsight-proj/openinsight:$(TAG)  \
    			--tag $(REGISTRY)/openinsight-proj/openinsight:dev  \
    			-f ./Dockerfile.single \
    			.

.PHONY: run-otelcol-docker
run-openinsight-demo: build-openinsight-docker
	docker-compose -f  examples/demo/docker-compose.yaml up

.PHONY: gotidy
gotidy:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="tidy"

.PHONY: gomoddownload
gomoddownload:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="moddownload"

.PHONY: gotest
gotest:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="test"

.PHONY: gofmt
gofmt:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="fmt"

.PHONY: golint
golint:
	$(MAKE) $(FOR_GROUP_TARGET) TARGET="lint"

# Define a delegation target for each module
.PHONY: $(ALL_MODS)
$(ALL_MODS):
	@echo "Running target '$(TARGET)' in module '$@' as part of group '$(GROUP)'"
	$(MAKE) -C $@ $(TARGET)

# Trigger each module's delegation target
.PHONY: for-all-target
for-all-target: $(ALL_MODS)

.PHONY: for-receiver-target
for-receiver-target: $(RECEIVER_MODS)

.PHONY: for-receiver-0-target
for-receiver-0-target: $(RECEIVER_MODS_0)

.PHONY: for-receiver-1-target
for-receiver-1-target: $(RECEIVER_MODS_1)

.PHONY: for-processor-target
for-processor-target: $(PROCESSOR_MODS)

.PHONY: for-exporter-target
for-exporter-target: $(EXPORTER_MODS)

.PHONY: for-extension-target
for-extension-target: $(EXTENSION_MODS)

.PHONY: for-internal-target
for-internal-target: $(INTERNAL_MODS)

.PHONY: for-other-target
for-other-target: $(OTHER_MODS)

.PHONY: add-tag
add-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Adding tag ${TAG}"
	@git tag -a ${TAG} -s -m "Version ${TAG}"
	@set -e; for dir in $(NONROOT_MODS); do \
	  (echo Adding tag "$${dir:2}/$${TAG}" && \
	 	git tag -a "$${dir:2}/$${TAG}" -s -m "Version ${dir:2}/${TAG}" ); \
	done

.PHONY: push-tag
push-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Pushing tag ${TAG}"
	@git push git@github.com:openinsight-proj/OpenInsight.git  ${TAG}
	@set -e; for dir in $(NONROOT_MODS); do \
	  (echo Pushing tag "$${dir:2}/$${TAG}" && \
	 	git push git@github.com:openinsight-proj/OpenInsight.git  "$${dir:2}/$${TAG}"); \
	done

.PHONY: delete-tag
delete-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Deleting tag ${TAG}"
	@git tag -d ${TAG}
	@set -e; for dir in $(NONROOT_MODS); do \
	  (echo Deleting tag "$${dir:2}/$${TAG}" && \
	 	git tag -d "$${dir:2}/$${TAG}" ); \
	done


