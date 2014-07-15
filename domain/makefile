# Copyright 2014, The Serviced Authors. All rights reserved.
# Use of this source code is governed by the Apache 2.0
# license that can be found in the LICENSE file.

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
	cd addressassignment && go build
	cd serviceconfigfile && go build

test: build
	go test $(GOTEST_FLAGS)
	cd host && go test $(GOTEST_FLAGS)
	cd pool && go test $(GOTEST_FLAGS)
	cd servicedefinition && go test $(GOTEST_FLAGS)
	cd servicetemplate && go test $(GOTEST_FLAGS)
	cd service && go test $(GOTEST_FLAGS)
	cd servicestate && go test $(GOTEST_FLAGS)
	cd addressassignment && go test $(GOTEST_FLAGS)
	cd serviceconfigfile && go test $(GOTEST_FLAGS)

clean:
	go clean
	cd host && go clean
	cd pool && go clean
	cd servicedefinition && go clean
	cd servicetemplate && go clean
	cd service && go clean
	cd servicestate && go clean
	cd addressassignment && go clean
	cd serviceconfigfile && go clean
