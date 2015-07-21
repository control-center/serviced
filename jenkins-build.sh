#!/bin/bash
set -e
set -x
unset EDITOR # so we don't fail a cli test :\
gvm use go1.4.2
go version
docker version
export GOPATH=$WORKSPACE/gopath
export PATH=$GOPATH/bin:$PATH
sudo umount /exports/serviced_var || true
sudo umount /exports/serviced_var_volumes || true
sudo rm /tmp/serviced-root/var/isvcs/* -Rf
sudo rm /tmp/serviced-test -Rf
docker ps -a -q | xargs --no-run-if-empty docker rm -fv
cd gopath/src/github.com/control-center/serviced
docker images | egrep 'zenoss/ubuntu[ ]+wget' || docker pull zenoss/ubuntu:wget
make clean
sudo GOPATH=$GOPATH make test DOCKERCFG=""
docker ps -a -q | xargs --no-run-if-empty docker rm -fv
sudo rm /tmp/serviced* -Rf
make
make smoketest DOCKERCFG=""
