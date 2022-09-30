#!/usr/bin/env bash
#
# Clean all things by auto generate.
#
set -ex

CURRENT_VERSION="v1alpha1"

if [ "$1" == "proto" ]; then
  echo "clean proto file"
  rm -f ./*/$CURRENT_VERSION/*.pb.go ./*/$CURRENT_VERSION/*.pb.gw.go
elif [ "$1" == "swagger" ]; then
  echo "clean swagger file"
  rm -rf assets
elif [ "$1" == "sdk"  ]; then
  echo "clean sdk"
  rm -rf sdk
else
  echo "clean all"
  rm -f ./*/$CURRENT_VERSION/*.pb.go ./*/$CURRENT_VERSION/*.pb.gw.go
  rm -rf assets
  rm -rf sdk
fi
