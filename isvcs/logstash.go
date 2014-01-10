package isvcs

import (
	"github.com/zenoss/glog"

	"fmt"
	"net/http"
	"time"
)

const logstash_dockerfile = `
FROM ubuntu:latest
MAINTAINER jhanson@zenoss.com

RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get -y update
RUN apt-get -y upgrade

ENV DEBIAN_FRONTEND noninteractive

# Fake a fuse install
RUN apt-get install libfuse2
RUN cd /tmp ; apt-get download fuse && dpkg-deb -x fuse_* . && dpkg-deb -e fuse_* &&  rm fuse_*.deb
RUN cd /tmp ; echo -en '#!/bin/bash\nexit 0\n' > DEBIAN/postinst
RUN cd /tmp ; dpkg-deb -b . /fuse.deb && dpkg -i /fuse.deb

RUN apt-get -y install openjdk-7-jdk git build-essential curl autoconf libtool gnuplot wget maven libxml-xpath-perl bc
RUN update-alternatives --install "/usr/bin/java" "java" "/usr/lib/jvm/java-7-openjdk-amd64/bin/java" 1
RUN update-alternatives --set "java" "/usr/lib/jvm/java-7-openjdk-amd64/bin/java"

RUN mkdir /root/logstash
RUN cd /root/logstash && wget https://download.elasticsearch.org/logstash/logstash/logstash-1.3.2-flatjar.jar

# listen for logstash-forwarder requests
EXPOSE 5043

# web requests
EXPOSE 9292

# /usr/local/serviced is hardcoded in isvcs.go
ENTRYPOINT java -jar /root/logstash/logstash-1.3.2-flatjar.jar agent -f /usr/local/serviced/resources/logstash/logstash.conf -- web
`

type LogstashISvc struct {
	ISvc
}

var LogstashContainer LogstashISvc

func init() {
	LogstashContainer = LogstashISvc{
		ISvc{
			Name:       "logstash_master",
			Dockerfile: logstash_dockerfile,
			Repository: "zctrl/logstash_master",
			Tag:        "v1",
			Ports:      []int{5043, 9292},
		},
	}
}

func (c *LogstashISvc) Run() error {
	err := c.ISvc.Run()
	if err != nil {
		return err
	}

	start := time.Now()
	timeout := time.Second * 30
	for {
		_, err = http.Get("http://localhost:9292/")
		if err == nil {
			break
		}
		running, err := c.Running()
		if !running {
			glog.Errorf("Logstash container stopped: %s", err)
			return err
		}
		if time.Since(start) > timeout {
			glog.Errorf("Timeout starting up logstash container")
			return fmt.Errorf("Could not startup logstash container.")
		}
		glog.V(2).Infof("Still trying to connect to logstash: %v", err)
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}
