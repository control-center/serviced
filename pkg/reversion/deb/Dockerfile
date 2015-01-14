#
# This container is used to run the script which changes the version number of a debian package for serviced
#
FROM ubuntu:14.04
MAINTAINER Zenoss <dev@zenoss.com>

# devscripts contains the deb-reversion script
RUN apt-get install -y wget devscripts

RUN apt-key adv --keyserver keys.gnupg.net --recv-keys AA5A1AD7

ADD reversion.sh /usr/local/bin/
