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
    echo "Waiting 120 seconds for elasticsearch-logstash to start ..."
    retry 120 curl http://localhost:9100/_cluster/health &>/dev/null
    err=$?
    return $err
}

migrate_elasticsearch_ls() {
    local err
    echo "Starting export for old elasticsearch-logstash storage"
    retry 120 $HOME_SERVICED/bin/elastic -c $HOME_SERVICED/etc/es_cluster.ini \
        -f $HOST_ISVCS_DIR/elasticsearch_logstash_data.json -E &>/dev/null
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

echo "Starting docker container elasticsearch-logstash ..."
docker run --rm -d --name $SVC_NAME_LS -p 9100:9100 \
  -v $HOST_ES_LS_DATA_DIR:$CONTAINER_ES_LS_DATA_DIR zenoss/serviced-isvcs:v68 \
  sh -c "$ELASTIC_LS_BIN -Des.insecure.allow.root=true -Des.node.name=$NODE_NAME_LS -Des.cluster.name=$CLUSTER_NAME_LS"

if check_elasticsearch_ls; then
  report "SUCCESS" "Container started within timeout"
else
  report "FAILURE" "Container failed to start within 120 seconds"
fi

if migrate_elasticsearch_ls; then
  report "SUCCESS" "Export completed"
else
  report "FAILURE" "Export failed try make the export manual"
fi

echo "Stopping the container with old elasticsearch"
docker stop $SVC_NAME_LS

echo "Preparing the host for data import to new elasticsearch storage"
rm -rf ${HOST_ES_LS_DATA_DIR:?}/*
groupadd -f elastic -g 1001
id -u 1001 &>/dev/null || useradd elastic -u 1001 -g 1001
chown 1001:1001 -R $HOST_ES_LS_DATA_DIR

echo "Starting container with new elasticsearch logstash"
docker run --rm -d --name $SVC_NAME_LS -p 9100:9100 \
  -v $HOST_ES_LS_DATA_DIR:$CONTAINER_ES_LS_DATA_DIR zenoss/serviced-isvcs:v69 \
  su elastic -c "$ELASTIC_LS_BIN -Enode.name=$NODE_NAME_LS -Ecluster.name=$CLUSTER_NAME_LS"

if check_elasticsearch_ls; then
    report "SUCCESS" "Container started within timeout"
else
    report "FAILURE" "Container failed to start within 120 seconds"
fi

echo "Importing data to new elasticsearch logstash"
$HOME_SERVICED/bin/elastic -c $HOME_SERVICED/etc/es_cluster.ini \
  -f $HOST_ISVCS_DIR/elasticsearch_logstash_data.json -I

echo "Stopping the container with new elasticsearch"
docker stop $SVC_NAME_LS
