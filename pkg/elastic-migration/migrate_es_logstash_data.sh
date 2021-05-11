#!/bin/sh

report() {
    local msg
    msg=$1
    shift
    echo ===== "$msg" =====
    echo "$*"
    echo ===================
}

retry() {
    local command
    local duration
    local result
    local timeout
    timeout=$1
    shift
    command="$*"
    duration=0
    until [[ ${duration} -ge ${timeout} ]]; do
        ${command}; result=$?; [ ${result} = 0 ] && break
        duration=$((duration + 1))
        sleep 1
    done
    return ${result}
}

check_elasticsearch_ls() {
    local err
    local port
    port=$1
    echo "Waiting 120 seconds for elasticsearch-logstash on localhost:${port} to start ..."
    retry 120 curl http://localhost:${port}/_cluster/health &>/dev/null
    err=$?
    return $err
}

migrate_elasticsearch_ls() {
    local err
    echo "Starting migration from old elasticsearch-logstash storage to new"
    $HOME_SERVICED/bin/elastic -c $HOME_SERVICED/etc/es_cluster.ini -M
    err=$?
    return $err
}

HOME_SERVICED=/opt/serviced
HOST_ISVCS_DIR=$HOME_SERVICED/var/isvcs
CONTAINER_ES_LS_DIR=/opt/elasticsearch-logstash
SVC_NAME_LS=/serviced-isvcs_elasticsearch-logstash
NODE_NAME_LS=elasticsearch-logstash
CLUSTER_NAME_LS=$(cat $HOST_ISVCS_DIR/elasticsearch-logstash.clustername)
HOST_ES_LS_DATA_DIR=$HOST_ISVCS_DIR/elasticsearch-logstash/data
CONTAINER_ES_LS_DATA_DIR=$CONTAINER_ES_LS_DIR/data
ELASTIC_LS_BIN=$CONTAINER_ES_LS_DIR/bin/elasticsearch
SVC_NAME_LS_NEW=/serviced-isvcs_elasticsearch-logstash-new

mkdir -p $HOST_ISVCS_DIR/elasticsearch-logstash-new/data
HOST_ES_LS_DATA_DIR_NEW=$HOST_ISVCS_DIR/elasticsearch-logstash-new/data

echo "Starting docker container elasticsearch-logstash ..."
docker run --rm --ulimit memlock=-1:-1 -d --name $SVC_NAME_LS -p 9100:9100 \
  -v $HOST_ES_LS_DATA_DIR:$CONTAINER_ES_LS_DATA_DIR zenoss/serviced-isvcs:v68 \
  sh -c "ES_HEAP_SIZE='8g' $ELASTIC_LS_BIN -Des.insecure.allow.root=true -Des.node.name=$NODE_NAME_LS -Des.cluster.name=$CLUSTER_NAME_LS"

if check_elasticsearch_ls 9100; then
  report "SUCCESS" "Container started within timeout"
else
  report "FAILURE" "Container failed to start within 120 seconds"
  exit 1
fi

# Removing the old version of kibana settings if exists
curl -XDELETE http://localhost:9100/kibana-int?ignore_unavailable=true

groupadd -f elastic -g 1001
id -u 1001 &>/dev/null || useradd elastic -u 1001 -g 1001
chown 1001:1001 -R $HOST_ES_LS_DATA_DIR_NEW

echo "Starting container with new elasticsearch logstash"
docker run --rm -d --name $SVC_NAME_LS_NEW -p 9101:9100 \
  -v $HOST_ES_LS_DATA_DIR_NEW:$CONTAINER_ES_LS_DATA_DIR zenoss/serviced-isvcs:v69 \
  su elastic -c "JAVA_HOME=/usr/lib/jvm/jre-11; ES_JAVA_OPTS='-Xms8g -Xmx8g' $ELASTIC_LS_BIN -Enode.name=$NODE_NAME_LS -Ecluster.name=$CLUSTER_NAME_LS"

if check_elasticsearch_ls 9101; then
    report "SUCCESS" "Container started within timeout"
else
    report "FAILURE" "Container failed to start within 120 seconds"
    exit 1
    docker stop $SVC_NAME_LS
fi

if migrate_elasticsearch_ls; then
  report "SUCCESS" "Migration completed"
else
  report "FAILURE" "Migration failed try make the export manual"
  docker stop $SVC_NAME_LS
  docker stop $SVC_NAME_LS_NEW
  rm -rf $HOST_ISVCS_DIR/elasticsearch-logstash-new
  exit 1
fi

echo "Force merge to restore segments"
curl -XPOST 'localhost:9101/_forcemerge?max_num_segments=5'

echo "Refresh all indecies"
curl -XPOST 'localhost:9101/_refresh'

echo "Stopping the container with old elasticsearch"
docker stop $SVC_NAME_LS

echo "Stopping the container with new elasticsearch"
docker stop $SVC_NAME_LS_NEW

echo "Replacing old data folder to new "
rm -rf ${HOST_ES_LS_DATA_DIR:?}
mv $HOST_ISVCS_DIR/elasticsearch-logstash-new/* $HOST_ISVCS_DIR/elasticsearch-logstash
