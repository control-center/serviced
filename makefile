################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

UID := $(shell id -u)
GID := $(shell id -g)

pwdchecksum := $(shell pwd | md5sum | awk '{print $$1}')
dockercache := /tmp/serviced-dind-$(pwdchecksum)

default: build_binary

install: build_binary
	go install github.com/zenoss/serviced/serviced

build_binary:
	cd serviced && make
	cd isvcs && make

go:
	cd serviced && go build


pkgs:
	cd pkg && make rpm && make deb


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
		-e BUILD_NUMBER=$(BUILD_NUMBER) \
		-t zenoss/serviced-build /bin/bash \
		-c "/usr/local/bin/wrapdocker && make build_binary pkgs; \
			chown -R $(UID):$(GID) serviced/serviced pkg/pkgroot \
			pkg/*.deb pkg/*.rpm pkg/build"

test: build_binary docker_ok
	go test
	go get github.com/onsi/ginkgo
	go get github.com/onsi/gomega
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

