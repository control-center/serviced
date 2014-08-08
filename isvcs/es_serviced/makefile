# Copyright 2014, The Serviced Authors. All rights reserved.
# Use of this source code is governed by the Apache 2.0
# license that can be found in the LICENSE file.

TARGET := $(shell pwd)
ES_VERSION :=  0.90.9

.PHONY: default tarball

default: tarball

tarball:
	wget -qO- https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-$(ES_VERSION).tar.gz | tar -C /opt -xz
	/opt/elasticsearch-$(ES_VERSION)/bin/plugin -install mobz/elasticsearch-head
	tar czf $(TARGET)/es_serviced.tar.gz /opt
