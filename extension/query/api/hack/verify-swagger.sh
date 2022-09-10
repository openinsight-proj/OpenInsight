#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CURRENT_VERSION="v1alpha1"
SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

DIFFASSETS="${SCRIPT_ROOT}/assets"
DIFFSDK="${SCRIPT_ROOT}/sdk/$CURRENT_VERSION"
TMP_DIFFASSETS="${SCRIPT_ROOT}/_tmp/api/swagger/$"
TMP_DIFFASDK="${SCRIPT_ROOT}/_tmp/api/sdk"
_tmp="${SCRIPT_ROOT}/_tmp"

cleanup() {
  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_DIFFASSETS}"
mkdir -p "${TMP_DIFFASDK}"
cp -a "${DIFFASSETS}"/* "${TMP_DIFFASSETS}"
cp -a "${DIFFSDK}"/* "${TMP_DIFFASDK}"

bash "${SCRIPT_ROOT}/hack/genswagger.sh"
echo "diffing ${DIFFASSETS} and ${TMP_DIFFASDK} against freshly generated files"

ret=0

diff -Naupr "${DIFFASSETS}" "${TMP_DIFFASSETS}" || ret=$?
diff -Naupr "${DIFFSDK}/models" "${TMP_DIFFASDK}/models" || ret=$?
cp -a "${TMP_DIFFASSETS}"/* "${DIFFASSETS}"
cp -a "${TMP_DIFFASDK}"/* "${DIFFSDK}"
if [[ $ret -eq 0 ]]
then
  echo "${DIFFASSETS} and ${TMP_DIFFASDK} up to date."
else
  echo "${DIFFASSETS} or ${DIFFSDK} is out of date. Please run make genswagger"
  exit 1
fi
