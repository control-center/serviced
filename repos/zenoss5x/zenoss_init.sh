#!/bin/sh
/usr/bin/mysql_install_db --user=mysql
/usr/bin/mysqld_safe &

/usr/sbin/redis-server /etc/redis.conf &
/usr/sbin/rabbitmq-server &
sleep 5
/usr/lib/rabbitmq/bin/rabbitmq-plugins enable rabbitmq_management
/sbin/rabbitmqctl stop
sleep 5
/usr/sbin/rabbitmq-server &
sleep 5

/etc/init.d/zenoss start
/etc/init.d/zenoss stop
mysqldump -u root zodb | gzip > /opt/zenoss/Products/ZenModel/data/zodb.sql.gz

