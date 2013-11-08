package isvcs

import (
	"fmt"
	"net/http"
	"time"
)

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

RUN cd /opt/elasticsearch-0.90.5 && ./bin/plugin -install mobz/elasticsearch-head
ENTRYPOINT ["/opt/elasticsearch-0.90.5/bin/elasticsearch"]
CMD ["-f"]
`

type ElasticSearchISvc struct {
	ISvc
}

var ElasticSearchContainer ElasticSearchISvc

func init() {
	ElasticSearchContainer = ElasticSearchISvc{
		ISvc{
			Name:       "elasticsearch",
			Dockerfile: es_dockerfile,
			Tag:        "zenoss/es",
			Ports:      []int{9200},
		},
	}
}

func (c *ElasticSearchISvc) Run() error {
	err := c.ISvc.Run()
	if err != nil {
		return err
	}

	start := time.Now()
	timeout := time.Second * 30
	for {
		_, err = http.Get("http://localhost:9200/")
		if err == nil {
			break
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("Could not startup elastic search container.")
		}
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}
