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

TARGET := $(shell pwd)
ZK_VERSION := 3.4.5

.PHONY: default tarball

default: tarball

tarball:
	wget -qO- http://apache.cs.utah.edu/zookeeper/zookeeper-$(ZK_VERSION)/zookeeper-$(ZK_VERSION).tar.gz | tar -C /opt -xz
	cp /opt/zookeeper-$(ZK_VERSION)/conf/zoo_sample.cfg /opt/zookeeper-$(ZK_VERSION)/conf/zoo.cfg
	echo maxClientCnxns=0 >> /opt/zookeeper-$(ZK_VERSION)/conf/zoo.cfg
	mkdir /opt/exhibitor && \
		wget -q -O /opt/exhibitor/pom.xml https://raw.github.com/Netflix/exhibitor/master/exhibitor-standalone/src/main/resources/buildscripts/standalone/maven/pom.xml && \
		cd /opt/exhibitor && \
		mvn assembly:single
	sed -i 's|start-foreground)|start-foreground)\n /usr/bin/java -jar /opt/exhibitor/target/exhibitor-*-jar-with-dependencies.jar -c file --port 12181 \&|g' /opt/zookeeper-$(ZK_VERSION)/bin/zkServer.sh
	tar czf $(TARGET)/zk.tar.gz /opt
