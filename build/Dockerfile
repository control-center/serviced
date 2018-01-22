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
FROM ubuntu:xenial

MAINTAINER Zenoss, Inc <dev@zenoss.com>

RUN	apt-get update -qq && apt-get install -qqy iptables ca-certificates aufs-tools

# Build dependencies
#RUN	dpkg-divert --local --rename --add /sbin/initctl
#RUN	ln -s /bin/true /sbin/initctl
RUN	apt-get update -qq && apt-get install -y -q wget curl git unzip

# Install Go
RUN	curl -s https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV	GOPATH  /go
ENV	PATH $PATH:/go/bin:/usr/local/go/bin
RUN	go get github.com/tools/godep

# build dependencies
RUN	apt-get update -qq && apt-get install -y -q make gcc libpam0g-dev ruby ruby-dev
RUN	gem install fpm
RUN	apt-get update -qq && apt-get install -y -q rpm
RUN	apt-get update -qq && apt-get install -y -q mercurial bzr
RUN apt-get update -qq && apt-get install -y -q libdevmapper-dev libsqlite3-dev

# Install the xvfb for firefox and chrome so they can run on a headless system
RUN apt-get update -qq && apt-get install -y -q xvfb

# Install chromedriver that selenium needs to work with chrome
# (from https://devblog.supportbee.com/2014/10/27/setting-up-cucumber-to-run-with-Chrome-on-Linux/)
RUN wget -N http://chromedriver.storage.googleapis.com/2.24/chromedriver_linux64.zip -P /tmp
RUN unzip /tmp/chromedriver_linux64.zip -d /tmp
RUN mv /tmp/chromedriver /usr/bin
RUN chmod +x /usr/bin/chromedriver
RUN rm /tmp/chromedriver_linux64.zip

# Install chrome - blend of info from several sources
# General process info: http://askubuntu.com/questions/79280/how-to-install-chrome-browser-properly-via-command-line
# Public Key info for safe download: http://www.google.com/linuxrepositories/
# Info about differnet Chrome versions: http://www.ubuntuupdates.org/package/google_chrome/stable/main/base/google-chrome-stable
RUN apt-get update -qq && apt-get install -y -q libxss1 libappindicator1 libindicator7
RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add -
RUN echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list

# Tried a specific version like 41.0.2272.76-1, but specifying on the command line doesn't always work :-(
RUN apt-get update -qq && apt-get install -y -q --force-yes google-chrome-stable


# Install nodejs, npm, gulp, etc
RUN curl -sL https://deb.nodesource.com/setup_6.x | bash -
RUN apt update -qq && apt install -y nodejs=6.12.3-1nodesource1
#RUN apt-get update -qq && apt-get install -y -q nodejs=0.10.25~dfsg2-2ubuntu1 npm=1.3.10~dfsg-1
# karma has dependencies that need to run native builds, so we need build-essential
RUN apt-get update -qq && apt-get install -yq build-essential=12.1ubuntu2

# Setup Xvfb - FF and chrome will connect to this DISPLAY
# (https://github.com/keyvanfatehi/docker-chrome-xvfb)
ENV DISPLAY :99
ADD xvfb_init /etc/init.d/xvfb
RUN chmod a+x /etc/init.d/xvfb

# install and use yarn instead of npm
RUN npm install -g yarn@0.16.1

# make a place for yarn globals to live
RUN mkdir /yarn-global
RUN chmod 755 /yarn-global
RUN yarn global add gulp@3.8.8 jshint@2.9.2 babel@5.8.23 --global-folder /yarn-global

# Cache the NPM packages within the docker image
RUN mkdir -p /npm-cache/serviced/node_modules
ADD package.json /npm-cache/serviced/package.json
ADD yarn.lock /npm-cache/serviced/yarn.lock
RUN cd /npm-cache/serviced; yarn install --pure-lockfile

# HACK - yarn doesnt try to normalize permissions after
# unpacking tar files so do it by hand. see
# https://github.com/yarnpkg/yarn/pull/1490
RUN chmod -R +r /npm-cache/serviced/node_modules

# add a script that allows tasks to be
# performed as a specific user
ADD userdo.sh /root/userdo.sh

WORKDIR	/go/src/github.com/control-center/serviced
