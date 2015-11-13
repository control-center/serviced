#!/bin/bash

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

ES_VER=0.90.13
ES_TMP=/tmp/serviced_elastic
ES_DIR=$ES_TMP/elasticsearch-$ES_VER

if [[ -z $GORACE ]]; then
	export GORACE="history_size=7 halt_on_error=1";
fi;
if [[ -z $GOTEST ]]; then
	export GOTEST="-cover -race -p 1";
fi;

function stop_elastic {
	if [ -e $ES_TMP/pid ]; then
		kill `cat $ES_TMP/pid`;
	fi
	rm -rf $ES_TMP
}

stop_elastic

BUILD_TAGS="$(bash ${DIR}/build-tags.sh)"

echo "Running unit tests ..."
START_TIME=`date --utc +%s`
godep go test -tags="${BUILD_TAGS} unit" $GOTEST ./...
UNIT_TEST_RESULT=$?
END_TIME=`date --utc +%s`
echo "Unit tests finished in $(($END_TIME - $START_TIME)) seconds"

echo "Running integration tests ..."
START_TIME=`date --utc +%s`
mkdir $ES_TMP
if [ ! -e /tmp/elasticsearch-$ES_VER.tar.gz ]; then
	curl https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-$ES_VER.tar.gz > /tmp/elasticsearch-$ES_VER.tar.gz;
fi

tar -xf /tmp/elasticsearch-$ES_VER.tar.gz -C $ES_TMP

cat <<EOF > $ES_DIR/config/elasticsearch.yml
cluster.name: $(head -c32 /dev/urandom | cksum | awk {'print $1'})
multicast.enabled: false
discovery.zen.ping.multicast.ping.enabled: false
EOF

$ES_DIR/bin/elasticsearch -f -Des.http.port=9202 > $ES_TMP/elastic.log & echo $!>$ES_TMP/pid

godep go test -tags="${BUILD_TAGS} integration" $GOTEST ./...
INTEGRATION_TEST_RESULT=$?
stop_elastic
END_TIME=`date --utc +%s`
echo "Integration tests finished in $(($END_TIME - $START_TIME)) seconds"

exit $(($UNIT_TEST_RESULT + $INTEGRATION_TEST_RESULT))
