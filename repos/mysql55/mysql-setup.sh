#!/bin/sh
/usr/bin/mysql_install_db --user=mysql
/usr/bin/mysqld_safe &
sleep 5
echo "GRANT ALL ON *.* TO root@'%' IDENTIFIED BY '' WITH GRANT OPTION; FLUSH PRIVILEGES" | mysql
MYSQL=$(cat /var/run/mysqld/mysqld.pid)
echo "Killing PID $MYSQL"
kill $MYSQL
while [ -f /var/run/mysqld/mysqld.pid ]; do
    echo "Waiting my mysql to die..."
    sleep 1
done
tar cvfz /var/lib/mysql.tgz /var/lib/mysql

