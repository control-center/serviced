TARGET := $(shell pwd)
QUERY_VERSION := 0.0.2

.PHONY: default tarball

default: tarball

tarball:
	mkdir /opt/zenoss && mkdir /opt/zenoss/log && mkdir -p /opt/zenoss/etc/supervisor && mkdir /opt/zenoss/var
	wget -qO- https://s3.amazonaws.com/pip.zenoss/packages/central-query-$(QUERY_VERSION)-zapp.tar.gz | tar -C /opt/zenoss -xz
	chmod a+x /opt/zenoss/bin/central-query.sh
	ln -s /opt/zenoss/etc/central-query/central-query_supervisor.conf /opt/zenoss/etc/supervisor
	sed -i 's/\(authEnabled:.*\)true/\1false/' /opt/zenoss/etc/central-query/configuration.yaml
	tar czf $(TARGET)/query.tar.gz /opt
