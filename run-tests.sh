#!/bin/bash

ES_VER=0.90.13
ES_TMP=/tmp/serviced_elastic
ES_DIR=$ES_TMP/elasticsearch-$ES_VER

if [[ -z $GORACE ]]; then
	export GORACE="history_size=7 halt_on_error=1";
fi;
if [[ -z $GOTEST ]]; then
	export GOTEST="-v -race -p 1";
fi;

function stop_elastic {
	if [ -e $ES_TMP/pid ]; then
		kill `cat $ES_TMP/pid`;
	fi
	rm -rf $ES_TMP
}

stop_elastic

mkdir $ES_TMP
if [ ! -e /tmp/elasticsearch-$ES_VER.tar.gz ]; then
	curl https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-$ES_VER.tar.gz > /tmp/elasticsearch-$ES_VER.tar.gz;
fi

tar -xf /tmp/elasticsearch-$ES_VER.tar.gz -C $ES_TMP
echo "cluster.name: zero" > $ES_DIR/config/elasticsearch.yml
$ES_DIR/bin/elasticsearch -f -Des.http.port=9202 > $ES_TMP/elastic.log & echo $!>$ES_TMP/pid

godep go test $GOTEST ./...
RESULT=$?

stop_elastic

exit $RESULT
