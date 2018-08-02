#
# This container is used to run the script which changes the version number of a debian package for serviced
#
FROM ubuntu:16.04
MAINTAINER Zenoss <dev@zenoss.com>

USER root

# devscripts contains the deb-reversion script
RUN apt-get update -y
RUN apt-get install -y wget devscripts sudo

