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
	cd agent && go build
	cd master && go build

test: build
	cd agent && go test $(GOTEST_FLAGS)
	cd master && go test $(GOTEST_FLAGS)

clean:
	cd agent && go clean
	cd master && go clean
