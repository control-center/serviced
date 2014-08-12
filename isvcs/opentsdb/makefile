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
HBASE_VERSION := 0.94.16

default: tarball

tarball:
	wget -qO- https://s3.amazonaws.com/pip.zenoss/binary_mirrors/hbase-$(HBASE_VERSION).tar.gz | tar -C /opt -xz
	git clone git://github.com/OpenTSDB/opentsdb.git /opt/opentsdb
	cd /opt/opentsdb && COMPRESSION=NONE HBASE_HOME=/opt/hbase-$(HBASE_VERSION) JAVA_HOME=/usr/lib/jvm/java-7-openjdk-amd64 ./build.sh
	cp /tmp/in/hbase-site.xml /opt/hbase-$(HBASE_VERSION)/conf/hbase-site.xml
	cp /tmp/in/start-opentsdb.sh /opt/opentsdb/build/start-opentsdb.sh && chmod a+x /opt/opentsdb/build/start-opentsdb.sh
	mkdir -p /opt/zenoss/etc/supervisor
	cp /tmp/in/opentsdb_supervisor.conf /opt/zenoss/etc/supervisor/opentsdb_supervisor.conf
	tar czf $(TARGET)/opentsdb.tar.gz /opt
