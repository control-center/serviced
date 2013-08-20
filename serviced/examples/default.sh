#!/bin/sh


cd $GOPATH/src/github.com/zenoss/serviced/
mysql -u root -e "drop database if exists cp; create database cp"
mysql -u root cp -e "source svc/database.sql"

POOLID=$(./serviced/serviced add-pool default 0 0 0)
HOST=$(./serviced/serviced add-host localhost:4979 $POOLID)

COMMAND='/bin/sh -c "while true; do echo hello world; sleep 1; done"'

SERVICE=$(./serviced/serviced add-service helloWorld $POOLID base $COMMAND)

echo "HOST = $HOST"
echo "POOLID = $POOLID"
echo "Hello, world service: $SERVICE"

./serviced/serviced start-service $SERVICE


# create a hellohost service
COMMAND='/helloHost'
SERVICE=$(./serviced/serviced add-service hellHost $POOLID dgarcia/helloHost /helloHost)

./serviced/serviced start-service $SERVICE

echo "hello host service: $SERVICE "

