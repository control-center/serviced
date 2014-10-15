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
