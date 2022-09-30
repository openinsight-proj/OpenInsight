#!/usr/bin/env bash
set -ex
echo "push grpc-ts file start"

CURRENT_VERSION="v1alpha1"

# git version
LATEST_TAG=$(git describe --tags)
# trim prefix alphabet v, because it's illegal for frontend
LATEST_TAG=$(echo "$LATEST_TAG" |sed  's/^v//g')
# trim git commit sha, because it's illegal for frontend
LATEST_TAG=${LATEST_TAG%-*}
export LATEST_TAG=$LATEST_TAG

if [ -z "$CI_JOB_ID" ]; then
  # run in local
  VOLUME_OPTION="--volume $PWD:$PWD:rw"
  WORKDIR=$PWD/sdk/$CURRENT_VERSION/ts
else
  # run in gitlab-runner
  JOB_CONTAINER_ID=`docker ps -q -f "label=com.gitlab.gitlab-runner.job.id=$CI_JOB_ID"`
  echo $CI_JOB_ID
  VOLUME_OPTION="--volumes-from ${JOB_CONTAINER_ID}:rw"
  WORKDIR=$CI_PROJECT_DIR/api/sdk/$CURRENT_VERSION/ts
fi

docker run --rm  \
  -e VERSION="$LATEST_TAG" \
  -e NPM_TOKEN=$NPM_TOKEN \
	-w $WORKDIR \
	$VOLUME_OPTION \
	docker.m.daocloud.io/node:16.13.2 \
	bash -c '
	 # query the version for package, if the package exists, skip upload
  if [ "`npm view @daocloud-proto/insight@$VERSION| wc -L`" -gt "0" ]
  then
    echo "insight package exits, skipped"
    exit 0
  fi

	rm package.json; # rm package.json incase the project name is illegal
  npm init -y; # init frontend project
	sed -i "s/\"name\":.*$/\"name\":\"\@daocloud-proto\/insight\",/" package.json; # replace the project default name
	sed -i "s/\"version\":.*$/\"version\":\"$VERSION\",/" package.json;    # replace the project default version
	cat package.json;  # look over package.json for debug
	npm config set //registry.npmjs.org/:_authToken $NPM_TOKEN; # init npm token
	npm publish --access=public  # publish the project'

echo "push grpc-ts file succeed"
