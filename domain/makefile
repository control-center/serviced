# Copyright 2014 The Serviced Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO ?= $(shell which go)

.PHONY: build test clean

default: build

build:
	$(GO) build
	cd host && $(GO) build
	cd pool && $(GO) build
	cd servicedefinition && $(GO) build
	cd servicetemplate && $(GO) build
	cd service && $(GO) build
	cd servicestate && $(GO) build
	cd addressassignment && $(GO) build
	cd serviceconfigfile && $(GO) build

test: build
	$(GO) test $(GOTEST_FLAGS)
	cd host && $(GO) test $(GOTEST_FLAGS)
	cd pool && $(GO) test $(GOTEST_FLAGS)
	cd servicedefinition && $(GO) test $(GOTEST_FLAGS)
	cd servicetemplate && $(GO) test $(GOTEST_FLAGS)
	cd service && $(GO) test $(GOTEST_FLAGS)
	cd servicestate && $(GO) test $(GOTEST_FLAGS)
	cd addressassignment && $(GO) test $(GOTEST_FLAGS)
	cd serviceconfigfile && $(GO) test $(GOTEST_FLAGS)

clean:
	$(GO) clean
	cd host && $(GO) clean
	cd pool && $(GO) clean
	cd servicedefinition && $(GO) clean
	cd servicetemplate && $(GO) clean
	cd service && $(GO) clean
	cd servicestate && $(GO) clean
	cd addressassignment && $(GO) clean
	cd serviceconfigfile && $(GO) clean
