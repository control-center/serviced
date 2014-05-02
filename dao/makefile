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
	cd tests && go build
	go build

test: build
	cd elasticsearch && make test
	cd tests && go test $(GOTEST_FLAGS)
	go test $(GOTEST_FLAGS)

clean:
	cd elasticsearch && make clean
	cd tests && go clean
	go clean
