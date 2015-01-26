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

javascript_BLDDIR        = $(PWD)/static/js
controlplane_JS          = $(javascript_BLDDIR)/controlplane.js
controlplane_MINIFIED_JS = $(javascript_BLDDIR)/controlplane.min.js
javascript_3rdparty_BLDDIR = $(PWD)/static/thirdparty
thirdparty_JS              = $(javascript_3rdparty_BLDDIR)/thirdparty.js
thirdparty_MINIFIED_JS     = $(javascript_3rdparty_BLDDIR)/thirdparty.min.js
javascript_TARGETS = $(controlplane_JS) $(controlplane_MINIFIED_JS) $(thirdparty_JS) $(thirdparty_MINIFIED_JS)

NODEJS = $(shell which node)
docker_working_DIR = "/mnt"

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
	if [ -x "$(NODEJS)" ]; then \
		npm install; \
		gulp release; \
	else \
		docker build -t zenoss/serviced-build ../build; \
		docker run --rm \
		-v $(PWD):$(docker_working_DIR) \
		-e UID_X=$(shell id -u) \
		-e GID_X=$(shell id -g) \
		zenoss/serviced-build \
		/bin/bash -c "source /root/userdo.sh \"cd $(docker_working_DIR) && npm install && gulp release\""; \
	fi

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
