#!/bin/bash

mkdir -p /tmp/tsd

echo "Starting HBase..."
/opt/hbase-0.94.14/bin/start-hbase.sh

while [[ `/opt/opentsdb/src/create_table.sh` != *"ERROR: Table already exists: tsdb"* ]]; do
    echo `date` ": Waiting for HBase to be ready..."
    sleep 2
done

echo "Starting opentsdb..."
exec /opt/opentsdb/build/tsdb tsd --port=4242 --staticroot=/opt/opentsdb/build/staticroot --cachedir=/tmp/tsd --auto-metric 2>&1 > /var/log/supervisor/opentsdb.out.log

