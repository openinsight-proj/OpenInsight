#!/usr/bin/env bash
#
# Generate all grpc-ts file.
#
set -ex

if ! [[ "$0" =~ hack/gen-grpc-ts.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

CURRENT_VERSION="v1alpha1"
GRPC_GATEWAY_TS_OUT=./sdk/$CURRENT_VERSION/ts
mkdir -p $GRPC_GATEWAY_TS_OUT
rm -rf $GRPC_GATEWAY_TS_OUT/*

CURRENT_VERSION="v1alpha1"
DIRS=("tracing")

for dir in "${DIRS[@]}"
do
  FILEPATH="./${dir}/${CURRENT_VERSION}"
  for var in `find $FILEPATH -maxdepth 1 -name *.proto`
  do
  protoc -I ./ \
    -I third_party/ \
    --grpc-gateway-ts_out=$GRPC_GATEWAY_TS_OUT \
    "${var}"
  done

  # find import file from proto and generate ts for them
  for var in `find ${dir} -name "*.proto"|xargs cat|grep -E "^import.*;"|xargs echo |awk '{ gsub(/import/,""); print $0 }'`
  do
    protoc  -I ./ \
      -I third_party/ \
      --grpc-gateway-ts_out=$GRPC_GATEWAY_TS_OUT \
      "${var%?}"
  done

done




