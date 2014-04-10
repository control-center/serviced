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
	cd key && go build
	cd driver && go build
	cd context && go build
	cd elastic && go build
	go build

test: build
	cd key && go test
	cd driver && go test
	cd context && go test
	cd elastic && go test
	cd integration_test && go test
	go test

clean:
	cd key && go clean
	cd driver && go clean
	cd context && go clean
	cd elastic && go clean
	go clean
