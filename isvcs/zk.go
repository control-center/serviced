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
	"fmt"
	"time"

	"github.com/control-center/go-zookeeper/zk"
	"github.com/zenoss/glog"
)

var Zookeeper IServiceDefinition
var zookeeper *IService

const (
	ZK_CLIENT_PORT = 2181
	ZK_EXHIBITOR_PORT = 12181
	ZK_PEER_PORT = 2888
	ZK_LEADER_PORT = 3888
)

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
				HostPort:       ZK_CLIENT_PORT,
			},
			// exhibitor port
			{
				HostIp:         "127.0.0.1",
				HostIpOverride: "SERVICED_ISVC_ZOOKEEPER_PORT_12181_HOSTIP",
				HostPort:       ZK_EXHIBITOR_PORT,
			},
			// peer port
			{
				HostIp:         "0.0.0.0",
				HostIpOverride: "",
				HostPort:       ZK_PEER_PORT,
			},
			// leader port
			{
				HostIp:         "0.0.0.0",
				HostIpOverride: "",
				HostPort:       ZK_LEADER_PORT,
			},
		},
		Volumes: map[string]string{"data": "/var/zookeeper"},
	}

	defaultHealthCheck := healthCheckDefinition{
		HealthCheck: zkHealthCheck,
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
	for {
		// establish a zookeeper connection
		conn, ec, err := zk.Connect([]string{fmt.Sprintf("127.0.0.1:%d", ZK_CLIENT_PORT)}, time.Second*10)
		defer func() {
			if conn != nil {
				conn.Close()
			}
		}()
		if err != nil {
			glog.V(1).Infof("Could not connect to zookeeper: %s", err)
			time.Sleep(1 * time.Second)
		} else {
			//wait for session
			sesstionTimeout := 5 * time.Second
			sessionTimer := time.NewTimer(sesstionTimeout)
			defer sessionTimer.Stop()
			timedOut := false
			for !timedOut {
				select {
				case e := <-ec:
					if e.State == zk.StateHasSession {
						// success
						glog.V(1).Infof("Zookeeper running, browser at http://localhost:12181/exhibitor/v1/ui/index.html")
						return nil
					}
				case <-halt:
					glog.V(1).Infof("Quit healthcheck for zookeeper")
					return nil
				case <-sessionTimer.C:
					//Fall through loop and try again
					glog.V(1).Infof("ZK Session not available in %s", sesstionTimeout)
					timedOut = true
				}
			}
		}
	}
}
