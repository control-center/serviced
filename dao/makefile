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
	cd elasticsearch && make "GO=$(GO)" build
	$(GO) build

test: build
	cd elasticsearch && make "GO=$(GO)" test
	$(GO) test $(GOTEST_FLAGS)

clean:
	cd elasticsearch && make "GO=$(GO)" clean
	$(GO) clean
