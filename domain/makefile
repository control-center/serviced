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
	cd host && go build
	cd pool && go build

test: build
	cd host && go test
	cd pool && go test

clean:
	cd host && go clean
	cd pool && go clean
