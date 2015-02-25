#!/bin/bash
gvm use go1.4.2
go version
docker version
sudo umount /exports/serviced_var || true
sudo rm /tmp/serviced-root/var/isvcs/* -Rf
sudo rm /tmp/serviced-test -Rf
docker ps -a -q | xargs --no-run-if-empty docker rm -f
cd gopath/src/github.com/control-center/serviced
docker images | egrep 'zenoss/ubuntu[ ]+wget' || docker pull zenoss/ubuntu:wget
make clean test DOCKERCFG=""
docker ps -a -q | xargs --no-run-if-empty docker rm -f
sudo rm /tmp/serviced* -Rf
make
make smoketest DOCKERCFG=""
