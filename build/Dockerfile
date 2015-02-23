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

# Please be careful when changing the 'FROM' line below - the packaging code
# uses the output of 'lsb_release' to determine the version string of the
# .deb pacakge, and if the base image changes here, then that string will
# change accordingly.
FROM ubuntu:trusty

MAINTAINER Zenoss, Inc <dev@zenoss.com>	

RUN	apt-get update -qq
RUN	apt-get install -qqy iptables ca-certificates aufs-tools

# Build dependencies
#RUN	dpkg-divert --local --rename --add /sbin/initctl
#RUN	ln -s /bin/true /sbin/initctl
RUN	apt-get update -qq && apt-get install -y -q wget curl git

# Install Go
RUN	curl -s https://storage.googleapis.com/golang/go1.3.3.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV	GOPATH  /go
ENV	PATH $PATH:/go/bin:/usr/local/go/bin

# build dependencies
RUN	apt-get update -qq && apt-get install -y -q make gcc libpam0g-dev ruby ruby-dev
RUN	gem install fpm
RUN	apt-get update -qq && apt-get install -y -q rpm
RUN	apt-get update -qq && apt-get install -y -q mercurial bzr
RUN	apt-get update -qq && apt-get install -y -q libdevmapper-dev libsqlite3-dev

WORKDIR	/go/src/github.com/control-center/serviced
