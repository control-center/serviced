################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

default: build

build: build-js
	./godep restore
	go build


test: build
	go test -i
	go test $(GOTEST_FLAGS)

clean:
	go clean

build-js:
	sed -e "s/%UID%/$$(id -u)/g" -e "s/%GID%/$$(id -g)/g" Dockerfile.in | docker build -t zenoss/grunt -
	docker run --rm -v $$PWD:/mnt/build -w /mnt/build zenoss/grunt 

