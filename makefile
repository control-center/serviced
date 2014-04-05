################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################


pwdchecksum := $(shell pwd | md5sum | awk '{print $$1}')
dockercache := /tmp/serviced-dind-$(pwdchecksum)

default: build_binary

install: build_binary install-nsinit
	go install github.com/zenoss/serviced/serviced

install-nsinit:
	# use go install to install nsinit to $GOBIN
	#    go install github.com/dotcloud/docker/pkg/libcontainer/nsinit/nsinit
	# WORKAROUND until master branch contains nsinit hotfix-0.9.2 (github.com/dotcloud/docker/issues/4975)
	GOPATH=$${PWD}/nsinit go get github.com/dotcloud/docker/pkg/libcontainer/nsinit/nsinit && \
	cd nsinit/src/github.com/dotcloud/docker/pkg/libcontainer/nsinit/nsinit && \
	git checkout 867b2a90c228f62cdcd44907ceef279a2d8f1ac5 && \
	cd - && \
	GOPATH=$${PWD}/nsinit go install github.com/dotcloud/docker/pkg/libcontainer/nsinit/nsinit


build_binary:
	make install-nsinit
	cd serviced && make
	cd isvcs && make

go:
	cd serviced && go build


pkgs:
	cd pkg && make rpm && make deb

dockerbuild_binary: docker_ok
	docker build -t zenoss/serviced-build build
	docker run -rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd /go/src/github.com/zenoss/serviced/pkg/ && make clean && mkdir -p /go/src/github.com/zenoss/serviced/pkg/build/tmp"
	echo "Using dock-in-docker cache dir $(dockercache)"
	mkdir -p $(dockercache)
	time docker run -rm \
	-privileged \
	-v $(dockercache):/var/lib/docker \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	-v `pwd`/pkg/build/tmp:/tmp \
	-e BUILD_NUMBER=$(BUILD_NUMBER) -t \
	zenoss/serviced-build /bin/bash \
	-c '/usr/local/bin/wrapdocker && make build_binary'


dockerbuild: docker_ok
	docker build -t zenoss/serviced-build build
	docker run -rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/bash -c "cd /go/src/github.com/zenoss/serviced/pkg/ && make clean && mkdir -p /go/src/github.com/zenoss/serviced/pkg/build/tmp"
	echo "Using dock-in-docker cache dir $(dockercache)"
	mkdir -p $(dockercache)
	time docker run -rm \
	-privileged \
	-v $(dockercache):/var/lib/docker \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	-v `pwd`/pkg/build/tmp:/tmp \
	-e BUILD_NUMBER=$(BUILD_NUMBER) -t \
	zenoss/serviced-build /bin/bash \
	-c '/usr/local/bin/wrapdocker && make build_binary pkgs'

test: build_binary docker_ok
	go test
	cd dao && make test
	cd web && go test
	cd serviced && go test
	cd utils && go test

docker_ok:
	if docker ps >/dev/null; then \
		echo "docker OK"; \
	else \
		echo "Check 'docker ps' command"; \
		exit 1;\
	fi

clean:
	cd dao && make clean
	cd serviced && ./godep restore && go clean -r && go clean -i github.com/zenoss/serviced/... # this cleans all dependencies
	docker run -rm \
	-v `pwd`:/go/src/github.com/zenoss/serviced \
	zenoss/serviced-build /bin/sh -c "cd /go/src/github.com/zenoss/serviced && make clean_fs" || exit 0

clean_fs:
	cd pkg && make clean

