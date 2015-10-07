#!/bin/bash

# This file is for running the tests in this directory that require root.
# It is intended as a tool for developers, to be used for quick checks 
# of the root integration tests in this directory. The jenkins-build.sh script
# in the serviced root directory should run these tests for build purposes.

# To run this, you should be able to sudo run commands, and your GOBIN GOPAH and
# PATH variables should be set up properly for control center development. 
# There are no parameterss - just do ./runtest.sh 
# It will run all of the tests in this directory that are marked 
# with // +build integration,root

sudo GOBIN=$GOBIN GOPATH=$GOPATH PATH=$PATH `which godep` go test --tags='integration root'
