FROM ubuntu:trusty
MAINTAINER Zenoss <dev@zenoss.com>

RUN mkdir /usr/local/serviced/resources -p # this is mounted in to every isvcs container
RUN echo "deb http://archive.ubuntu.com/ubuntu trusty main universe" > /etc/apt/sources.list; \
    apt-get update; \
    apt-get upgrade -y; \
    apt-get install -y -q openjdk-7-jre-headless gnuplot-nox supervisor make \
    redis-server git-core build-essential python-dev \
    libevent1-dev python-openssl liblzma-dev wget; \
    apt-get clean; \
    cd /tmp; \
    wget https://bitbucket.org/pypa/setuptools/raw/bootstrap/ez_setup.py; \
    python ez_setup.py; \
    easy_install pip; \
    rm -rf /var/lib/apt/lists/*_* /tmp/*
RUN git clone --branch 0.6.9 --depth 1 https://github.com/dotcloud/docker-registry.git && \
    cd docker-registry && \
    echo "common:"                                                                 > config/config_serviced.yml  && \
    echo "  standalone: true"                                                      >> config/config_serviced.yml && \
    echo "  disable_token_auth: true"                                              >> config/config_serviced.yml && \
    echo "  secret_key: _env:SECRET_KEY"                                           >> config/config_serviced.yml && \
    echo "  search_backend: sqlalchemy"                                            >> config/config_serviced.yml && \
    echo "  sqlalchemy_index_database: sqlite:////tmp/registry/docker-registry.db" >> config/config_serviced.yml && \
    echo "  storage: local"                                                        >> config/config_serviced.yml && \
    echo "  storage_path: /tmp/registry"                                           >> config/config_serviced.yml && \
    echo "  loglevel: debug"                                                       >> config/config_serviced.yml && \
    pip install /docker-registry/
ADD supervisor.conf /opt/zenoss/etc/supervisor.conf
ADD consumer.tar.gz /
ADD opentsdb.tar.gz /
ADD query.tar.gz /
ADD es.tar.gz /
ADD zk.tar.gz /
ADD logstash.tar.gz /
ADD celery.tar.gz /
