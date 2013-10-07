package isvcs

var OpenTsdbContainer ISvc

const opentsdb_dockerfile = `
FROM ubuntu
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

ADD http://www.apache.org/dist/hbase/hbase-0.94.12/hbase-0.94.12.tar.gz /opt/
RUN tar xzf /opt/hbase-0.94.12.tar.gz -C /opt

ENV COMPRESSION NONE
ENV HBASE_HOME /opt/hbase-0.94.12
ENV JAVA_HOME /usr/lib/jvm/java-7-openjdk-amd64

# Configure hbase (xml file's base64 encoded hbase-site.xml)
RUN bash -c "echo PD94bWwgdmVyc2lvbj0iMS4wIj8+Cjw/eG1sLXN0eWxlc2hlZXQgdHlwZT0idGV4dC94c2wiIGhyZWY9ImNvbmZpZ3VyYXRpb24ueHNsIj8+Cjxjb25maWd1cmF0aW9uPgogIDxwcm9wZXJ0eT4KICAgIDxuYW1lPmhiYXNlLnJvb3RkaXI8L25hbWU+CiAgICA8dmFsdWU+L3RtcC9oYmFzZTwvdmFsdWU+CiAgPC9wcm9wZXJ0eT4KICA8cHJvcGVydHk+CiAgICA8bmFtZT5oYmFzZS56b29rZWVwZXIuZG5zLmludGVyZmFjZTwvbmFtZT4KICAgIDx2YWx1ZT5sbzwvdmFsdWU+CiAgPC9wcm9wZXJ0eT4KICA8cHJvcGVydHk+CiAgICA8bmFtZT5oYmFzZS5yZWdpb25zZXJ2ZXIuZG5zLmludGVyZmFjZTwvbmFtZT4KICAgIDx2YWx1ZT5sbzwvdmFsdWU+CiAgPC9wcm9wZXJ0eT4KICA8cHJvcGVydHk+CiAgICA8bmFtZT5oYmFzZS5tYXN0ZXIuZG5zLmludGVyZmFjZTwvbmFtZT4KICAgIDx2YWx1ZT5sbzwvdmFsdWU+CiAgPC9wcm9wZXJ0eT4KPC9jb25maWd1cmF0aW9uPgo= | base64 -d > /opt/hbase-0.94.12/conf/hbase-site.xml"

# Build and Configure OpenTsdb
RUN git clone git://github.com/OpenTSDB/opentsdb.git /opt/opentsdb
RUN cd /opt/opentsdb && ./build.sh
RUN mkdir -p /tmp/tsd

EXPOSE 4242:4242

# Start an Hbase cluster, wait for master to initialize, configure opentsdb tables, start opentsdb
CMD bash -c "/opt/hbase-0.94.12/bin/start-hbase.sh && { sleep 10s; /opt/opentsdb/src/create_table.sh; /opt/opentsdb/build/tsdb tsd --port=4242 --staticroot=/opt/opentsdb/build/staticroot --cachedir=/tmp/tsd; }"
`

func init() {
	OpenTsdbContainer = ISvc{
		Name:       "opentsdb",
		Dockerfile: opentsdb_dockerfile,
		Tag:        "zenoss/cp_opentsdb",
	}
}
