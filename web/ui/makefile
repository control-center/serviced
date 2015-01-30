# Copyright 2015 The Serviced Authors.
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

UID = $(shell id -u)
GID = $(shell id -g)
PWD = $(shell pwd)

serviced_build_DOCKERFILE = ../../build
javascript_BLDDIR   = $(PWD)/build/js
controlplane_JS     = $(javascript_BLDDIR)/controlplane.js

NODEJS = $(shell which node)
docker_working_DIR = "/mnt"


.PHONY: default
default: build

.PHONY: build
build: $(controlplane_JS)

$(controlplane_JS): src/*.js src/**/*.js
	if [ -x "$(NODEJS)" ]; then \
		npm install; \
		gulp release; \
	else \
		docker build -t zenoss/serviced-build $(serviced_build_DOCKERFILE); \
		docker run --rm \
		-v $(PWD):$(docker_working_DIR) \
		-e UID_X=$(shell id -u) \
		-e GID_X=$(shell id -g) \
		zenoss/serviced-build \
		/bin/bash -c "source /root/userdo.sh \"cd $(docker_working_DIR) && npm install && gulp release\""; \
	fi

.PHONY: test
test: build
	if [ -x "$(NODEJS)" ]; then \
		npm install; \
		gulp test; \
	else \
		docker build -t zenoss/serviced-build $(serviced_build_DOCKERFILE); \
		docker run --rm \
		-v $(PWD):$(docker_working_DIR) \
		-e UID_X=$(shell id -u) \
		-e GID_X=$(shell id -g) \
		zenoss/serviced-build \
		/bin/bash -c "source /root/userdo.sh \"cd $(docker_working_DIR) && npm install && gulp test\""; \
	fi

.PHONY: clean
clean:
	rm -rf build
	rm -rf test/results
