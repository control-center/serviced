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

GO = $(shell which go)
UID = $(shell id -u)
GID = $(shell id -g)
PWD = $(shell pwd)

javascript_TARGETS       = $(controlplane_JS) $(controlplane_MINIFIED_JS)
controlplane_JS          = $(javascript_BLDDIR)/controlplane.js
controlplane_MINIFIED_JS = $(javascript_BLDDIR)/controlplane.min.js

.PHONY: default
default: build

.PHONY: build 
build: build_js
	./godep restore
	go build

.PHONY: test
test: build
	go test -i ;\
	go test $(GOTEST_FLAGS)

.PHONY: build_js build-js
build_js build-js:
	docker pull zenoss/gulp
	docker run --rm -v $(PWD)/:/mnt/ -e UID=$(UID) -e GID=$(GID) zenoss/gulp npm install
	docker run --rm -v $(PWD)/:/mnt/ -e UID=$(UID) -e GID=$(GID) zenoss/gulp gulp release

.PHONY: clean
clean:
	if [ ! -z "$(GO)" ];then \
		$(GO) clean ;\
	fi
	for target in $(javascript_TARGETS) ;\
	do \
		if [ -f "$${target}" ];then \
			rm -f $${target} ;\
		fi ;\
	done
