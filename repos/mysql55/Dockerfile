FROM ubuntu

RUN dpkg-divert --local --rename --add /sbin/initctl
RUN ln -s /bin/true /sbin/initctl
RUN apt-get update && apt-get install -y mysql-server && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN sed -i -e"s/^bind-address\s*=\s*127.0.0.1/bind-address = 0.0.0.0/" /etc/mysql/my.cnf

ADD ./mysql-setup.sh /tmp/mysql-setup.sh
ADD ./mysql-run.sh /mysql-run.sh
RUN /bin/sh /tmp/mysql-setup.sh

VOLUME /var/lib/mysql
EXPOSE 3306
ENTRYPOINT ["/mysql-run.sh"]

