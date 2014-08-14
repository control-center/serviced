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
QUERY_VERSION := 0.0.8

.PHONY: default tarball

default: tarball

tarball:
	mkdir /opt/zenoss && mkdir /opt/zenoss/log && mkdir -p /opt/zenoss/etc/supervisor && mkdir /opt/zenoss/var
	wget -qO- https://s3.amazonaws.com/pip.zenoss/packages/central-query-$(QUERY_VERSION)-zapp.tar.gz | tar -C /opt/zenoss -xz
	chmod a+x /opt/zenoss/bin/central-query.sh
	ln -s /opt/zenoss/etc/central-query/central-query_supervisor.conf /opt/zenoss/etc/supervisor
	sed -i 's/\(authEnabled:.*\)true/\1false/' /opt/zenoss/etc/central-query/configuration.yaml
	tar czf $(TARGET)/query.tar.gz /opt
