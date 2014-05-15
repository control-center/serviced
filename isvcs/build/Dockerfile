FROM ubuntu:trusty
MAINTAINER Zenoss <dev@zenoss.com>

RUN echo "deb http://archive.ubuntu.com/ubuntu trusty main universe" > /etc/apt/sources.list
RUN apt-get update; \
    apt-get upgrade -y; \
    apt-get install -y -q openjdk-7-jre-headless gnuplot-nox supervisor make \
                          redis-server git-core build-essential python-dev \
                          libevent1-dev python-openssl liblzma-dev wget \
                          sqlite; \
    apt-get clean; \
    cd /tmp; \
    wget https://bitbucket.org/pypa/setuptools/raw/bootstrap/ez_setup.py; \
    python ez_setup.py; easy_install pip; rm ez_setup.py; \
    rm -rf /var/lib/apt/lists/*_* /tmp/* /var/tmp/*
RUN mkdir /usr/local/serviced/resources -p # this is mounted in to every isvcs container
RUN git clone --branch 0.6.9 --depth 1 https://github.com/dotcloud/docker-registry.git && \
    cd docker-registry && \
    pip install /docker-registry/
ADD supervisor.conf /opt/zenoss/etc/supervisor.conf
ADD consumer.tar.gz /
ADD opentsdb.tar.gz /
ADD query.tar.gz /
ADD es.tar.gz /
ADD zk.tar.gz /
ADD logstash.tar.gz /
ADD celery.tar.gz /
