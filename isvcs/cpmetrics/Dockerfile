FROM ubuntu:12.04
MAINTAINER Zenoss <dev@zenoss.com>

RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get update
RUN apt-get upgrade -y

##
# Fake a fuse install -- necessary for openjdk-7-jdk
#  https://gist.github.com/henrik-muehe/6155333
RUN apt-get install libfuse2
RUN cd /tmp ; apt-get download fuse
RUN cd /tmp ; dpkg-deb -x fuse_* .
RUN cd /tmp ; dpkg-deb -e fuse_*
RUN cd /tmp ; rm fuse_*.deb
RUN cd /tmp ; echo -en '#!/bin/bash\nexit 0\n' > DEBIAN/postinst
RUN cd /tmp ; dpkg-deb -b . /fuse.deb
RUN cd /tmp ; dpkg -i /fuse.deb

# Install Packages required to run opentsdb
RUN apt-get install -y -q openjdk-7-jdk git autoconf build-essential libtool gnuplot wget 
RUN update-alternatives --install "/usr/bin/java" "java" "/usr/lib/jvm/java-7-openjdk-amd64/bin/java" 1
RUN update-alternatives --set "java" "/usr/lib/jvm/java-7-openjdk-amd64/bin/java"

RUN wget http://www.apache.org/dist/hbase/hbase-0.94.14/hbase-0.94.14.tar.gz
RUN tar xzf hbase-0.94.14.tar.gz -C /opt

ENV COMPRESSION NONE
ENV HBASE_HOME /opt/hbase-0.94.14
ENV JAVA_HOME /usr/lib/jvm/java-7-openjdk-amd64

# Build and Configure OpenTsdb
RUN git clone git://github.com/OpenTSDB/opentsdb.git /opt/opentsdb
RUN cd /opt/opentsdb && ./build.sh

VOLUME /tmp
VOLUME /opt/hbase-0.94.14/logs

# Configure hbase (xml file's base64 encoded hbase-site.xml)
ADD hbase-site.xml /opt/hbase-0.94.14/conf/hbase-site.xml
ADD start-opentsdb.sh /opt/opentsdb/build/start-opentsdb.sh
RUN chmod a+x /opt/opentsdb/build/start-opentsdb.sh

RUN apt-get install -y -q maven

RUN git clone -b develop http://github.com/zenoss/zenoss.metric.consumer /var/tmp/zenoss.metric.consumer
ADD settings.xml /var/tmp/settings.xml
RUN cd /var/tmp/zenoss.metric.consumer && mvn --settings /var/tmp/settings.xml -DskipTests package -P assemble
RUN mkdir /opt/zenoss && mkdir /opt/zenoss/log && mkdir -p /opt/zenoss/etc/supervisor
RUN tar zxf /var/tmp/zenoss.metric.consumer/metric-consumer-app/target/metric-consumer-app-*-SNAPSHOT-zapp.tar.gz -C /opt/zenoss
RUN chmod a+x /opt/zenoss/bin/metric-consumer-app.sh

EXPOSE 4242 8443 9090 60000 60010 60020 60030

CMD ["/opt/opentsdb/build/start-opentsdb.sh"]
