# es-serviced-migration
The python project for migrating elastic from 0.9 to 7.7.4 version.
For building the binary just run `make` inside the script folder. 
Binary should be distributed for migration system with config `.ini` file, `es_cluster.ini` by default

#### Manual steps for upgrading serviced from 1.7.0 to 1.8.0

*1.* Copy migration script and config to host for the upgrade:
```shell script
UPGRADED_HOST=10.87.209.233
scp elastic root@$UPGRADED_HOST:/tmp
scp es_cluster.ini root@UPGRADED_HOST:/tmp
```

*2.* Stop serviced and set env variables: 
```shell script
systemctl stop serviced
HOST_ES_DIR=/opt/serviced/var/isvcs
CONTAINER_ES_DIR=/opt/elasticsearch-serviced
SVC_NAME=/serviced-isvcs_elasticsearch-serviced
NODE_NAME=elasticsearch-serviced
CLUSTER_NAME=$(cat $HOST_ES_DIR/elasticsearch-serviced.clustername)
HOST_ES_DATA_DIR=$HOST_ES_DIR/elasticsearch-serviced/data
CONTAINER_ES_DATA_DIR=$CONTAINER_ES_DIR/data
ELASTIC_BIN=$CONTAINER_ES_DIR/bin/elasticsearch
```

*3.* Run docker container with elastic
```shell script
docker run --rm -d --name $SVC_NAME -p 9200:9200 -v $HOST_ES_DATA_DIR:$CONTAINER_ES_DATA_DIR zenoss/serviced-isvcs:v63 sh -c "$ELASTIC_BIN -f -Des.node.name=$NODE_NAME  -Des.cluster.name=$CLUSTER_NAME"
```

*4.* Run export script from old elastic storage
```shell script
./elastic
```

*5.* Stop the container with old elastic 
```shell script
docker stop $SVC_NAME
```

*6.* Download new serviced RPM and new docker RPMs as the required dependencies 
```shell script
cd /opt/zenoss-repo-mirror/
wget http://platform-jenkins.zenoss.eng/job/ControlCenter/job/develop/job/merge-rpm-build/254/artifact/output/serviced-1.8.0-0.0.406.unstable.x86_64.rpm
wget https://download.docker.com/linux/centos/7/x86_64/stable/Packages/docker-ce-19.03.11-3.el7.x86_64.rpm
wget https://download.docker.com/linux/centos/7/x86_64/stable/Packages/docker-ce-cli-19.03.11-3.el7.x86_64.rpm
yum update --enablerepo=zenoss-mirror /opt/zenoss-repo-mirror/serviced-1.8.0-0.0.406.unstable.x86_64.rpm /opt/zenoss-repo-mirror/docker-ce-19.03.11-3.el7.x86_64.rpm  /opt/zenoss-repo-mirror/docker-ce-cli-19.03.11-3.el7.x86_64.rpm

```

*7.* Set `vm` config according to SRE-1010 issue.
```shell script
sysctl -w vm.max_map_count=262144
```

*8.* Copy stub `snapshot.0` file to zookeeper storage dir for migration from 3.4.x to 3.5.5 version 
according to the [issue ZOOKEEPER-3056](https://issues.apache.org/jira/browse/ZOOKEEPER-3056) 
```shell script
cd /tmp
wget https://issues.apache.org/jira/secure/attachment/12928686/snapshot.0
mv snapshot.0 $HOST_ES_DIR/zookeeper/data/version-2/
```

*9.* Prepare the host for data import to new elastic 
```shell script
rm -rf $HOST_ES_DATA_DIR/*
groupadd -f elastic -g 1001
id -u 1001 &>/dev/null || useradd elastic -u 1001 -g 1001
chown 1001:1001 -R $HOST_ES_DATA_DIR
```

*10.* Start container with new elastic 
```shell script
docker run --rm -d --name $SVC_NAME -p 9200:9200 -v $HOST_ES_DATA_DIR:$CONTAINER_ES_DATA_DIR zenoss/serviced-isvcs:v66 su elastic -c "$ELASTIC_BIN -Ecluster.initial_master_nodes=$NODE_NAME -Enode.name=$NODE_NAME -Ecluster.name=$CLUSTER_NAME"
#wait for start check by 
curl http://localhost:9200/_cluster/health
```

*11.* Import data to new elastic 
```shell script
./elastic -i
docker stop $SVC_NAME
```

*12.* Start serviced
```shell script
systemctl daemon-reload
systemctl start serviced
```




