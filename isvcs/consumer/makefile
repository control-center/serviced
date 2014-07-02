TARGET := $(shell pwd)
CONSUMER_VERSION := 0.0.8

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
