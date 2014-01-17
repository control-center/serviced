TARGET := $(shell pwd)
ES_VERSION := 0.90.9

.PHONY: default tarball

default: tarball

tarball:
	@wget -qO- https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-$(ES_VERSION).tar.gz | tar -C /opt -xz
	@/opt/elasticsearch-$(ES_VERSION)/bin/plugin -install mobz/elasticsearch-head
	@tar czf $(TARGET)/es.tar.gz /opt
