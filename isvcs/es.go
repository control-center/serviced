package isvcs

var ElasticSearchContainer ISvc

const es_dockerfile = `
FROM ubuntu
MAINTAINER Zenoss <dev@zenoss.com>

RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get update
RUN apt-get upgrade -y

RUN apt-get install -y -q openjdk-7-jre-headless wget
RUN wget -q -O elasticsearch-0.90.5.tar.gz https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-0.90.5.tar.gz 

RUN tar xvfz elasticsearch-0.90.5.tar.gz -C /opt

ENV JAVA_HOME /usr/lib/jvm/java-7-openjdk-amd64

EXPOSE 9200:9200

ENTRYPOINT ["/opt/elasticsearch-0.90.5/bin/elasticsearch"]
CMD ["-f"]
`

func init() {
	ElasticSearchContainer = ISvc{
		Name:       "elasticsearch",
		Dockerfile: es_dockerfile,
		Tag:        "zenoss/es",
	}
}

