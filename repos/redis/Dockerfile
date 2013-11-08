FROM mattdm/fedora:f19
RUN yum clean all
RUN yum --exclude=systemd\* update -y
RUN rpmdb --rebuilddb
RUN yum -y install redis

VOLUME /var/lib/redis
EXPOSE 6379

CMD ["/usr/sbin/redis-server", "--dir", "/var/lib/redis"]

