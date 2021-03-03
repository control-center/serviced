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
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"

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

	elasticsearch_servicedPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_ELASTICSEARCH_SERVICED_PORT_9200_HOSTIP",
		HostPort:       9200,
	}

	defaultHealthCheck := healthCheckDefinition{
		healthCheck: esHealthCheck(getHostIp(elasticsearch_servicedPortBinding), 9200, ESYellow),
		Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
		Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
	}

	healthChecks := []map[string]healthCheckDefinition{
		{
			DEFAULT_HEALTHCHECK_NAME: defaultHealthCheck,
		},
	}

	elasticsearch_serviced, err = NewIService(
		IServiceDefinition{
			ID:             ElasticsearchServicedISVC.ID,
			Name:           serviceName,
			Repo:           IMAGE_REPO,
			Tag:            IMAGE_TAG,
			Command:        func() string { return "" },
			PortBindings:   []portBinding{elasticsearch_servicedPortBinding},
			Volumes:        map[string]string{"data": "/opt/elasticsearch-serviced/data"},
			Configuration:  make(map[string]interface{}),
			HealthChecks:   healthChecks,
			StartupTimeout: time.Duration(DEFAULT_ES_STARTUP_TIMEOUT_SECONDS) * time.Second,
			CustomStats:    GetElasticSearchCustomStats,
		},
	)
	if err != nil {
		log.WithFields(logrus.Fields{
			"isvc": elasticsearch_serviced.ID,
		}).WithError(err).Fatal("Unable to initialize internal service")
	}
	elasticsearch_serviced.Command = func() string {
		clusterArg := ""
		if clusterName, ok := elasticsearch_serviced.Configuration["cluster"]; ok {
			clusterArg = fmt.Sprintf(`-Ecluster.name="%s" `, clusterName)
		}
		cmd := fmt.Sprintf(`export JAVA_HOME=/usr/lib/jvm/jre-11; su elastic -c 'exec /opt/elasticsearch-serviced/bin/elasticsearch -Ecluster.initial_master_nodes="%s" -Enode.name="%s" %s'`,
			elasticsearch_serviced.Name, elasticsearch_serviced.Name, clusterArg)
		log.Infof("Build the command for running es-serviced: %s", cmd)
		return cmd
	}

	serviceName = "elasticsearch-logstash"

	elasticsearch_logstashPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_ELASTICSEARCH_LOGSTASH_PORT_9100_HOSTIP",
		HostPort:       9100,
	}

	logStashHealthCheck := defaultHealthCheck
	logStashHealthCheck.healthCheck = esHealthCheck(getHostIp(elasticsearch_logstashPortBinding), 9100, ESYellow)

	healthChecks = []map[string]healthCheckDefinition{
		{
			DEFAULT_HEALTHCHECK_NAME: logStashHealthCheck,
		},
	}

	elasticsearch_logstash, err = NewIService(
		IServiceDefinition{
			ID:             ElasticsearchLogStashISVC.ID,
			Name:           serviceName,
			Repo:           IMAGE_REPO,
			Tag:            IMAGE_TAG,
			Command:        func() string { return "" },
			PortBindings:   []portBinding{elasticsearch_logstashPortBinding},
			Volumes:        map[string]string{"data": "/opt/elasticsearch-logstash/data"},
			Configuration:  make(map[string]interface{}),
			HealthChecks:   healthChecks,
			Recover:        recoverES,
			StartupTimeout: time.Duration(DEFAULT_ES_STARTUP_TIMEOUT_SECONDS) * time.Second,
			StartupFailed:  getESShardStatus,
		},
	)
	if err != nil {
		log.WithFields(logrus.Fields{
			"isvc": elasticsearch_logstash.ID,
		}).WithError(err).Fatal("Unable to initialize internal service")
	}

	// This value will be overwritten by SERVICED_ISVCS_ENV_X in
	// /etc/default/serviced
	envPerService[serviceName]["ES_JAVA_OPTS"] = "-Xmx4g"
	elasticsearch_logstash.Command = func() string {
		nodeName := elasticsearch_logstash.Name
		clusterName := elasticsearch_logstash.Configuration["cluster"]
		return fmt.Sprintf("exec /opt/elasticsearch-logstash/bin/es-logstash-start.sh %s %s", nodeName, clusterName)
	}
}

