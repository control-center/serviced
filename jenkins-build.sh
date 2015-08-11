#!/bin/bash
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
set -e
set -x
docker ps -a -q | xargs --no-run-if-empty docker rm -fv
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
cd $GOPATH/src/github.com/control-center/serviced
BUILD_TAGS="$(sudo bash ${DIR}/build-tags.sh)"
sudo su - root -c "source /home/jenkins/.gvm/scripts/gvm; gvm use go1.4.2; cd $PWD/volume; GOPATH=$GOPATH godep go test -tags=\"${BUILD_TAGS}\" ./..."
make clean test DOCKERCFG=""
docker ps -a -q | xargs --no-run-if-empty docker rm -fv
sudo rm /tmp/serviced* -Rf
make
make smoketest DOCKERCFG=""
