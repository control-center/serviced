#!/bin/bash

mkdir -p /tmp/tsdb

/opt/hbase-0.94.14/bin/start-hbase.sh

while [[ `/opt/opentsdb/src/create_table.sh` != *"ERROR: Table already exists: tsdb"* ]]; do
    echo `date` ": Waiting for HBase to be ready..."
    sleep 2
done

/opt/opentsdb/build/tsdb tsd --port=4242 --staticroot=/opt/opentsdb/build/staticroot --cachedir=/tmp/tsd --auto-metric &

cd /opt/zenoss

exec bin/metric-consumer-app.sh
