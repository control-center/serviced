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

VERSION := $(shell cat ./VERSION)

.PHONY: default
default: dockerBuild

dockerBuild: reporterJar
	docker build -t zenoss/capybara:$(VERSION) build

reporterJar: build/reporter.jar

build/reporter.jar: reporter/target/reporter.jar

reporter/target/reporter.jar:
	cd reporter && mvn -s ./settings.xml package
	cp reporter/target/reporter.jar build/reporter.jar

dockerPush:
	docker push zenoss/capybara:$(VERSION)
