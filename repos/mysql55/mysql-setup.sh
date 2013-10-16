#!/bin/sh

/usr/sbin/mysqld &
MYSQL=$!
sleep 5
echo "GRANT ALL ON *.* TO root@'%' IDENTIFIED BY '' WITH GRANT OPTION; FLUSH PRIVILEGES" | mysql
kill $MYSQL
wait $MYSQL
tar cvfz /var/lib/mysql.tgz /var/lib/mysql

