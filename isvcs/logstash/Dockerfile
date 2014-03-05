FROM ubuntu:latest
MAINTAINER jhanson@zenoss.com

RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get -y update
RUN apt-get -y upgrade

ENV DEBIAN_FRONTEND noninteractive

# Fake a fuse install
RUN apt-get install libfuse2
RUN cd /tmp ; apt-get download fuse && dpkg-deb -x fuse_* . && dpkg-deb -e fuse_* &&  rm fuse_*.deb
RUN cd /tmp ; echo -en '#!/bin/bash\nexit 0\n' > DEBIAN/postinst
RUN cd /tmp ; dpkg-deb -b . /fuse.deb && dpkg -i /fuse.deb

RUN apt-get -y install openjdk-7-jre-headless wget
RUN update-alternatives --install "/usr/bin/java" "java" "/usr/lib/jvm/java-7-openjdk-amd64/bin/java" 1
RUN update-alternatives --set "java" "/usr/lib/jvm/java-7-openjdk-amd64/bin/java"

RUN mkdir /root/logstash
RUN cd /root/logstash && wget https://download.elasticsearch.org/logstash/logstash/logstash-1.3.2-flatjar.jar

# listen for logstash requests
EXPOSE 5042

# listen for logstash-forwarder requests
EXPOSE 5043

# web requests
EXPOSE 9292

# /usr/local/serviced is hardcoded in isvcs.go
ENTRYPOINT java -jar /root/logstash/logstash-1.3.2-flatjar.jar agent -f /usr/local/serviced/resources/logstash/logstash.conf -- web
