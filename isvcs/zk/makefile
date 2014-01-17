TARGET := $(shell pwd)
ZK_VERSION := 3.4.5

.PHONY: default tarball

default: tarball

tarball:
	@wget -qO- http://apache.cs.utah.edu/zookeeper/zookeeper-$(ZK_VERSION)/zookeeper-$(ZK_VERSION).tar.gz | tar -C /opt -xz
	@cp /opt/zookeeper-$(ZK_VERSION)/conf/zoo_sample.cfg /opt/zookeeper-$(ZK_VERSION)/conf/zoo.cfg
	@mkdir /opt/exhibitor && \
		wget -q -O /opt/exhibitor/pom.xml https://raw.github.com/Netflix/exhibitor/master/exhibitor-standalone/src/main/resources/buildscripts/standalone/maven/pom.xml && \
		cd /opt/exhibitor && \
		mvn assembly:single
	@sed -i 's|start-foreground)|start-foreground)\n /usr/bin/java -jar /opt/exhibitor/target/exhibitor-*-jar-with-dependencies.jar -c file --port 12181 \&|g' /opt/zookeeper-$(ZK_VERSION)/bin/zkServer.sh
	@tar czf $(TARGET)/zk.tar.gz /opt
