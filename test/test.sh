#!/bin/bash

set -x # show commands being run
set -e # stop on error

cf create-space staticfile-tests

ORG=${ORG:-"cloudfoundry-community"}
REPO=${REPO:="staticfile-buildpack"}
BRANCH=${BRANCH:="master"}
buildpack="https://github.com/$ORG/$REPO#$BRANCH"

stacks=( cflinuxfs2 )
for stack in $stacks; do
  for test_app in test/fixtures/*; do
    echo $stack $test_app
    cf push staticfile-$test_app -p $test_app -b $buildpack -s $stack --random-route
    cf open staticfile-$test_app
  done
done

cf delete-space staticfile-tests -f
