################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

default: build

install:
	go install github.com/serviced/serviced

build:
	go get github.com/zenoss/serviced/serviced
	cd serviced && go build

pkgs:
	cd pkg && make rpm && make deb


dockerbuild: docker_ok
	docker build -t zenoss/serviced-build .
	docker run -v `pwd`:/go/src/github.com/zenoss/serviced -e BUILD_NUMBER=$(BUILD_NUMBER) -t zenoss/serviced-build make clean pkgs

test: build docker_ok
	go test
	cd web && go test
	cd dao && go test
	cd serviced && go test

docker_ok:
	if docker ps >/dev/null; then \
		echo "docker OK"; \
	else \
		echo "Check 'docker ps' command"; \
		exit 1;\
	fi

clean:
	go get github.com/zenoss/serviced/serviced # make sure dependencies exist
	cd serviced && go clean -r # this cleans all dependencies

