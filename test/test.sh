#!/bin/bash

set -x # show commands being run
set -e # stop on error

cf create-space staticfile-tests

ORG=${ORG:-"cloudfoundry-community"}
REPO=${REPO:="staticfile-buildpack"}
BRANCH=${BRANCH:="master"}
buildpack="https://github.com/$ORG/$REPO#$BRANCH"

default_stacks=(lucid64 cflinuxfs2)
STACKS=${STACKS:=${default_stacks[*]}}
TEST_APPS=${TEST_APPS:-test/fixtures/*}
for stack in ${STACKS[*]}; do
  for test_app in $TEST_APPS; do
    name=$(basename $test_app)
    cf push $name -p $test_app -b $buildpack -s $stack --random-route
    cf open $name
    cf d $name -f
  done
done

cf delete-space staticfile-tests -f
