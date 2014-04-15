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
	cd elastic && go build
	go build

test: build
	cd elastic && go test
	cd integration_test && go test
	go test

clean:
	cd elastic && go clean
	go clean
