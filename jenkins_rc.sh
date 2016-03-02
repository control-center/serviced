#!/bin/bash
#
# Jenkins resource file - this file is sourced by Jenkins build jobs to setup common environment
# variables and functions for various jenkins builds
#

#
# This is a fail-safe to make sure that any files accidentally created as root during a test
# can be cleaned up by jenkins in the next execution of the build job
#
function dochown {
    sudo chown -R jenkins:jenkins $WORKSPACE
}
trap dochown EXIT

. /home/jenkins/.gvm/scripts/gvm
gvm use go1.6
export GOPATH=$WORKSPACE/gopath
export PATH=$WORKSPACE/gopath/bin:$PATH
