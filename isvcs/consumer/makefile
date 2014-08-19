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
CONSUMER_VERSION := 0.0.9

.PHONY: default tarball

default: tarball

tarball:
	@mkdir -p /opt/zenoss && mkdir -p /opt/zenoss/log && mkdir -p /opt/zenoss/etc/supervisor && mkdir /opt/zenoss/var
	@wget -qO- https://s3.amazonaws.com/pip.zenoss/packages/metric-consumer-app-$(CONSUMER_VERSION)-zapp.tar.gz | tar -C /opt/zenoss -xz
	@chmod a+x /opt/zenoss/bin/metric-consumer-app.sh
	@ln -s /opt/zenoss/etc/metric-consumer-app/metric-consumer-app_supervisor.conf /opt/zenoss/etc/supervisor
	@sed -i 's/\(authEnabled:.*\)true/\1false/' /opt/zenoss/etc/metric-consumer-app/configuration.yaml
	@echo "httpParameterTags:" >> /opt/zenoss/etc/metric-consumer-app/configuration.yaml
	@echo "  - controlplane" >> /opt/zenoss/etc/metric-consumer-app/configuration.yaml
	@echo "tagWhiteList:" >> /opt/zenoss/etc/metric-consumer-app/configuration.yaml
	@echo "  - controlplane_service_id" >> /opt/zenoss/etc/metric-consumer-app/configuration.yaml
	@echo "  - controlplane_tenant_id" >> /opt/zenoss/etc/metric-consumer-app/configuration.yaml
	@echo "  - controlplane_host_id" >> /opt/zenoss/etc/metric-consumer-app/configuration.yaml
	@echo "  - controlplane_instance_id" >> /opt/zenoss/etc/metric-consumer-app/configuration.yaml
	@tar czf $(TARGET)/consumer.tar.gz /opt
