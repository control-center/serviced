FROM ubuntu:12.04
MAINTAINER Zenoss <dev@zenoss.com>

RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get install -y -q openjdk-7-jre-headless wget
ENV JAVA_HOME /usr/lib/jvm/java-7-openjdk-amd64

RUN wget -q -O elasticsearch-0.90.9.tar.gz https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-0.90.9.tar.gz 
RUN tar xvfz elasticsearch-0.90.9.tar.gz -C /opt && cd /opt/elasticsearch-0.90.9 && ./bin/plugin -install mobz/elasticsearch-head

ENTRYPOINT ["/opt/elasticsearch-0.90.9/bin/elasticsearch"]
CMD ["-f"]
EXPOSE 9200

