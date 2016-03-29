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

RUN	apt-get update -qq && apt-get install -qqy iptables ca-certificates aufs-tools

# Build dependencies
#RUN	dpkg-divert --local --rename --add /sbin/initctl
#RUN	ln -s /bin/true /sbin/initctl
RUN	apt-get update -qq && apt-get install -y -q wget curl git

# Install Go
RUN	curl -s https://storage.googleapis.com/golang/go1.6.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV	GOPATH  /go
ENV	PATH $PATH:/go/bin:/usr/local/go/bin
RUN	go get github.com/tools/godep

# Install nodejs, npm, gulp, etc
RUN apt-get update -qq && apt-get install -y -q nodejs=0.10.25~dfsg2-2ubuntu1 npm=1.3.10~dfsg-1
# karma has dependencies that need to run native builds, so we need build-essential
RUN apt-get update -qq && apt-get install build-essential=11.6ubuntu6

# create link from nodejs to node
RUN ln /usr/bin/nodejs /usr/bin/node
RUN npm install -g gulp@3.8.10 jshint@2.5.11 6to5@2.13.3
# add a script that allows tasks to be
# performed as a specific user
ADD userdo.sh /root/userdo.sh

# build dependencies
RUN	apt-get update -qq && apt-get install -y -q make gcc libpam0g-dev ruby ruby-dev
RUN	gem install fpm
RUN	apt-get update -qq && apt-get install -y -q rpm
RUN	apt-get update -qq && apt-get install -y -q mercurial bzr
RUN apt-get update -qq && apt-get install -y -q libdevmapper-dev libsqlite3-dev

# Cache the NPM packages within the docker image
RUN mkdir -p /npm-cache/serviced/node_modules
ADD package.json /npm-cache/serviced/package.json
ADD npm-shrinkwrap.json /npm-cache/serviced/npm-shrinkwrap.json
RUN cd /npm-cache/serviced; npm install

WORKDIR	/go/src/github.com/control-center/serviced
