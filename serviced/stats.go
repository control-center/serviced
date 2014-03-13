// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/cgroup"
	"github.com/rcrowley/go-metrics"
	"net/http"
	"bytes"
	"strconv"
	"strings"
	"time"
)

// StatsReporter collects and posts serviced stats to the TSDB.
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

// StartReporting runs report every d time units. Blocks. Should be run as a goroutine.
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
	sr.updateStats()
	stats := sr.gatherStats(t)
	sr.post(stats)
}

// Updates the default registry.
func (sr StatsReporter) updateStats() {
	cpuacctStat := cgroup.ReadCpuacctStat("")
	metrics.GetOrRegisterGauge("CpuacctStat.system", metrics.DefaultRegistry).Update(cpuacctStat.System)
	metrics.GetOrRegisterGauge("CpuacctStat.user", metrics.DefaultRegistry).Update(cpuacctStat.User)
	memoryStat := cgroup.ReadMemoryStat("")
	metrics.GetOrRegisterGauge("MemoryStat.pgfault", metrics.DefaultRegistry).Update(memoryStat.Pgfault)
	metrics.GetOrRegisterGauge("MemoryStat.rss", metrics.DefaultRegistry).Update(memoryStat.Rss)
}

// Fills out the metric consumer format.
func (sr StatsReporter) gatherStats(t time.Time) []containerStat {
	stats := []containerStat{}
	reg, _ := metrics.DefaultRegistry.(*metrics.StandardRegistry)
	reg.Each(func(n string, i interface{}) {
		metric := i.(metrics.Gauge)
		tagmap := make(map[string]string)
		tagmap["datasource"] = n
		tagmap["uuid"] = n
		stats = append(stats, containerStat{n, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
	})
	return stats
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
		return fmt.Errorf("couldn't post stats: ", resp.Status)
	}
	resp.Body.Close()
	return nil
}

