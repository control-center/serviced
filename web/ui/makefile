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

VERSION := $(shell cat ../../VERSION)

UID = $(shell id -u)
GID = $(shell id -g)
PWD = $(shell pwd)

serviced_build_DOCKERDIR = ../../build
javascript_BLDDIR   = $(PWD)/build/js
controlplane_JS     = $(javascript_BLDDIR)/controlplane.js

NODEJS = $(shell which node)
docker_working_DIR = /mnt

.PHONY: default
default: build

.PHONY: build
build: npmInstall $(controlplane_JS)

#
# If you have NPM installed locally, this step will refresh the NPM dependencies.
# If you are using the serviced-build image to build the UI AND you need
# to refresh the NPM dependencies, then run the following commands:
# 'make buildDockerImage;make pushDockerImage' to
# rebuild and publish zenoss/serviced-build with the NPM dependencies.
#
npmInstall:
	if [ -x "$(NODEJS)" ]; then \
		./npmLink.sh; \
		npm install; \
	fi

$(controlplane_JS): src/*.js src/**/*.js
	if [ -x "$(NODEJS)" ]; then \
		gulp release; \
	else \
		docker run --rm \
		-v $(PWD):$(docker_working_DIR) \
		-e UID_X=$(shell id -u) \
		-e GID_X=$(shell id -g) \
		zenoss/serviced-build:$(VERSION) \
		/bin/bash -c "source /root/userdo.sh \"cd $(docker_working_DIR) && ./npmLink.sh && gulp release\""; \
	fi

.PHONY: test
test: build
	if [ -x "$(NODEJS)" ]; then \
		gulp test; \
	else \
		docker run --rm \
		-v $(PWD):$(docker_working_DIR) \
		-e UID_X=$(shell id -u) \
		-e GID_X=$(shell id -g) \
		zenoss/serviced-build:$(VERSION) \
		/bin/bash -c "source /root/userdo.sh \"cd $(docker_working_DIR) && ./npmLink.sh && gulp test\""; \
	fi

.PHONY: clean
clean:
	rm -rf build
	rm -rf test/results
