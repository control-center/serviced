FROM ubuntu:12.04
MAINTAINER Zenoss <dev@zenoss.com>

RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get update
RUN apt-get upgrade -y

##
# Fake a fuse install -- necessary for openjdk-7-jdk
#  https://gist.github.com/henrik-muehe/6155333
RUN apt-get install libfuse2
RUN cd /tmp ; apt-get download fuse; dpkg-deb -x fuse_* . ; dpkg-deb -e fuse_* ; rm fuse_*.deb ; echo -en '#!/bin/bash\nexit 0\n' > DEBIAN/postinst ; dpkg-deb -b . /fuse.deb ; dpkg -i /fuse.deb

# Install Packages required to run opentsdb
RUN apt-get install -y -q openjdk-7-jdk git autoconf build-essential libtool gnuplot wget supervisor
ENV COMPRESSION NONE
ENV HBASE_HOME /opt/hbase-0.94.14
ENV JAVA_HOME /usr/lib/jvm/java-7-openjdk-amd64

RUN apt-get -y install maven
