#!/bin/bash
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

set -e
set -x

cleanup() {
	sudo umount /exports/serviced_var || true
	sudo umount /exports/serviced_var_volumes || true
	sudo rm /tmp/serviced-root/var/isvcs/* -Rf
	sudo rm /tmp/serviced-test -Rf
	sudo rm /tmp/serviced* -Rf
	docker ps -a -q | xargs --no-run-if-empty docker rm -fv
}

#
# Run cleanup first in case a previous build left anything behind
#
cleanup

#
# Setup
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

#
# First, run the tests that require root
#
BUILD_TAGS="$(sudo bash ${DIR}/build-tags.sh) integration"
sudo su - root -c "source /home/jenkins/.gvm/scripts/gvm; gvm use go1.4.2; cd $PWD/volume; GOPATH=$GOPATH godep go test -tags=\"${BUILD_TAGS}\" ./..."
sudo su - root -c "source /home/jenkins/.gvm/scripts/gvm; gvm use go1.4.2; cd $PWD/web; GOPATH=$GOPATH godep go test -tags=\"${BUILD_TAGS}\" ./..."
unset BUILD_TAGS    # Make sure the makefile can choose it's own build tags wihtout interference

#
# Second, run the regular battery of tests
#
cleanup
make clean test DOCKERCFG=""

#
# Lastly, run the smoke tests
#
cleanup
make smoketest DOCKERCFG=""

cleanup
