FROM mattdm/fedora
RUN yum clean all
RUN yum --exclude=systemd\* update -y
RUN yum -y install redis

VOLUME /var/lib/redis
EXPOSE 6379

ENTRYPOINT ["/usr/sbin/redis-server"]
CMD ["--dir", "/var/lib/redis"]

