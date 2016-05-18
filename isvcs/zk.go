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
	"github.com/zenoss/glog"

	"io/ioutil"
	"net"
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
	for {
		select {
		case <-halt:
			glog.V(1).Infof("Quit healthcheck for zookeeper")
			return nil
		default:
			// Try ruok.
			ruok, err := zkFourLetterWord("127.0.0.1:2181", "ruok", time.Second*10)
			if err != nil {
				glog.Warningf("No response to ruok from ZooKeeper: %s", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// ruok should respond either with "imok" or not at all.
			// If for some reason that isn't the case, there's a problem.
			glog.V(2).Infof("ruok: \"%s\"", ruok)
			if string(ruok) != "imok" {
				glog.Warningf("Improper response to ruok from ZooKeeper: %s", ruok)
				time.Sleep(1 * time.Second)
				continue
			}

			// Since ruok works, try stat next.
			stat, err := zkFourLetterWord("127.0.0.1:2181", "stat", time.Second*10)
			if err != nil {
				glog.Warningf("No response to stat from ZooKeeper: %s", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// If we get "This ZooKeeper instance is not currently serving requests", we know it's waiting for quorum and can at least note that in the logs.
			glog.V(2).Infof("stat: \"%s\"", stat)
			if string(stat) == "This ZooKeeper instance is not currently serving requests\n" {
				glog.Warningf("ZooKeeper is running, but still establishing quorum.")
				time.Sleep(1 * time.Second)
				continue
			}
			// We can optionally parse stat for information including this node's role or the number of connections.

			return nil
		}
	}

}

// transcribed from upstream samuel/go-zookeeper
func zkFourLetterWord(server, command string, timeout time.Duration) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", server, timeout)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(timeout))

	_, err = conn.Write([]byte(command))
	if err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(timeout))

	resp, err := ioutil.ReadAll(conn)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
