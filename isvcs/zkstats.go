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

	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var store = NewZooKeeperStatsStore()

type ZooKeeperStats struct {
	Address     string
	InstanceID  int
	Connections int
	Mode        string
}

type ZooKeeperStatsCache struct {
	stats map[string]ZooKeeperStats

	mux sync.RWMutex
}

func NewZooKeeperStatsCache() *ZooKeeperStatsCache {
	return &ZooKeeperStatsCache{
		stats: make(map[string]ZooKeeperStats),
	}
}

func (c *ZooKeeperStatsCache) ReadAll() []ZooKeeperStats {
	c.mux.RLock()
	defer c.mux.RUnlock()

	stats := []ZooKeeperStats{}
	for _, s := range c.stats {
		stats = append(stats, s)
	}

	return stats
}

func (c *ZooKeeperStatsCache) Read(address string) (ZooKeeperStats, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if s, ok := c.stats[address]; ok {
		return s, nil
	}

	return ZooKeeperStats{}, errors.New(fmt.Sprintf("Unable to find %s in cache", address))
}

func (c *ZooKeeperStatsCache) Write(stats ZooKeeperStats) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.stats[stats.Address] = stats
}

type ZooKeeperStatsStore interface {
	ReadAll() []ZooKeeperStats
	Read(key ZooKeeperKey) (ZooKeeperStats, error)
	Write(stats ZooKeeperStats)
	WriteAll(stats []ZooKeeperStats)
}

type ZKstatsStore struct {
	cache *ZooKeeperStatsCache
}

func NewZooKeeperStatsStore() ZooKeeperStatsStore {
	return &ZKstatsStore{NewZooKeeperStatsCache()}
}

func (ss *ZKstatsStore) ReadAll() []ZooKeeperStats {
	return ss.cache.ReadAll()
}

func (ss *ZKstatsStore) Read(key ZooKeeperKey) (ZooKeeperStats, error) {
	return ss.cache.Read(key.Connection)
}

func (ss *ZKstatsStore) Write(stats ZooKeeperStats) {
	ss.cache.Write(stats)
	writeToOpenTSDB([]ZooKeeperStats{stats})
}

func (ss *ZKstatsStore) WriteAll(stats []ZooKeeperStats) {
	for _, s := range stats {
		ss.cache.Write(s)
	}

	writeToOpenTSDB(stats)
}

func newZooKeeperMetric(name string, value string, timestamp int64, instance int) metric {
	tags := map[string]string{
		"isvc":                    "true",
		"instanceid":              strconv.Itoa(instance),
		"controlplane_service_id": zookeeper.ID,
	}

	return metric{name, value, timestamp, tags}
}

func writeToOpenTSDB(stats []ZooKeeperStats) {
	t := time.Now()

	metrics := []metric{}

	for _, s := range stats {
		connectionsMetric := newZooKeeperMetric(
			"isvcs.zookeeper.connections",
			strconv.Itoa(s.Connections),
			t.Unix(),
			s.InstanceID,
		)

		metrics = append(metrics, connectionsMetric)

		leaderValue := "0"
		if s.Mode == "leader" {
			leaderValue = "2"
		} else if s.Mode == "follower" {
			leaderValue = "1"
		}

		roleMetric := newZooKeeperMetric(
			"isvcs.zookeeper.role",
			leaderValue,
			t.Unix(),
			s.InstanceID,
		)

		metrics = append(metrics, roleMetric)
	}

	err := postDataToOpenTSDB(metrics)
	if err != nil {
		log.WithFields(logrus.Fields{
			"numberOfMetrics": len(metrics),
		}).WithError(err).Warn("Unable to write ZooKeeper metrics to OpenTSDB")
	}
}

func queryZooKeeperStats(key ZooKeeperKey) ZooKeeperStats {
	logger := log.
		WithField("instanceid", key.InstanceID).
		WithField("connection", key.Connection)

	stats := ZooKeeperStats{Address: key.Connection, InstanceID: key.InstanceID}

	bytes, err := zkFourLetterWord(key.Connection, "srvr", 10*time.Second)
	if err != nil {
		logger.WithError(err).Warn("Unable to get ZooKeeper stats")
		return stats
	}

	lines := strings.Split(string(bytes[:]), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Connections:") {
			tokens := strings.SplitN(line, ":", 2)
			value, err := strconv.Atoi(strings.TrimSpace(tokens[1]))
			if err != nil {
				logger.WithError(err).Warn("Unable to get number of connections")
			}
			stats.Connections = value
			continue
		}

		if strings.HasPrefix(line, "Mode:") {
			tokens := strings.SplitN(line, ":", 2)
			if len(tokens) < 2 {
				logger.Warn("Unable to get mode")
			}
			stats.Mode = strings.TrimSpace(tokens[1])
		}
	}

	if len(lines) == 0 {
		logger.Warn("No ZooKeeper stats collected")
	}

	return stats
}

func GetZooKeeperStatsByID(instanceID int) (ZooKeeperStats, error) {
	for _, key := range GetZooKeeperKeys() {
		if key.InstanceID != instanceID {
			continue
		}

		return store.Read(key)
	}
	return ZooKeeperStats{}, errors.New(fmt.Sprintf("Unable to get stats for %v", instanceID))
}

// GetZooKeeperCustomStats retrieves ZooKeeper specific stats form the ZooKeeper servers.
// This should be run as a separate go routine.
func GetZooKeeperCustomStats(halt <-chan struct{}) error {
	timeout := 30 * time.Second
	timer := time.NewTimer(timeout)

	for {
		select {
		case <-timer.C:
			stats := []ZooKeeperStats{}
			for _, key := range GetZooKeeperKeys() {
				stats = append(stats, queryZooKeeperStats(key))
			}
			store.WriteAll(stats)
		case <-halt:
			log.Info("Stopped getting custom stats for ZooKeeper")
			return nil
		}

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		timer.Reset(timeout)
	}
}
