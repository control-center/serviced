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
	"github.com/Sirupsen/logrus"

	"io/ioutil"
	"net"
	"time"
	"github.com/control-center/serviced/config"
	"fmt"
)

var Zookeeper IServiceDefinition
var zookeeper *IService

const QUORUM_HEALTHCHECK_NAME = "Participating in Quorum"

func initZK() {
	var err error

	// build the list of ZooKeeper connect strings for use in health checks.
	zkConnectStrings := config.GetOptions().Zookeepers

	if len(zkConnectStrings) == 0 {
		zkConnectStrings = append(zkConnectStrings, "127.0.0.1:2181")
	}

	// Build the service definition for the Zookeeper instance.
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

	// setup the health check definitions
	Zookeeper.HealthChecks 	= make(map[string]healthCheckDefinition)

	for _, zkIP := range zkConnectStrings {

		defaultHealthCheck := healthCheckDefinition{
			healthCheck: SetZKHealthCheck(zkIP),
			Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
			Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
		}

		quorumHealthCheck := healthCheckDefinition{
			healthCheck: SetZKQuorumCheck(zkIP),
			Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
			Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
		}

		Zookeeper.HealthChecks[DEFAULT_HEALTHCHECK_NAME] = defaultHealthCheck
		Zookeeper.HealthChecks[QUORUM_HEALTHCHECK_NAME] = quorumHealthCheck
	}

	zookeeper, err = NewIService(Zookeeper)

	if err != nil {
		log.WithError(err).Fatal("Unable to initialize ZooKeeper internal service container")
	}
}

// This function sets up a ZooKeeper health check function parameterized with the connection string eg: "127.0.0.1:2181".
func SetZKHealthCheck(connectString string) HealthCheckFunction {
	return func(halt <-chan struct{}) error {
		for {
			select {
			case <-halt:
				log.Debug("Stopped health checks for ZooKeeper")
				return nil
			default:
				// Try ruok.

				ruok, err := zkFourLetterWord(connectString, "ruok", time.Second*2)
				if err != nil {
					log.WithError(err).Debug("No response to ruok from ZooKeeper")
					return fmt.Errorf("No response to \"ruok\" from ZooKeeper instance (%s)", connectString)
				}

				// ruok should respond either with "imok" or not at all.
				// If for some reason that isn't the case, there's a problem.
				if string(ruok) != "imok" {
					log.WithFields(logrus.Fields{
						"response": ruok,
					}).Debug("Improper response to ruok from ZooKeeper")
					return fmt.Errorf("Improper response to \"ruok\" from ZooKeeper instance (%s)", connectString)
				}

				log.Debug("ZooKeeper checked in healthy")

				return nil
			}
		}
	}
}

// This function sets up a ZooKeeper quorum check function parameterized with the connection string eg: "127.0.0.1:2181".
func SetZKQuorumCheck(connectString string) HealthCheckFunction {
	return func(halt <-chan struct{}) error {

		for {
			select {
			case <-halt:
				log.Debug("Stopped health checks for ZooKeeper")
				return nil
			default:
				// check the 'stat' command and see what the instance returns to the caller.
				stat, err := zkFourLetterWord(connectString, "stat", time.Second*2)
				if err != nil {
					log.WithError(err).Debug("No response to stat from ZooKeeper")
					return fmt.Errorf("No response to stat from ZooKeeper instance (%s)", connectString)
				}

				// If we get "This ZooKeeper instance is not currently serving requests", we know it's waiting for quorum and can at least note that in the logs.
				if string(stat) == "This ZooKeeper instance is not currently serving requests\n" {
					log.Debug("ZooKeeper is running, but still establishing a quorum")
					return fmt.Errorf("ZooKeeper instance (%s) is running, but still establishing a quorum", connectString)
				}

				log.Debug("ZooKeeper Quorum checked in healthy")

				return nil
			}
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
