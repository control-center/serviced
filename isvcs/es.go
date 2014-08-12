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
	"github.com/control-center/serviced/utils"

	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

var elasticsearch_logstash *Container
var elasticsearch_serviced *Container

func init() {
	var err error
	elasticsearch_serviced, err = NewContainer(
		ContainerDescription{
			Name:        "elasticsearch-serviced",
			Repo:        IMAGE_REPO,
			Tag:         IMAGE_TAG,
			Command:     `/opt/elasticsearch-0.90.9/bin/elasticsearch -f`,
			Ports:       []int{9200},
			Volumes:     map[string]string{"data": "/opt/elasticsearch-0.90.9/data"},
			HealthCheck: elasticsearchHealthCheck(9200),
		},
	)
	if err != nil {
		glog.Fatal("Error initializing elasticsearch container: %s", err)
	}

	elasticsearch_logstash, err = NewContainer(
		ContainerDescription{
			Name:        "elasticsearch-logstash",
			Repo:        IMAGE_REPO,
			Tag:         IMAGE_TAG,
			Command:     `/opt/elasticsearch-1.3.1/bin/elasticsearch`,
			Ports:       []int{9100},
			Volumes:     map[string]string{"data": "/opt/elasticsearch-1.3.1/data"},
			HealthCheck: elasticsearchHealthCheck(9100),
		},
	)
	if err != nil {
		glog.Fatal("Error initializing elasticsearch container: %s", err)
	}
}

// getEnvVarInt() returns the env var as an int value or the defaultValue if env var is unset
func getEnvVarInt(envVar string, defaultValue int) int {
	envVarValue := os.Getenv(envVar)
	if len(envVarValue) > 0 {
		if value, err := strconv.Atoi(envVarValue); err != nil {
			glog.Errorf("Could not convert env var %s:%s to integer, error:%s", envVar, envVarValue, err)
			return defaultValue
		} else {
			return value
		}
	}
	return defaultValue
}

// elasticsearchHealthCheck() determines if elasticsearch is healthy
func elasticsearchHealthCheck(port int) func() error {
	return func() error {
		start := time.Now()
		lastError := time.Now()
		minUptime := time.Second * 2
		timeout := time.Second * time.Duration(getEnvVarInt("ES_STARTUP_TIMEOUT", 600))
		baseUrl := fmt.Sprintf("http://localhost:%d", port)

		schemaFile := path.Join(utils.ResourcesDir(), "controlplane.json")

		for {
			if healthResponse, err := cluster.Health(true); err == nil && (healthResponse.Status == "green" || healthResponse.Status == "yellow") {
				if buffer, err := os.Open(schemaFile); err != nil {
					glog.Fatalf("problem reading %s", err)
					return err
				} else {
					http.Post(baseUrl+"/controlplane", "application/json", buffer)
					buffer.Close()
				}
			} else {
				lastError = time.Now()
				glog.V(2).Infof("Still trying to connect to elasticsearch container: %v: %s", err, healthResponse)
			}
			if time.Since(lastError) > minUptime {
				break
			}
			if time.Since(start) > timeout {
				return fmt.Errorf("Could not startup elasticsearch container.  waited timeout:%v", timeout)
			}
			time.Sleep(time.Millisecond * 1000)
		}
		glog.Info("elasticsearch container started, browser at %s/_plugin/head/", baseUrl)
		return nil
	}
}
