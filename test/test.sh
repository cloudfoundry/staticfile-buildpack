#!/bin/bash

set -x # show commands being run
set -e # stop on error

cf create-space staticfile-tests

ORG=${ORG:-"cloudfoundry-community"}
REPO=${REPO:="staticfile-buildpack"}
BRANCH=${BRANCH:="master"}
buildpack="https://github.com/$ORG/$REPO#$BRANCH"

stacks=( lucid64 )
for stack in $stacks; do
  for test_app in test/fixtures/*; do
    name=$(basename $test_app)
    cf push $name -p $test_app -b $buildpack -s $stack --random-route
    cf open $name
  done
done

cf delete-space staticfile-tests -f
