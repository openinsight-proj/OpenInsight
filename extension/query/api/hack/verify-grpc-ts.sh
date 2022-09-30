#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CURRENT_VERSION="v1alpha1"
SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

DIFFPROTO="${SCRIPT_ROOT}/sdk/${CURRENT_VERSION}/ts"
TMP_DIFFPROTO="../_tmp/${CURRENT_VERSION}"
_tmp="../_tmp"

cleanup() {
  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_DIFFPROTO}"
cp -a "${DIFFPROTO}"/* "${TMP_DIFFPROTO}"

bash "${SCRIPT_ROOT}/hack/gen-grpc-ts.sh"
echo "diffing ${DIFFPROTO} against freshly generated files"

ret=0

echo "${DIFFPROTO}"
echo "${TMP_DIFFPROTO}"

diff -Naupr "${DIFFPROTO}" "${TMP_DIFFPROTO}" || ret=$?
cp -a "${TMP_DIFFPROTO}"/* "${DIFFPROTO}"
if [[ $ret -eq 0 ]]
then
  echo "${DIFFPROTO} up to date."
else
  echo "${DIFFPROTO} is out of date. Please run make gen-grpc-ts"
  exit 1
fi
