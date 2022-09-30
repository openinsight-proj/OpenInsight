#!/usr/bin/env bash
#
# Generate all swagger bindings.
#
set -ex

CURRENT_VERSION="v1alpha1"
GOPATH=$GOPATH

mkdir -p ./assets/swagger/$CURRENT_VERSION
mkdir -p assets_bak/$CURRENT_VERSION

DIRS=("tracing")

for dir in "${DIRS[@]}"
do
  FILEPATH="./${dir}/${CURRENT_VERSION}"
  for var in `find $FILEPATH -maxdepth 1 -name *.proto`
  do
    protoc -I ./ \
      -I third_party/ \
      --openapiv2_out ./assets_bak/$CURRENT_VERSION --openapiv2_opt logtostderr=true \
      $var
  done
done

for var in `find ./assets_bak -name "*.json"`
do
  mv "${var}" ./assets/swagger/$CURRENT_VERSION
done

rm -rf assets_bak

cp -r third_party/swagger-ui/* ./assets/swagger/$CURRENT_VERSION

if [ "$(uname)" == "Darwin" ]; then
  sed -i '' 's/https:\/\/petstore.swagger.io\/v2\/swagger.json/.\/rpc.swagger.json/g'  ./assets/swagger/$CURRENT_VERSION/index.html
else
  sed -i  's/https:\/\/petstore.swagger.io\/v2\/swagger.json/.\/rpc.swagger.json/g'  ./assets/swagger/$CURRENT_VERSION/index.html
fi

# generate sdk
mkdir -p ./sdk/$CURRENT_VERSION

if [ -z "$CI_JOB_ID" ]; then
  VOLUME_OPTION="--volume $PWD:$PWD:rw"
  WORKDIR=$PWD
else
  JOB_CONTAINER_ID=`docker ps -q -f "label=com.gitlab.gitlab-runner.job.id=$CI_JOB_ID"`
  VOLUME_OPTION="--volumes-from ${JOB_CONTAINER_ID}:rw"
  WORKDIR=$CI_PROJECT_DIR/api
fi

docker run --platform linux/amd64 --rm --user $(id -u):$(id -g) -v $GOPATH:/go $VOLUME_OPTION -w $WORKDIR quay.m.daocloud.io/goswagger/swagger:v0.28.0 generate client -f ./assets/swagger/$CURRENT_VERSION/query_service.swagger.json -t ./sdk/$CURRENT_VERSION
