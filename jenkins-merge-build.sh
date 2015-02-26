#!/bin/bash
set -e
set -x
unset EDITOR # so we don't fail a cli test :\
gvm use go1.4.2
go version
docker version
export GOPATH=$WORKSPACE/gopath
export PATH=$GOPATH/bin:$PATH
cd gopath/src/github.com/control-center/serviced
docker pull zenoss/ubuntu:wget
make clean test smoketest
