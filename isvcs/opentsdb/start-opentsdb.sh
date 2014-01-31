#!/bin/bash

mkdir -p /tmp/tsd

export COMPRESSION=NONE
export HBASE_HOME=/opt/hbase-0.94.14
export JAVA_HOME=/usr/lib/jvm/java-7-openjdk-amd64

echo "Starting HBase..."
/opt/hbase-0.94.14/bin/start-hbase.sh 

while [[ `/opt/opentsdb/src/create_table.sh` != *"ERROR: Table already exists: tsdb"* ]]; do
    echo `date` ": Waiting for HBase to be ready..."
    sleep 2
done

echo "Starting opentsdb..."
exec /opt/opentsdb/build/tsdb tsd --port=4242 --staticroot=/opt/opentsdb/build/staticroot --cachedir=/tmp/tsd --auto-metric


