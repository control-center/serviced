# This file describes the standard way to build serviced, using docker
#
# Usage:
#
# # Assemble the full dev environment. This is slow the first time.
# docker build -t docker .
# # Apparmor messes with privileged mode: disable it
# /etc/init.d/apparmor stop ; /etc/init.d/apparmor teardown
#
# # Mount your source in an interactive container for quick testing:
# docker run -v `pwd`:/go/src/github.com/zenoss/serviced -privileged -lxc-conf=lxc.aa_profile=unconfined -i -t serviced bash
#
#
FROM ubuntu:saucy
MAINTAINER Zenoss, Inc <dev@zenoss.com>	

RUN	echo deb http://archive.ubuntu.com/ubuntu precise universe > /etc/apt/sources.list.d/universe.list
RUN	apt-get update -qq
RUN	apt-get install -qqy iptables ca-certificates lxc aufs-tools

# Build dependencies
#RUN	dpkg-divert --local --rename --add /sbin/initctl
#RUN	ln -s /bin/true /sbin/initctl
RUN	apt-get install -y -q wget curl git

# Install Go
RUN	curl -s http://go.googlecode.com/files/go1.2.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV	GOPATH  /go
ENV	PATH $PATH:/go/bin:/usr/local/go/bin

# build dependencies
RUN	apt-get install -y -q make gcc libpam0g-dev ruby ruby-dev
RUN	apt-get	install -y -q rubygems
RUN	gem install fpm
RUN	apt-get install -y -q rpm
RUN	apt-get install -y -q mercurial bzr
RUN apt-get install -y -q libdevmapper-dev libsqlite3-dev

# install docker
RUN	wget -q -O /usr/local/bin/docker https://get.docker.io/builds/Linux/x86_64/docker-latest
RUN	wget -q -O /usr/local/bin/wrapdocker https://raw2.github.com/jpetazzo/dind/master/wrapdocker
RUN chmod +x /usr/local/bin/docker /usr/local/bin/wrapdocker

WORKDIR	/go/src/github.com/zenoss/serviced
CMD	wrapdocker
