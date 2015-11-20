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

package isvcs

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"

	"github.com/control-center/serviced/volume"
	"github.com/zenoss/elastigo/cluster"
	"github.com/zenoss/glog"

	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	ESRed ESHealth = iota
	ESYellow
	ESGreen
)

type ESHealth int

func GetHealth(health string) ESHealth {
	switch health {
	case "red":
		return ESRed
	case "yellow":
		return ESYellow
	case "green":
		return ESGreen
	}
	return ESHealth(-1)
}

func (health ESHealth) String() string {
	switch health {
	case ESRed:
		return "red"
	case ESYellow:
		return "yellow"
	case ESGreen:
		return "green"
	}
	return "unknown"
}

const DEFAULT_ES_STARTUP_TIMEOUT_SECONDS = 240 //default startup timeout in seconds (4 minutes)
const MIN_ES_STARTUP_TIMEOUT_SECONDS = 30      //minimum startup timeout in seconds

var elasticsearch_logstash *IService
var elasticsearch_serviced *IService

func initElasticSearch() {
	var serviceName string
	var err error

	serviceName = "elasticsearch-serviced"

	defaultHealthCheck := healthCheckDefinition{
		healthCheck: esHealthCheck(9200, ESYellow),
		Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
		Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
	}
	healthChecks := map[string]healthCheckDefinition{
		DEFAULT_HEALTHCHECK_NAME: defaultHealthCheck,
	}
	elasticsearch_servicedPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_ELASTICSEARCH_SERVICED_PORT_9200_HOSTIP",
		HostPort:       9200,
	}

	elasticsearch_serviced, err = NewIService(
		IServiceDefinition{
			ID:             ElasticsearchServicedISVC.ID,
			Name:           serviceName,
			Repo:           IMAGE_REPO,
			Tag:            IMAGE_TAG,
			Command:        func() string { return "" },
			PortBindings:   []portBinding{elasticsearch_servicedPortBinding},
			Volumes:        map[string]string{"data": "/opt/elasticsearch-0.90.9/data"},
			Configuration:  make(map[string]interface{}),
			HealthChecks:   healthChecks,
			StartupTimeout: time.Duration(DEFAULT_ES_STARTUP_TIMEOUT_SECONDS) * time.Second,
		},
	)
	if err != nil {
		glog.Fatalf("Error initializing elasticsearch container: %s", err)
	}
	elasticsearch_serviced.Command = func() string {
		clusterArg := ""
		if clusterName, ok := elasticsearch_serviced.Configuration["cluster"]; ok {
			clusterArg = fmt.Sprintf(" -Des.cluster.name=%s ", clusterName)
		}
		return fmt.Sprintf(`exec /opt/elasticsearch-0.90.9/bin/elasticsearch -f -Des.node.name=%s %s`, elasticsearch_serviced.Name, clusterArg)
	}

	serviceName = "elasticsearch-logstash"
	logStashHealthCheck := defaultHealthCheck
	logStashHealthCheck.healthCheck = esHealthCheck(9100, ESYellow)
	healthChecks = map[string]healthCheckDefinition{
		DEFAULT_HEALTHCHECK_NAME: logStashHealthCheck,
	}
	elasticsearch_logstashPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_ELASTICSEARCH_LOGSTASH_PORT_9100_HOSTIP",
		HostPort:       9100,
	}

	elasticsearch_logstash, err = NewIService(
		IServiceDefinition{
			ID:             ElasticsearchLogStashISVC.ID,
			Name:           serviceName,
			Repo:           IMAGE_REPO,
			Tag:            IMAGE_TAG,
			Command:        func() string { return "" },
			PortBindings:   []portBinding{elasticsearch_logstashPortBinding},
			Volumes:        map[string]string{"data": "/opt/elasticsearch-1.3.1/data"},
			Configuration:  make(map[string]interface{}),
			HealthChecks:   healthChecks,
			Recover:        recoverES,
			StartupTimeout: time.Duration(DEFAULT_ES_STARTUP_TIMEOUT_SECONDS) * time.Second,
		},
	)
	if err != nil {
		glog.Fatalf("Error initializing elasticsearch container: %s", err)
	}
	envPerService[serviceName]["ES_JAVA_OPTS"] = "-Xmx4g"
	elasticsearch_logstash.Command = func() string {
		clusterArg := ""
		if clusterName, ok := elasticsearch_logstash.Configuration["cluster"]; ok {
			clusterArg = fmt.Sprintf(" -Des.cluster.name=%s ", clusterName)
		}
		return fmt.Sprintf(`exec /opt/elasticsearch-1.3.1/bin/elasticsearch -Des.node.name=%s %s`, elasticsearch_logstash.Name, clusterArg)
	}
}

