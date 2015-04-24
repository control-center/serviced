// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package isvcs

import (
	"github.com/zenoss/elastigo/cluster"
	"github.com/zenoss/glog"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var elasticsearch_logstash *IService
var elasticsearch_serviced *IService

func init() {
	var serviceName string
	var err error

	serviceName = "elasticsearch-serviced"

	elasticsearch_servicedPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_ELASTICSEARCH_SERVICED_PORT_9200_HOSTIP",
		HostPort:       9200,
	}

	defaultHealthCheck := healthCheckDefinition{
		healthCheck: elasticsearchHealthCheck(9200),
		Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
		Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
	}
	healthChecks := map[string]healthCheckDefinition{
		DEFAULT_HEALTHCHECK_NAME: defaultHealthCheck,
	}

	elasticsearch_serviced, err = NewIService(
		IServiceDefinition{
			Name:          serviceName,
			Repo:          IMAGE_REPO,
			Tag:           IMAGE_TAG,
			Command:       func() string { return "" },
			PortBindings:  []portBinding{elasticsearch_servicedPortBinding},
			Volumes:       map[string]string{"data": "/opt/elasticsearch-0.90.9/data"},
			Configuration: make(map[string]interface{}),
			HealthChecks:  healthChecks,
			HostNetwork:   true,
		},
	)
	if err != nil {
		glog.Fatal("Error initializing elasticsearch container: %s", err)
	}
	elasticsearch_serviced.Command = func() string {
		clusterArg := ""
		if clusterName, ok := elasticsearch_serviced.Configuration["cluster"]; ok {
			clusterArg = fmt.Sprintf(" -Des.cluster.name=%s ", clusterName)
		}
		return fmt.Sprintf(`/opt/elasticsearch-0.90.9/bin/elasticsearch -f -Des.node.name=%s %s`, elasticsearch_serviced.Name, clusterArg)
	}

	serviceName = "elasticsearch-logstash"
	elasticsearch_logstashPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_ELASTICSEARCH_LOGSTASH_PORT_9100_HOSTIP",
		HostPort:       9100,
	}

	logStashHealthCheck := defaultHealthCheck
	logStashHealthCheck.healthCheck = elasticsearchHealthCheck(9100)
	healthChecks = map[string]healthCheckDefinition{
		DEFAULT_HEALTHCHECK_NAME: logStashHealthCheck,
	}

	elasticsearch_logstash, err = NewIService(
		IServiceDefinition{
			Name:          serviceName,
			Repo:          IMAGE_REPO,
			Tag:           IMAGE_TAG,
			Command:       func() string { return "" },
			PortBindings:  []portBinding{elasticsearch_logstashPortBinding},
			Volumes:       map[string]string{"data": "/opt/elasticsearch-1.3.1/data"},
			Configuration: make(map[string]interface{}),
			HealthChecks:  healthChecks,
		},
	)
	if err != nil {
		glog.Fatal("Error initializing elasticsearch container: %s", err)
	}
	envPerService[serviceName]["ES_JAVA_OPTS"] = "-Xmx4g"
	elasticsearch_logstash.Command = func() string {
		clusterArg := ""
		if clusterName, ok := elasticsearch_logstash.Configuration["cluster"]; ok {
			clusterArg = fmt.Sprintf(" -Des.cluster.name=%s ", clusterName)
		}
		return fmt.Sprintf(`/opt/elasticsearch-1.3.1/bin/elasticsearch -Des.node.name=%s %s`, elasticsearch_logstash.Name, clusterArg)
	}
}

// elasticsearchHealthCheck() determines if elasticsearch is healthy
func elasticsearchHealthCheck(port int) func() error {
	return func() error {
		lastError := time.Now()
		minUptime := time.Second * 2
		baseUrl := fmt.Sprintf("http://localhost:%d", port)

		for {
			healthResponse, err := getElasticHealth(baseUrl)
			if err == nil && (healthResponse.Status == "green" || healthResponse.Status == "yellow") {
				break
			} else {
				lastError = time.Now()
				glog.Infof("Still trying to connect to elasticsearch at %s: %v: %s", baseUrl, err, healthResponse.Status)
				glog.V(1).Infof("Still trying to connect to elasticsearch at %s: %v: %s", baseUrl, err, healthResponse.Status)
			}

			if time.Since(lastError) > minUptime {
				break
			}
			time.Sleep(time.Millisecond * 1000)
		}
		glog.V(1).Infof("elasticsearch running browser at %s/_plugin/head/", baseUrl)
		return nil
	}
}

func getElasticHealth(baseUrl string) (cluster.ClusterHealthResponse, error) {
	healthUrl := fmt.Sprintf("%s/_cluster/health?pretty=true", baseUrl)
	healthResponse := cluster.ClusterHealthResponse{}
	healthResponse.Status = "unknown"
	response, err := http.Get(healthUrl)
	if err != nil {
		return healthResponse, err
	}

	defer response.Body.Close()
	if response.StatusCode != 200 {
		err = fmt.Errorf("Failed to HTTP GET for %s, return code = %d", healthUrl, response.StatusCode)
		return healthResponse, err
	}

	body, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		err = fmt.Errorf("Failed to read HTTP response from %s: %s", healthUrl, readErr)
		return healthResponse, err
	}

	jsonErr := json.Unmarshal(body, &healthResponse)
	if jsonErr != nil {
		err = fmt.Errorf("Failed to unmarshall response to healthcheck from %s: %s", healthUrl, jsonErr)
	}

	return healthResponse, err
}

func PurgeLogstashIndices(days int, gb int) {
	iservice := elasticsearch_logstash
	port := iservice.PortBindings[0].HostPort
	glog.Infof("Purging logstash entries older than %d days", days)
	err := iservice.Exec([]string{
		"/usr/local/bin/curator", "--port", fmt.Sprintf("%d", port),
		"delete", "indices", "--older-than", fmt.Sprintf("%d", days), "--time-unit", "days", "--timestring", "%Y.%m.%d"})
	if err != nil {
		glog.Errorf("Unable to purge logstash entries older than %d days: %s", days, err)
	}
	glog.Infof("Purging oldest logstash entries to limit disk usage to %d GB.", gb)
	err = iservice.Exec([]string{
		"/usr/local/bin/curator", "--port", fmt.Sprintf("%d", port),
		"delete", "--disk-space", fmt.Sprintf("%d", gb), "indices", "--all-indices"})
	if err != nil {
		glog.Errorf("Unable to purge logstash entries to limit disk usage to %d GB: %s", gb, err)
	}
}
