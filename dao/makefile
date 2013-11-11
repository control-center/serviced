################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

.PHONY: build test clean

default: build

build:
	cd elasticsearch && make build
	go build

test: build
	#cd elasticsearch && make test
	go test

clean:
	cd elasticsearch && make clean
	go clean
