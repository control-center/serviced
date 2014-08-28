# Copyright 2014, The Serviced Authors. All rights reserved.
# Use of this source code is governed by the Apache 2.0
# license that can be found in the LICENSE file.

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
