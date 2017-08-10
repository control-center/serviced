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

	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/dao"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"
)

var Zookeeper IServiceDefinition
var zookeeper *IService
var zkConnectStrings []string

const QUORUM_HEALTHCHECK_NAME = "hasQuorum"

func initZK() {
	var err error

	// build the list of ZooKeeper connect strings for use in health checks.
	zkConnectStrings = append(zkConnectStrings, "127.0.0.1:2181")

	// iterate through the zookeepers slice and add to our local slice the instance that are remote.
	opts := config.GetOptions()

	for instIndex, configZKIP := range opts.Zookeepers {
		if instIndex == 0 {
			continue
		}

		zkConnectStrings = append(zkConnectStrings, configZKIP)
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
	Zookeeper.HealthChecks = make([]map[string]healthCheckDefinition, len(zkConnectStrings))
	var indexSlice []int

	for instIndex, zkIP := range zkConnectStrings {
		indexSlice = append(indexSlice, instIndex)

		Zookeeper.HealthChecks[instIndex] = make(map[string]healthCheckDefinition)

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

		Zookeeper.HealthChecks[instIndex][DEFAULT_HEALTHCHECK_NAME] = defaultHealthCheck
		Zookeeper.HealthChecks[instIndex][QUORUM_HEALTHCHECK_NAME] = quorumHealthCheck
	}

	zookeeper, err = NewIService(Zookeeper)

	if err != nil {
		log.WithError(err).Fatal("Unable to initialize ZooKeeper internal service container")
	}
}

// This function sets up a ZooKeeper health check function parameterized with the connection string eg: "127.0.0.1:2181".
func SetZKHealthCheck(connectString string) HealthCheckFunction {
	return func(halt <-chan struct{}) error {
		healthy := true
		times := -1
		logger := log.WithFields(logrus.Fields{"healthcheck": DEFAULT_HEALTHCHECK_NAME})

		for {
			if !healthy && times == 3 {
				logger.Warn("Unable to validate health of ZooKeeper. Retrying silently")
			}

			select {
			case <-halt:
				logger.Debug("Stopped health checks for ZooKeeper")
				return nil
			default:
				// Try ruok.
				times++

				ruok, err := zkFourLetterWord(connectString, "ruok", time.Second*10)
				if err != nil {
					logger.WithError(err).Debug("No response to ruok from ZooKeeper")
					healthy = false
					time.Sleep(1 * time.Second)
					continue
				}

				// ruok should respond either with "imok" or not at all.
				// If for some reason that isn't the case, there's a problem.
				if string(ruok) != "imok" {
					logger.WithFields(logrus.Fields{
						"response": ruok,
					}).Debug("Improper response to ruok from ZooKeeper")
					healthy = false
					time.Sleep(1 * time.Second)
					continue
				}

				logger.Debug("ZooKeeper checked in healthy")

				return nil
			}
		}
	}
}

// This function sets up a ZooKeeper quorum check function parameterized with the connection string eg: "127.0.0.1:2181".
func SetZKQuorumCheck(connectString string) HealthCheckFunction {
	return func(halt <-chan struct{}) error {
		healthy := true
		times := -1
		logger := log.WithFields(logrus.Fields{"healthcheck": QUORUM_HEALTHCHECK_NAME})

		for {
			if !healthy && times == 3 {
				logger.Warn("Unable to validate health of ZooKeeper. Retrying silently")
			}

			select {
			case <-halt:
				logger.Debug("Stopped health checks for ZooKeeper")
				return nil
			default:
				times++

				// check the 'stat' command and see what the instance returns to the caller.
				stat, err := zkFourLetterWord(connectString, "stat", time.Second*10)
				if err != nil {
					logger.WithError(err).Debug("No response to stat from ZooKeeper")
					healthy = false
					time.Sleep(1 * time.Second)
					continue
				}

				// If we get "This ZooKeeper instance is not currently serving requests", we know it's waiting for quorum and can at least note that in the logs.
				if string(stat) == "This ZooKeeper instance is not currently serving requests\n" {
					logger.Debug("ZooKeeper is running, but still establishing a quorum")
					healthy = false
					time.Sleep(1 * time.Second)
					continue
				}

				logger.Debug("ZooKeeper Quorum checked in healthy")

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

type ZooKeeperServerStats struct {
	InstanceID  int
	Connections int
	Mode        string
}

func GetAllZooKeeperServerStats() []ZooKeeperServerStats {
	stats := []ZooKeeperServerStats{}
	for _, key := range GetZooKeeperKeys() {
		stats = append(stats, GetZooKeeperServerStats(key.InstanceID, key.Connection))
	}

	return stats
}

func GetZooKeeperServerStatsByID(instanceID int) ZooKeeperServerStats {
	for _, key := range GetZooKeeperKeys() {
		if key.InstanceID != instanceID {
			continue
		}

		return GetZooKeeperServerStats(instanceID, key.Connection)
	}
	return ZooKeeperServerStats{}
}

func GetZooKeeperServerStats(instanceID int, connection string) ZooKeeperServerStats {
	logger := log.
		WithField("instanceid", instanceID).
		WithField("connection", connection)

	stats := ZooKeeperServerStats{InstanceID: instanceID}

	bytes, err := zkFourLetterWord(connection, "srvr", 1*time.Second)
	if err != nil {
		logger.WithError(err).Error("Unable to get ZooKeeper stats")
		return stats
	}

	lines := strings.Split(string(bytes[:]), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Connections:") {
			tokens := strings.SplitN(line, ":", 2)
			value, err := strconv.Atoi(strings.TrimSpace(tokens[1]))
			if err != nil {
				logger.WithError(err).Error("Unable to get number of connections")
			}
			stats.Connections = value
			logger = logger.WithField("connections", stats.Connections)
			continue
		}

		if strings.HasPrefix(line, "Mode:") {
			tokens := strings.SplitN(line, ":", 2)
			if len(tokens) < 2 {
				logger.Error("Unable to get mode")
			}
			stats.Mode = strings.TrimSpace(tokens[1])
			logger = logger.WithField("mode", stats.Mode)
		}
	}

	return stats
}

func BuildZooKeeperRunningInstance(instanceID int) dao.RunningService {
	newInst := &dao.RunningService{}

	*newInst = ZookeeperIRS
	newInst.InstanceID = instanceID

	return *newInst
}

func GetZooKeeperRunningInstances() []dao.RunningService {
	instances := []dao.RunningService{}

	// loop through and create an instance for each IP we have.
	for _, key := range GetZooKeeperKeys() {
		instances = append(instances, BuildZooKeeperRunningInstance(key.InstanceID))
	}

	return instances
}

type ZooKeeperInstance struct {
	dao.RunningService
	IP    string
	Port  int
	Stats ZooKeeperServerStats
}

func GetZooKeeperInstances() []ZooKeeperInstance {
	instances := []ZooKeeperInstance{}

	// loop through and create an instance for each IP we have.
	for _, key := range GetZooKeeperKeys() {
		logger := log.WithField("connection", key.Connection)

		tokens := strings.SplitN(key.Connection, ":", 2)
		if len(tokens) < 2 {
			logger.Error("Unable to parse ZooKeeper connection")
			continue
		}
		ip := tokens[0]

		port, err := strconv.Atoi(strings.TrimSpace(tokens[1]))
		if err != nil {
			logger.Info("Unable to set ZooKeeper port for stats")
		}

		instances = append(instances, ZooKeeperInstance{
			BuildZooKeeperRunningInstance(key.InstanceID),
			ip,
			port,
			GetZooKeeperServerStats(key.InstanceID, key.Connection)})
	}

	return instances
}

type ZooKeeperKey struct {
	InstanceID int
	Connection string
}

func GetZooKeeperKeys() []ZooKeeperKey {
	keys := []ZooKeeperKey{}
	opts := config.GetOptions()

	if len(opts.Zookeepers) == 0 {
		keys = append(keys, ZooKeeperKey{0, "127.0.0.1:2181"})
		return keys
	}

	for i, connection := range opts.Zookeepers {
		keys = append(keys, ZooKeeperKey{i, connection})
	}

	return keys
}
