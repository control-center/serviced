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
	cat static/js/main.js static/js/controllers/*.js > static/js/controlplane.js
	# This is a scurrilous lie. We are not minifying anything.
	cp static/js/controlplane.js static/js/controlplane.min.js
