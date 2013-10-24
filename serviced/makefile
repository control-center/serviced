################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

default: build

build:
	go get github.com/ant0ine/go-json-rest
	go build

test: build
	go test -i
	go test

clean:
	go clean

