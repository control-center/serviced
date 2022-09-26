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
	"github.com/control-center/serviced/config"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var tsdbStore = NewOpenTSDBStatsStore()

type OpenTSDBStats struct {
	gc_young_count int
	gc_young_time  float64
	gc_old_count   int
	gc_old_time    float64
	threads        int
}

type OpenTSDBStatsCache struct {
	stats map[string]OpenTSDBStats

	mux sync.RWMutex
}

func NewOpenTSDBStatsCache() *OpenTSDBStatsCache {
	return &OpenTSDBStatsCache{
		stats: make(map[string]OpenTSDBStats),
	}
}

func (c *OpenTSDBStatsCache) ReadAll() []OpenTSDBStats {
	c.mux.RLock()
	defer c.mux.RUnlock()

	stats := []OpenTSDBStats{}
	for _, s := range c.stats {
		stats = append(stats, s)
	}

	return stats
}

func (c *OpenTSDBStatsCache) Read() (OpenTSDBStats, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if s, ok := c.stats[opentsdb.ID]; ok {
		return s, nil
	}

	return OpenTSDBStats{}, errors.New(fmt.Sprintf("Unable to find %s in cache", opentsdb.ID))
}

func (c *OpenTSDBStatsCache) Write(stats OpenTSDBStats) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.stats[opentsdb.ID] = stats
}

type OpenTSDBStatsStore interface {
	ReadAll() []OpenTSDBStats
	Read() (OpenTSDBStats, error)
	Write(stats OpenTSDBStats)
	WriteAll(stats []OpenTSDBStats)
}

type openTSDBStatsStore struct {
	cache *OpenTSDBStatsCache
}

func NewOpenTSDBStatsStore() OpenTSDBStatsStore {
	return &openTSDBStatsStore{NewOpenTSDBStatsCache()}
}

func (ss *openTSDBStatsStore) ReadAll() []OpenTSDBStats {
	return ss.cache.ReadAll()
}

func (ss *openTSDBStatsStore) Read() (OpenTSDBStats, error) {
	return ss.cache.Read()
}

func (ss *openTSDBStatsStore) Write(stats OpenTSDBStats) {
	ss.cache.Write(stats)
	writeTSDBToOpenTSDB([]OpenTSDBStats{stats})
}

func (ss *openTSDBStatsStore) WriteAll(stats []OpenTSDBStats) {
	for _, s := range stats {
		ss.cache.Write(s)
	}

	writeTSDBToOpenTSDB(stats)
}

func newOpenTSDBMetric(name string, value string, timestamp int64) metric {
	tags := map[string]string{
		"isvc":                    "true",
		"controlplane_service_id": opentsdb.ID,
		"daemon":                  "opentsdb",
	}

	return metric{name, value, timestamp, tags}
}

func writeTSDBToOpenTSDB(stats []OpenTSDBStats) {
	t := time.Now()

	metrics := []metric{}

	for _, s := range stats {
		threadsMetric := newOpenTSDBMetric(
			"isvcs.jvm.threads.count",
			strconv.Itoa(s.threads),
			t.Unix(),
		)

		metrics = append(metrics, threadsMetric)

		gcOldTimeMetric := newOpenTSDBMetric(
			"isvcs.jvm.gc.old.collection_time",
			fmt.Sprintf("%.2f", s.gc_old_time),
			t.Unix(),
		)

		metrics = append(metrics, gcOldTimeMetric)

		gcOldCountMetric := newOpenTSDBMetric(
			"isvcs.jvm.gc.old.collection_count",
			strconv.Itoa(s.gc_old_count),
			t.Unix(),
		)

		metrics = append(metrics, gcOldCountMetric)

		gcYoungTimeMetric := newOpenTSDBMetric(
			"isvcs.jvm.gc.young.collection_time",
			fmt.Sprintf("%.2f", s.gc_young_time),
			t.Unix(),
		)

		metrics = append(metrics, gcYoungTimeMetric)

		gcYoungCountMetric := newOpenTSDBMetric(
			"isvcs.jvm.gc.young.collection_count",
			strconv.Itoa(s.gc_young_count),
			t.Unix(),
		)

		metrics = append(metrics, gcYoungCountMetric)
	}

	err := postDataToOpenTSDB(metrics)
	if err != nil {
		log.WithFields(logrus.Fields{
			"numberOfMetrics": len(metrics),
		}).WithError(err).Warn("Unable to write OpenTSDB metrics to OpenTSDB")
	}
}

func queryOpenTSDBStats(address string) OpenTSDBStats {
	logger := log.WithField("opentsdb_address", address)
	stats := OpenTSDBStats{}

	url := address + "/api/stats/threads"
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		logger.WithError(err).Warnf("Unable to create request to %s", url)
		return stats
	}

	options := config.GetOptions()
	req.SetBasicAuth(options.IsvcsOpenTsdbUsername, options.IsvcsOpenTsdbPasswd)
	threads_resp, err := http.DefaultClient.Do(req)

	if err != nil {
		logger.WithError(err).Warn("Unable to get OpenTSDB threads")
		return stats
	}
	defer threads_resp.Body.Close()
	threads_body, err := ioutil.ReadAll(threads_resp.Body)
	if err != nil {
		logger.WithError(err).Warn("Unable to read OpenTSDB trheads")
		return stats
	}
	var threads_array []interface{}
	json.Unmarshal([]byte(threads_body), &threads_array)
	stats.threads = len(threads_array)

	url = address + "/api/stats/jvm"
	req, err = http.NewRequest("GET", url, http.NoBody)

	if err != nil {
		logger.WithError(err).Warnf("Unable to create request to %s", url)
		return stats
	}

	req.SetBasicAuth(options.IsvcsOpenTsdbUsername, options.IsvcsOpenTsdbPasswd)
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		logger.WithError(err).Warn("Unable to get OpenTSDB jvm stats")
		return stats
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Warn("Unable to read OpenTSDB jvm stats")
		return stats
	}
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)
	gc, ok := result["gc"].(map[string]interface{})
	if !ok {
		logger.Warn("Unable to read OpenTSDB gc stats")
		logger.Debug(result)
		return stats
	}
	for key, value := range gc {
		gc_metrics := value.(map[string]interface{})
		if key == "ParNew" || key == "pSScavenge" {
			stats.gc_young_time = gc_metrics["collectionTime"].(float64) / 1000
			stats.gc_young_count = int(gc_metrics["collectionCount"].(float64))
		} else if key == "ConcurrentMarkSweep" || key == "pSMarkSweep" {
			stats.gc_old_time = gc_metrics["collectionTime"].(float64) / 1000
			stats.gc_old_count = int(gc_metrics["collectionCount"].(float64))
		}
	}
	return stats
}

// GetOpenTSDBCustomStats retrieves OpenTSDB specific stats form the OpenTSDB servers.
// This should be run as a separate go routine.
func GetOpenTSDBCustomStats(halt <-chan struct{}) error {
	timeout := 30 * time.Second
	timer := time.NewTimer(timeout)
	tsdb_addres := "http://localhost:4242"

	for {
		select {
		case <-timer.C:
			stats := []OpenTSDBStats{}
			stats = append(stats, queryOpenTSDBStats(tsdb_addres))
			tsdbStore.WriteAll(stats)
		case <-halt:
			log.Info("Stopped getting custom stats for OpenTSDB")
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