func recoverES(path string) error {
	if err := func() error {
		file, err := os.Create(path + "-backup.tgz")
		if err != nil {
			glog.Errorf("Could not create backup for %s: %s", path, err)
			return err
		}
		defer file.Close()
		gz := gzip.NewWriter(file)
		defer gz.Close()
		tarfile := tar.NewWriter(gz)
		defer tarfile.Close()
		if err := volume.ExportDirectory(tarfile, path, filepath.Base(path)); err != nil {
			glog.Errorf("Could not backup %s: %s", path, err)
			return err
		}
		if err := volume.ExportFile(tarfile, path+".clustername", filepath.Base(path)+".clustername"); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		glog.Errorf("Could not remove %s: %s", path, err)
		return err
	}
	return nil
}

type esres struct {
	url      string
	response *cluster.ClusterHealthResponse
	err      error
}

func getESHealth(url string) <-chan esres {
	esresC := make(chan esres, 1)
	go func() {
		resp, err := http.Get(url)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			esresC <- esres{url, nil, err}
			return
		}
		if resp.StatusCode != 200 {
			esresC <- esres{url, nil, fmt.Errorf("received %d status code", resp.StatusCode)}
			return
		}
		var health cluster.ClusterHealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			esresC <- esres{url, nil, err}
			return
		}
		esresC <- esres{url, &health, nil}

	}()
	return esresC
}

func esHealthCheck(port int, minHealth ESHealth) HealthCheckFunction {
	return func(cancel <-chan struct{}) error {
		url := fmt.Sprintf("http://localhost:%d/_cluster/health", port)
		var r esres
		for {
			select {
			case r = <-getESHealth(url):
				if r.err != nil {
					glog.Warningf("Problem looking up %s: %s", r.url, r.err)
					break
				}
				if status := GetHealth(r.response.Status); status < minHealth {
					glog.Warningf("Received health status {%+v} at %s", r.response, r.url)
					break
				}
				return nil
			case <-cancel:
				glog.Infof("Cancel healthcheck for elasticsearch at %s", url)
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func PurgeLogstashIndices(days int, gb int) {
	iservice := elasticsearch_logstash
	port := iservice.PortBindings[0].HostPort
	prefix := []string{"/usr/bin/curator", "--port", fmt.Sprintf("%d", port)}

	glog.Infof("Purging logstash entries older than %d days", days)
	indices := []string{"indices", "--older-than", fmt.Sprintf("%d", days), "--time-unit", "days", "--timestring", "%Y.%m.%d"}
	if output, err := iservice.Exec(append(append(prefix, "delete"), indices...)); err != nil {
		if !(strings.Contains(string(output), "No indices found in Elasticsearch") ||
			strings.Contains(string(output), "No indices matched provided args")) {
			glog.Errorf("Unable to purge logstash entries older than %d days: %s", days, err)
		}
	}

	glog.Infof("Purging oldest logstash entries to limit disk usage to %d GB.", gb)
	indices = []string{"--disk-space", fmt.Sprintf("%d", gb), "indices", "--all-indices"}
	if output, err := iservice.Exec(append(append(prefix, "delete"), indices...)); err != nil {
		if !(strings.Contains(string(output), "No indices found in Elasticsearch") ||
			strings.Contains(string(output), "No indices matched provided args")) {
			glog.Errorf("Unable to purge logstash entries to limit disk usage to %d GB: %s", gb, err)
		}
	}
}
