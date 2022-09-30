#!/usr/bin/env bash
#
# Generate all protobuf bindings.
#
set -ex

CURRENT_VERSION="v1alpha1"
DIRS=("tracing")

for dir in "${DIRS[@]}"
do
  FILEPATH="./${dir}/${CURRENT_VERSION}"
  for var in `find $FILEPATH -maxdepth 1 -name *.proto`
  do
    protoc -I ./ \
      -I ./third_party \
      --go_out=plugins=grpc,:. \
      --grpc-gateway_out=logtostderr=true:. \
      "${var}";
  done
done

#rsync -a ./openinsight.io/api/* ../api
#rm -rf ./github.com/openinsight

