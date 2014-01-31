TARGET := $(shell pwd)
HBASE_VERSION := 0.94.14

default: tarball

tarball:
	wget -qO- http://www.apache.org/dist/hbase/hbase-$(HBASE_VERSION)/hbase-$(HBASE_VERSION).tar.gz | tar -C /opt -xz
	git clone git://github.com/OpenTSDB/opentsdb.git /opt/opentsdb
	cd /opt/opentsdb && COMPRESSION=NONE HBASE_HOME=/opt/hbase-$(HBASE_VERSION) JAVA_HOME=/usr/lib/jvm/java-7-openjdk-amd64 ./build.sh
	cp /tmp/in/hbase-site.xml /opt/hbase-$(HBASE_VERSION)/conf/hbase-site.xml
	cp /tmp/in/start-opentsdb.sh /opt/opentsdb/build/start-opentsdb.sh && chmod a+x /opt/opentsdb/build/start-opentsdb.sh
	mkdir -p /opt/zenoss/etc/supervisor
	cp /tmp/in/opentsdb_supervisor.conf /opt/zenoss/etc/supervisor/opentsdb_supervisor.conf
	tar czf $(TARGET)/opentsdb.tar.gz /opt
