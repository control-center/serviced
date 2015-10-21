#!/bin/bash
set -e
set -x

#
# SETUP
#
unset EDITOR # so we don't fail a cli test :\
gvm use go1.4.2
go version
docker version
docker images | egrep 'zenoss/ubuntu[ ]+wget' || docker pull zenoss/ubuntu:wget
export GOPATH=$WORKSPACE/gopath
export PATH=$GOPATH/bin:$PATH
go get github.com/tools/godep
cd $GOPATH/src/github.com/control-center/serviced

sudo umount /exports/serviced_var || true
sudo umount /exports/serviced_var_volumes || true
sudo rm /tmp/serviced-root/var/isvcs/* -Rf
sudo rm /tmp/serviced-test -Rf
docker ps -a -q | xargs --no-run-if-empty docker rm -fv

#
# Second, run the regular battery of tests
#
make clean test DOCKERCFG=""
docker ps -a -q | xargs --no-run-if-empty docker rm -fv
sudo rm /tmp/serviced* -Rf

# do a build?
make

#
# Lastly, run the smoke tests
#
make smoketest DOCKERCFG=""
