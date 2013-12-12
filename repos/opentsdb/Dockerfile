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

RUN wget http://www.apache.org/dist/hbase/hbase-0.94.14/hbase-0.94.14.tar.gz
RUN tar xzf hbase-0.94.14.tar.gz -C /opt

ENV COMPRESSION NONE
ENV HBASE_HOME /opt/hbase-0.94.14
ENV JAVA_HOME /usr/lib/jvm/java-7-openjdk-amd64

# Build and Configure OpenTsdb
RUN git clone git://github.com/OpenTSDB/opentsdb.git /opt/opentsdb
RUN cd /opt/opentsdb && ./build.sh
EXPOSE 4242
CMD /opt/opentsdb/build/start-opentsdb.sh
VOLUME /tmp
VOLUME /opt/hbase-0.94.14/logs/

# Configure hbase (xml file's base64 encoded hbase-site.xml)
ADD hbase-site.xml /opt/hbase-0.94.14/conf/hbase-site.xml
ADD start-opentsdb.sh /opt/opentsdb/build/start-opentsdb.sh



