FROM mattdm/fedora
RUN yum clean all
RUN yum --exclude=systemd\* update -y
RUN rpmdb --rebuilddb
RUN yum -y install mariadb-server mariadb

ADD ./mysql-setup.sh /tmp/mysql-setup.sh
ADD ./mysql-run.sh /mysql-run.sh
RUN /bin/sh /tmp/mysql-setup.sh

VOLUME /var/lib/mysql
EXPOSE 3306
ENTRYPOINT ["/mysql-run.sh"]

