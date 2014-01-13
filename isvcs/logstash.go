package isvcs

import (
	"fmt"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"io/ioutil"
	"net/http"
	"strings"
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
			Tag:        "zenoss/logstash_master",
			Ports:      []int{5043, 9292},
		},
	}
}

// this method re
func getFilterDefinitions(services []dao.ServiceDefinition) map[string]string {
	filterDefs := make(map[string]string)
	for _, service := range services {
		for name, value := range service.LogFilters {
			filterDefs[name] = value
		}

		if len(service.Services) > 0 {
			subFilterDefs := getFilterDefinitions(service.Services)
			for name, value := range subFilterDefs {
				filterDefs[name] = value
			}
		}
	}
	return filterDefs
}

func getFilters(services []dao.ServiceDefinition, filterDefs map[string]string) string {
	filters := ""
	for _, service := range services {
		for _, config := range service.LogConfigs {
			for _, filtName := range config.Filters {
				filters += fmt.Sprintf("\nif [type] == \"%s\" \n {\n  %s \n}", config.Type, filterDefs[filtName])
			}
		}
		if len(service.Services) > 0 {
			subFilts := getFilters(service.Services, filterDefs)
			filters += subFilts
		}
	}
	return filters
}

// This method writes out the config file for logstash. It uses
// the logstash.conf.template and does a variable replacement.
func writeLogStashConfigFile(filters string) error {
	// read the log configuration template
	templatePath := resourcesDir() + "/logstash/logstash.conf.template"
	configPath := resourcesDir() + "/logstash/logstash.conf"

	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return err
	}
	newContents := strings.Replace(string(contents), "${FILTER_SECTION}", filters, 1)
	newBytes := []byte(newContents)
	// generate the filters section
	// write the log file
	err = ioutil.WriteFile(configPath, newBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (c *LogstashISvc) StartService(templates map[string]*dao.ServiceTemplate) error {

	// the definitions are a map of filter name to content
	// they are found by recursively going through all the service definitions
	filterDefs := make(map[string]string)
	for _, template := range templates {
		subFilterDefs := getFilterDefinitions(template.Services)
		for name, value := range subFilterDefs {
			filterDefs[name] = value
		}
	}

	// filters will be a syntactically correct logstash filters section
	filters := ""

	for _, template := range templates {
		filters += getFilters(template.Services, filterDefs)
	}

	glog.V(2).Infof("%s", filters)

	err := writeLogStashConfigFile(filters)
	if err != nil {
		return err
	}
	// make a map of the type => filters for all the types that have filters

	// start up the service

	err = c.ISvc.Run()
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
