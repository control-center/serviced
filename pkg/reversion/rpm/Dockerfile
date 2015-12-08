#
# This container is used to run the script which changes the version an RPM package for serviced
#
FROM centos:centos7
MAINTAINER Zenoss <dev@zenoss.com>

RUN yum clean all

# Add the docker repo
RUN echo -e "[dockerrepo]\nname=Docker Repository\nbaseurl=https://yum.dockerproject.org/repo/main/centos/7\nenabled=1\ngpgcheck=1\ngpgkey=https://yum.dockerproject.org/gpg" > /etc/yum.repos.d/docker.repo
RUN yum makecache fast

RUN yum install -y wget rpm-build

# Install the rpmrebuild tool
RUN wget "http://downloads.sourceforge.net/project/rpmrebuild/rpmrebuild/2.11/rpmrebuild-2.11-1.noarch.rpm?r=http%3A%2F%2Fsourceforge.net%2Fprojects%2Frpmrebuild%2Ffiles%2Frpmrebuild%2F2.11%2F&ts=1421178242&use_mirror=hivelocity" \
	-O rpmrebuild-2.11-1.noarch.rpm
RUN rpm -Uvh rpmrebuild-2.11-1.noarch.rpm

RUN rpm -ivh http://get.zenoss.io/yum/zenoss-repo-1-1.x86_64.rpm

ADD reversion.sh /usr/local/bin/
