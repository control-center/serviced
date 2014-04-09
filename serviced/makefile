################################################################################
#
# Copyright (C) Zenoss, Inc. 2013, all rights reserved.
#
# This content is made available according to terms specified in
# License.zenoss under the directory where your Zenoss product is installed.
#
################################################################################

# nsinit is a utility to enter containers. Godep restore should place the source
# at the directory given below.
NSINITDIR=../../../dotcloud/docker/pkg/libcontainer/nsinit/nsinit

default: build

install:
	go install

build:
	./godep restore
	go build
	cd $(NSINITDIR) && go build && go install
	cp $(NSINITDIR)/nsinit .

test: build
	go test -i
	go test

clean:
	go clean
	cd $(NSINITDIR) && go clean
	rm -Rf nsinit

