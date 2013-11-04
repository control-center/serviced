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

docker-version 0.6.1
from	ubuntu:12.04
maintainer	Zenoss, Inc <dev@zenoss.com>	

# Build dependencies
run	apt-get update
run	dpkg-divert --local --rename --add /sbin/initctl
run	ln -s /bin/true /sbin/initctl
run	apt-get install -y -q wget curl git
run	echo 'deb http://archive.ubuntu.com/ubuntu precise main universe' > /etc/apt/sources.list
run	wget -qO- https://get.docker.io/gpg | apt-key add -
run	echo 'deb http://get.docker.io/ubuntu docker main' > /etc/apt/sources.list.d/docker.list
run	apt-get update
run	apt-get install -y -q lxc-docker

# Install Go
run	curl -s http://go.googlecode.com/files/go1.1.2.linux-amd64.tar.gz | tar -v -C / -xz && mv /go /goroot
env GOROOT	/goroot
env	PATH	$PATH:/goroot/bin
env	GOPATH	/go

# Runtime dependencies
run	apt-get install -y -q iptables lxc aufs-tools

# build dependencies
run	apt-get install -y -q make gcc libpam0g-dev ruby ruby-dev
run	apt-get	install -y -q rubygems
run	gem install fpm
run	apt-get install -y -q rpm

volume	/var/lib/serviced
workdir	/go/src/github.com/zenoss/serviced

