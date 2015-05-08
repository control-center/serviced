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

.PHONY: build test clean

GO        ?= $(shell which go)

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
