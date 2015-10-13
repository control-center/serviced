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
	"github.com/control-center/go-zookeeper/zk"
	"github.com/zenoss/glog"

	"time"
)

var Zookeeper IServiceDefinition
var zookeeper *IService

func initZK() {
	var err error
	Zookeeper = IServiceDefinition{
		ID:      ZookeeperISVC.ID,
		Name:    "zookeeper",
		Repo:    ZK_IMAGE_REPO,
		Tag:     ZK_IMAGE_TAG,
		Command: func() string { return "exec start-zookeeper" },
		PortBindings: []portBinding{
			// client port
			{
				HostIp:         "0.0.0.0",
				HostIpOverride: "",
				HostPort:       2181,
			},
			// exhibitor port
			{
				HostIp:         "127.0.0.1",
				HostIpOverride: "SERVICED_ISVC_ZOOKEEPER_PORT_12181_HOSTIP",
				HostPort:       12181,
			},
			// peer port
			{
				HostIp:         "0.0.0.0",
				HostIpOverride: "",
				HostPort:       2888,
			},
			// leader port
			{
				HostIp:         "0.0.0.0",
				HostIpOverride: "",
				HostPort:       3888,
			},
		},
		Volumes: map[string]string{"data": "/var/zookeeper"},
	}

	defaultHealthCheck := healthCheckDefinition{
		healthCheck: zkHealthCheck,
		Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
		Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
	}

	Zookeeper.HealthChecks = map[string]healthCheckDefinition{
		DEFAULT_HEALTHCHECK_NAME: defaultHealthCheck,
	}

	zookeeper, err = NewIService(Zookeeper)
	if err != nil {
		glog.Fatalf("Error initializing zookeeper container: %s", err)
	}
}

// a health check for zookeeper
func zkHealthCheck(halt <-chan struct{}) error {
	lastError := time.Now()
	minUptime := time.Second * 2
	zookeepers := []string{"127.0.0.1:2181"}

	for {
		if conn, _, err := zk.Connect(zookeepers, time.Second*10); err == nil {
			conn.Close()
		} else {
			conn.Close()
			glog.V(1).Infof("Could not connect to zookeeper: %s", err)
			lastError = time.Now()
		}
		// make sure that service has been good for at least minUptime
		if time.Since(lastError) > minUptime {
			break
		}

		select {
		case <-halt:
			glog.V(1).Infof("Quit healthcheck for zookeeper")
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
	glog.V(1).Info("zookeeper running, browser at http://localhost:12181/exhibitor/v1/ui/index.html")
	return nil
}
