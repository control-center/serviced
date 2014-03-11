// // Copyright 2014, The Serviced Authors. All rights reserved.
// // Use of this source code is governed by a
// // license that can be found in the LICENSE file.

// // Package agent implements a service that runs on a serviced node. It is
// // responsible for ensuring that a particular node is running the correct services
// // and reporting the state and health of those services back to the master
// // serviced.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/zenoss/glog"
	"github.com/rcrowley/go-metrics"
	"io/ioutil"
	"net/http"
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"time"
)

type StatsReporter struct {
	destination string
	username    string
	password    string
}

type containerStat struct {
	Metric    string            `json:"metric"`
	Value     string            `json:"value"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

// Runs report every d. Blocks. Should be run as a goroutine.
func (sr StatsReporter) StartReporting(d time.Duration) {
	tc := time.Tick(d)
	for t := range tc {
		go sr.report(t)
	}
}

// Updates the default registry, fills out the metric consumer format, and posts
// the data to the TSDB.
func (sr StatsReporter) report(t time.Time) {
	fmt.Println("Reporting container stats at:", t)
	sr.UpdateSSKVint64("/sys/fs/cgroup/memory/memory.stat", "memory")
	sr.UpdateSSKVint64("/sys/fs/cgroup/cpuacct/cpuacct.stat", "cpu")
	sr.UpdateSSKVint64("/sys/fs/cgroup/blkio/blkio.sectors", "blkio")
	sr.UpdateSSKVint64("/sys/fs/cgroup/blkio/blkio.io_service_bytes", "blkio")
	sr.UpdateSSKVint64("/sys/fs/cgroup/blkio/blkio.io_serviced", "blkio")
	sr.UpdateSSKVint64("/sys/fs/cgroup/blkio/blkio.io_queued", "blkio")
	sr.UpdateSSKVint64("/sys/fs/cgroup/memory/lxc/memory.stat", "memory")
	sr.UpdateSSKVint64("/sys/fs/cgroup/cpuacct/lxc/cpuacct.stat", "cpu")
	sr.UpdateSSKVint64("/sys/fs/cgroup/blkio/lxc/blkio.sectors", "blkio")
	sr.UpdateSSKVint64("/sys/fs/cgroup/blkio/lxc/blkio.io_service_bytes", "blkio")
	sr.UpdateSSKVint64("/sys/fs/cgroup/blkio/lxc/blkio.io_serviced", "blkio")
	sr.UpdateSSKVint64("/sys/fs/cgroup/blkio/lxc/blkio.io_queued", "blkio")
	stats := []containerStat{}
	reg, _ := metrics.DefaultRegistry.(*metrics.StandardRegistry)
	reg.Each(func(n string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Gauge:
			tagmap := make(map[string]string)
			tagmap["datasource"] = n
			tagmap["uuid"] = n
			stats = append(stats, containerStat{n, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
		}
	})
	sr.post(stats)
}

// Send the list of stats to the TSDB.
func (sr StatsReporter) post(stats []containerStat) error {
	payload := map[string][]containerStat{"metrics": stats}
	data, err := json.Marshal(payload)
	if err != nil {
		glog.V(3).Info("Couldn't marshal stats: ", err)
		return err
	}
	statsreq, err := http.NewRequest("POST", sr.destination, bytes.NewBuffer(data))
	if err != nil {
		glog.V(3).Info("Couldn't create stats request: ", err)
		return err
	}
	statsreq.Header["User-Agent"] = []string{"Zenoss Metric Publisher"}
	statsreq.Header["Content-Type"] = []string{"application/json"}
	if glog.V(4) {
		glog.Info(string(data))
	}
	resp, reqerr := http.DefaultClient.Do(statsreq)
	if reqerr != nil {
		glog.V(3).Info("Couldn't post stats: ", reqerr)
		return reqerr
	}
	if strings.Contains(resp.Status, "200") == false {
		glog.V(3).Info("Non-success: ", resp.Status)
		return fmt.Errorf("Couldn't post stats: ", resp.Status)
	}
	resp.Body.Close()
	return nil
}

// Updates a list of metrics produced from a space-separated key-value pair
// file. The prefix is prepended to the key in the final metric name.
func (sr StatsReporter) UpdateSSKVint64(filename string, prefix string) {
	kv, _ := parseSSKVint64(filename)
	for k, v := range kv {
		name := prefix + "." + k
		gauge := metrics.GetOrRegisterGauge(name, metrics.DefaultRegistry)
		gauge.Update(v)
	}
}

// Parses a space-separated key-value pair file and returns a
// key(string):value(int64) mapping.
func parseSSKVint64(filename string) (map[string]int64, error) {
	if kv, err := parseSSKV(filename); err != nil {
		return nil, err
	} else {
		mapping := make(map[string]int64)
		for k, v := range kv {
			if n, err := strconv.ParseInt(v, 0, 64); err != nil {
				return nil, err
			} else {
				mapping[k] = n
			}
		}
		return mapping, nil
	}
	return nil, nil
}

// Parses a space-separated key-value pair file and returns a
// key(string):value(string) mapping.
func parseSSKV(filename string) (map[string]string, error) {
	if stats, err := ioutil.ReadFile(filename); err != nil {
		return nil, err
	} else {
		mapping := make(map[string]string)
		scanner := bufio.NewScanner(strings.NewReader(string(stats)))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, " ")
			mapping[parts[0]] = parts[1]
		}
		return mapping, nil
	}
}