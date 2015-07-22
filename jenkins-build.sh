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
docker images | egrep 'zenoss/ubuntu[ ]+wget' || docker pull zenoss/ubuntu:wget
go get github.com/tools/godep
cd $GOPATH/src/github.com/control-center/serviced/volume
sudo su root -c "source /home/jenkins/.gvm/scripts/gvm; gvm use go1.4.2; GOPATH=$GOPATH godep go test -tags=root ./..."
cd $GOPATH/src/github.com/control-center/serviced
make clean test DOCKERCFG=""
docker ps -a -q | xargs --no-run-if-empty docker rm -fv
sudo rm /tmp/serviced* -Rf
make
make smoketest DOCKERCFG=""
