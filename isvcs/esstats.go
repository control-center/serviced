// Copyright 2019 The Serviced Authors.
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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"strconv"
	"time"
)

var esStore = NewElasticSearchStatsStore()

type ElasticSearchStats struct {
	Address           string
	gc_young_count    int
	gc_young_time     float64
	gc_old_count      int
	gc_old_time       float64
	threads           int
}

type ElasticSearchStatsCache struct {
	stats map[string]ElasticSearchStats

	mux sync.RWMutex
}

func NewElasticSearchStatsCache() *ElasticSearchStatsCache {
	return &ElasticSearchStatsCache{
		stats: make(map[string]ElasticSearchStats),
	}
}

func (c *ElasticSearchStatsCache) ReadAll() []ElasticSearchStats {
	c.mux.RLock()
	defer c.mux.RUnlock()

	stats := []ElasticSearchStats{}
	for _, s := range c.stats {
		stats = append(stats, s)
	}

	return stats
}

func (c *ElasticSearchStatsCache) Read(address string) (ElasticSearchStats, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if s, ok := c.stats[address]; ok {
		return s, nil
	}

	return ElasticSearchStats{}, errors.New(fmt.Sprintf("Unable to find %s in cache", address))
}

func (c *ElasticSearchStatsCache) Write(stats ElasticSearchStats) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.stats[stats.Address] = stats
}

type ElasticSearchStatsStore interface {
	ReadAll() []ElasticSearchStats
	Read(address string) (ElasticSearchStats, error)
	Write(stats ElasticSearchStats)
	WriteAll(stats []ElasticSearchStats)
}

type elasticSearchStatsStore struct {
	cache *ElasticSearchStatsCache
}

func NewElasticSearchStatsStore() ElasticSearchStatsStore {
	return &elasticSearchStatsStore{NewElasticSearchStatsCache()}
}

func (ss *elasticSearchStatsStore) ReadAll() []ElasticSearchStats {
	return ss.cache.ReadAll()
}

func (ss *elasticSearchStatsStore) Read(address string) (ElasticSearchStats, error) {
	return ss.cache.Read(address)
}

func (ss *elasticSearchStatsStore) Write(stats ElasticSearchStats) {
	ss.cache.Write(stats)
	writeESToOpenTSDB([]ElasticSearchStats{stats})
}

func (ss *elasticSearchStatsStore) WriteAll(stats []ElasticSearchStats) {
	for _, s := range stats {
		ss.cache.Write(s)
	}

	writeESToOpenTSDB(stats)
}

func newElasticSearchMetric(name string, value string, timestamp int64, address string) metric {
	tags := map[string]string{
		"isvc":    "true",
	}
	if address == "http://127.0.0.1:9200" {
		tags["controlplane_service_id"] = elasticsearch_serviced.ID
	} else if address == "http://127.0.0.1:9100" {
		tags["controlplane_service_id"] = elasticsearch_logstash.ID
	}

	return metric{name, value, timestamp, tags}
}

func writeESToOpenTSDB(stats []ElasticSearchStats) {
	t := time.Now()

	metrics := []metric{}

	for _, s := range stats {
		threadsMetric := newElasticSearchMetric(
			"isvcs.jvm.threads.count",
			strconv.Itoa(s.threads),
			t.Unix(),
			s.Address,
		)

		metrics = append(metrics, threadsMetric)

		gcOldTimeMetric := newElasticSearchMetric(
			"isvcs.jvm.gc.old.collection_time",
			fmt.Sprintf("%.2f", s.gc_old_time),
			t.Unix(),
			s.Address,
		)

		metrics = append(metrics, gcOldTimeMetric)

		gcOldCountMetric := newElasticSearchMetric(
			"isvcs.jvm.gc.old.collection_count",
			strconv.Itoa(s.gc_old_count),
			t.Unix(),
			s.Address,
		)

		metrics = append(metrics, gcOldCountMetric)

		gcYoungTimeMetric := newElasticSearchMetric(
			"isvcs.jvm.gc.young.collection_time",
			fmt.Sprintf("%.2f", s.gc_young_time),
			t.Unix(),
			s.Address,
		)

		metrics = append(metrics, gcYoungTimeMetric)

		gcYoungCountMetric := newElasticSearchMetric(
			"isvcs.jvm.gc.young.collection_count",
			strconv.Itoa(s.gc_young_count),
			t.Unix(),
			s.Address,
		)

		metrics = append(metrics, gcYoungCountMetric)
	}

	err := postDataToOpenTSDB(metrics)
	if err != nil {
		log.WithFields(logrus.Fields{
			"numberOfMetrics": len(metrics),
		}).WithError(err).Warn("Unable to write ElasticSearch metrics to OpenTSDB")
	}
}

func queryElasticSearchStats(address string) ElasticSearchStats {
	logger := log.WithField("elasticsearch_address", address)
	stats := ElasticSearchStats{Address: address}

	resp, err := http.Get(address + "/_nodes/stats?jvm=true")
	if err != nil {
		logger.WithError(err).Warn("Unable to get ElasticSearch stats")
		return stats
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Warn("Unable to read ElasticSearch stats")
		return stats
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)
	nodes := result["nodes"].(map[string]interface{})
	for _, value := range nodes {
		jvm_metrics := value.(map[string]interface{})["jvm"].(map[string]interface{})
		thread_count := jvm_metrics["threads"].(map[string]interface{})["count"].(float64)
		stats.threads += int(thread_count)
		gc_collectors := jvm_metrics["gc"].(map[string]interface{})["collectors"].(map[string]interface{})
		for key, value := range gc_collectors {
			gc_metrics := value.(map[string]interface{})
			if key == "young" || key == "ParNew" {
				stats.gc_young_count += int(gc_metrics["collection_count"].(float64))
				stats.gc_young_time += gc_metrics["collection_time_in_millis"].(float64) / 1000
			} else if key == "old" || key == "ConcurrentMarkSweep" {
				stats.gc_old_count += int(gc_metrics["collection_count"].(float64))
				stats.gc_old_time += gc_metrics["collection_time_in_millis"].(float64) / 1000
			}
		}
	}

	return stats
}

// GetElasticSearchCustomStats retrieves ElasticSearch specific stats form the ElasticSearch servers.
// This should be run as a separate go routine.
func GetElasticSearchCustomStats(halt <-chan struct{}) error {
	timeout := 30 * time.Second
	timer := time.NewTimer(timeout)
	es_addresses := [2]string{"http://127.0.0.1:9200", "http://127.0.0.1:9100"}

	for {
		select {
		case <-timer.C:
			stats := []ElasticSearchStats{}
			for key := range es_addresses {
				stats = append(stats, queryElasticSearchStats(es_addresses[key]))
			}
			esStore.WriteAll(stats)
		case <-halt:
			log.Info("Stopped getting custom stats for ElasticSearch")
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
