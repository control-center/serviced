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
	go build
	cd host && go build
	cd pool && go build
	cd servicedefinition && go build
	cd servicetemplate && go build
	cd service && go build
	cd servicestate && go build

test: build
	go test $(GOTEST_FLAGS)
	cd host && go test $(GOTEST_FLAGS)
	cd pool && go test $(GOTEST_FLAGS)
	cd servicedefinition && go test $(GOTEST_FLAGS)
	cd servicetemplate && go test $(GOTEST_FLAGS)
	cd service && go test $(GOTEST_FLAGS)
	cd servicestate && go test $(GOTEST_FLAGS)

clean:
	go clean
	cd host && go clean
	cd pool && go clean
	cd servicedefinition && go clean
	cd servicetemplate && go clean
	cd service && go clean
	cd servicestate && go clean