func recoverES(path string) error {
	recoveryPath := path + "-backup"
	log := log.WithFields(logrus.Fields{
		"basepath":     path,
		"recoverypath": recoveryPath,
	})

	if _, err := os.Stat(recoveryPath); err == nil {
		log.Info("Overwriting existing recovery path")
		os.RemoveAll(recoveryPath)
	} else if !os.IsNotExist(err) {
		log.Debug("Could not stat recovery path")
		return err
	}

	if err := os.Rename(path, recoveryPath); err != nil {
		log.WithError(err).Debug("Could not recover elasticsearch")
		return err
	}
	log.Info("Moved and reset elasticsearch data")
	return nil
}

type esres struct {
	url      string
	response map[string]interface{}
	err      error
}

func getESShardStatus() {
	// try to get more information about how the shards are looking.
	// If some are 'UNASSIGNED', it may be possible to delete just those and restart
	host := elasticsearch_logstash.PortBindings[0].HostIp
	port := elasticsearch_logstash.PortBindings[0].HostPort
	url := fmt.Sprintf("http://%s:%d/_cat/shards", host, port)
	resp, err := http.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.WithError(err).Error("Failed to get ES shard status.")
	}
	output, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("Failed to get ES shard status.")
	} else {
		log.Warnf("Shard Status:\n%s", string(output))
	}
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

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			esresC <- esres{url, nil, err}
			return
		}
		esresC <- esres{url, health, nil}

	}()
	return esresC
}

func esHealthCheck(host string, port int, minHealth ESHealth) HealthCheckFunction {
	return func(cancel <-chan struct{}) error {
		url := fmt.Sprintf("http://%s:%d/_cluster/health", host, port)
		log := log.WithFields(logrus.Fields{
			"url":       url,
			"minhealth": minHealth,
		})
		var r esres
		for {
			select {
			case r = <-getESHealth(url):
				if r.err != nil {
					log.WithError(r.err).Debugf("Unable to check Elastic health: %s", r.err)
					break
				}
				if status := GetHealth(r.response["status"].(string)); status < minHealth {
					log.WithFields(logrus.Fields{
						"reported":              r.response["status"],
						"cluster_name":          r.response["cluster_name"],
						"timed_out":             r.response["timed_out"],
						"number_of_nodes":       r.response["number_of_nodes"],
						"number_of_data_nodes":  r.response["number_of_data_nodes"],
						"active_primary_shards": r.response["active_primary_shards"],
						"active_shards":         r.response["active_shards"],
						"relocating_shards":     r.response["relocating_shards"],
						"initializing_shards":   r.response["initializing_shards"],
						"unassigned_shards":     r.response["unassigned_shards"],
					}).Warn("Elastic health reported below minimum")
					break
				}
				return nil
			case <-cancel:
				log.Debug("Canceled health check for Elastic")
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

	log := log.WithFields(logrus.Fields{
		"maxagedays": days,
		"maxsizegb":  gb,
	})

	log.Debug("Purging Logstash entries older than max age")
	indices := []string{"indices", "--older-than", fmt.Sprintf("%d", days), "--time-unit", "days", "--timestring", "%Y.%m.%d"}
	if output, err := iservice.Exec(append(append(prefix, "delete"), indices...)); err != nil {
		if !(strings.Contains(string(output), "No indices found in Elasticsearch") ||
			strings.Contains(string(output), "No indices matched provided args")) {
			log.WithError(err).Warn("Unable to purge logstash entries older than max age")
		}
	}
	log.Info("Purged Logstash entries older than max age")

	log.Debug("Purging Logstash entries to be below max size")
	indices = []string{"--disk-space", fmt.Sprintf("%d", gb), "indices", "--all-indices"}
	if output, err := iservice.Exec(append(append(prefix, "delete"), indices...)); err != nil {
		if !(strings.Contains(string(output), "No indices found in Elasticsearch") ||
			strings.Contains(string(output), "No indices matched provided args")) {
			log.WithError(err).Warn("Unable to purge logstash entries to be below max size")
		}
	}
	log.Info("Purged Logstash entries to be below max size")
}
