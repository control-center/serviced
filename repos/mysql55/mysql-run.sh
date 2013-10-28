#!/bin/sh

tar xvfz /var/lib/mysql.tgz
chown mysql:mysql /var/lib/mysql
/usr/bin/mysqld_safe

