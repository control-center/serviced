#!/bin/sh

mysql -u root -e "drop database if exists cp; create database cp"
mysql -u root cp -e "source ../database.sql"

HOST=$(./serviced add-host localhost:4979)
POOLID=$(./serviced add-pool default 0 0 0)

# add host to pool
./serviced add-host-to-pool $HOST $POOLID

COMMAND='/bin/sh -c "while true; do echo hello world; sleep 1; done"'

SERVICE=$(./serviced add-service helloWorld $POOLID base $COMMAND)

echo "HOST = $HOST"
echo "POOLID = $POOLID"
echo "SERVICE = $SERVICE"

./serviced start-service $SERVICE
