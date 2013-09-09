#!/usr/bin/env bash


./serviced/serviced add-host 10.87.110.30:4979 default
#./serviced/serviced add-service -p tcp:8000:www www default ubuntu_p python -m SimpleHTTPServer
#./serviced/serviced add-service -q tcp:8000:www www default ubuntu_p '/bin/sh -c "while true; do wget http://127.0.0.1:8000/ ; sleep 10; done"'
#mysql -u root cp -e "update service set instances = 1"

./serviced/serviced add-service -p tcp:3306:mysql mysql default dgarcia/zenoss424 'su - mysql -c "/usr/sbin/mysqld"'
./serviced/serviced add-service -p tcp:5672:amqp amqp default dgarcia/zenoss424 /usr/sbin/rabbitmq-server
./serviced/serviced add-service -q tcp:3306:mysql -q tcp:5672:amqp -p tcp:8080:zope zope default dgarcia/zenoss424 'su - zenoss -c "/opt/zenoss/bin/zopectl fg"'
#./serviced/serviced add-service -q tcp:3306:mysql mysql_test default ubuntu_p /test_mysql.sh
#mysql -u root cp -e "update service set instances = 1"

