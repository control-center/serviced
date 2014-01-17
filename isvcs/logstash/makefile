TARGET := /tmp/out
LOGSTASH_VERSION := 1.3.2

default: tarball

tarball:
	@mkdir /opt/logstash
	@wget -q -O /opt/logstash/logstash-$(LOGSTASH_VERSION)-flatjar.jar https://download.elasticsearch.org/logstash/logstash/logstash-$(LOGSTASH_VERSION)-flatjar.jar
	@tar czf $(TARGET)/logstash.tar.gz /opt
