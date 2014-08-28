# Copyright 2014, The Serviced Authors. All rights reserved.
# Use of this source code is governed by the Apache 2.0
# license that can be found in the LICENSE file.

TARGET := $(shell pwd)
ES_VERSION := 1.3.1

.PHONY: default tarball

default: tarball

tarball:
	wget -qO- https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-$(ES_VERSION).tar.gz | tar -C /opt -xz
	/opt/elasticsearch-$(ES_VERSION)/bin/plugin -install mobz/elasticsearch-head
	sed -i -e 's/^.*http.port.*$$/http.port: 9100/' /opt/elasticsearch-$(ES_VERSION)/config/elasticsearch.yml
	tar czf $(TARGET)/es_logstash.tar.gz /opt
