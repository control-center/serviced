// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package stats collects serviced metrics and posts them to the TSDB.

package stats

import (
	"bytes"
	"encoding/json"
	"github.com/rcrowley/go-metrics"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/stats/cgroup"
	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/serviced/zzk"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StatsReporter collects and posts serviced stats to the TSDB.
type StatsReporter struct {
	destination  string
	closeChannel chan bool
	zkDAO        *zzk.ZkDao
}

type containerStat struct {
	Metric    string            `json:"metric"`
	Value     string            `json:"value"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

type registryKey struct {
	serviceID string
	instanceID int
}

var registries map[registryKey]metrics.Registry
var hostID string
var hostRegistry metrics.Registry

// getOrCreateRegistry returns a registry for a given service id or creates it
// if it doesn't exist.
func getOrCreateRegistry(serviceID string, instanceID int) metrics.Registry {
	key := registryKey{serviceID, instanceID}
	if registry, ok := registries[key]; ok {
		return registry
	}
	registries[key] = metrics.NewRegistry()
	return registries[key]
}

// NewStatsReporter creates a new StatsReporter and kicks off the reporting goroutine.
func NewStatsReporter(destination string, interval time.Duration, zkDAO *zzk.ZkDao) (*StatsReporter, error) {
	registries = make(map[registryKey]metrics.Registry)
	var err error
	hostID, err = utils.HostID()
	if err != nil {
		glog.Errorf("Could not determine host ID.")
		return nil, err
	}
	hostRegistry = getOrCreateRegistry("", 0)
	sr := StatsReporter{destination, make(chan bool), zkDAO}
	go sr.report(interval)
	return &sr, nil
}

// Close shuts down the reporting goroutine. Blocks waiting for the goroutine to signal that it
// is indeed shutting down.
func (sr StatsReporter) Close() {
	sr.closeChannel <- true
	_ = <-sr.closeChannel
}

// Updates the default registry, fills out the metric consumer format, and posts
// the data to the TSDB. Stops when close signal is received on closeChannel.
func (sr StatsReporter) report(d time.Duration) {
	tc := time.Tick(d)
	for {
		select {
		case _ = <-sr.closeChannel:
			glog.V(3).Info("Ceasing stat reporting.")
			sr.closeChannel <- true
			return
		case t := <-tc:
			glog.V(3).Info("Reporting container stats at:", t)
			sr.updateStats()
			stats := sr.gatherStats(t)
			err := sr.post(stats)
			if err != nil {
				glog.Errorf("Error reporting container stats.")
			}
		}
	}
}

// Updates the default registry.
func (sr StatsReporter) updateStats() {
	// Stats for host.
	if cpuacctStat, err := cgroup.ReadCpuacctStat(""); err != nil {
		glog.V(3).Info("Couldn't read CpuacctStat:", err)
	} else {
		metrics.GetOrRegisterGauge("CpuacctStat.system", hostRegistry).Update(cpuacctStat.System)
		metrics.GetOrRegisterGauge("CpuacctStat.user", hostRegistry).Update(cpuacctStat.User)
	}

	if memoryStat, err := cgroup.ReadMemoryStat(""); err != nil {
		glog.V(3).Info("Couldn't read MemoryStat:", err)
	} else {
		metrics.GetOrRegisterGauge("MemoryStat.pgfault", hostRegistry).Update(memoryStat.Pgfault)
		metrics.GetOrRegisterGauge("MemoryStat.rss", hostRegistry).Update(memoryStat.Rss)
	}

	if openFileDescriptorCount, err := GetOpenFileDescriptorCount(); err != nil {
		glog.V(3).Info("Couldn't get open file descriptor count", err)
	} else {
		metrics.GetOrRegisterGauge("Serviced.OpenFileDescriptors", hostRegistry).Update(openFileDescriptorCount)
	}
	// Stats for the containers.
	var running []*dao.RunningService
	sr.zkDAO.GetRunningServicesForHost(hostID, &running)
	for _, rs := range running {
		containerRegistry := getOrCreateRegistry(rs.ServiceID, rs.InstanceID)
		if cpuacctStat, err := cgroup.ReadCpuacctStat("/sys/fs/cgroup/cpuacct/docker/" + rs.DockerID + "/cpuacct.stat"); err != nil {
			glog.V(3).Info("Couldn't read CpuacctStat:", err)
		} else {
			metrics.GetOrRegisterGauge("CpuacctStat.system", containerRegistry).Update(cpuacctStat.System)
			metrics.GetOrRegisterGauge("CpuacctStat.user", containerRegistry).Update(cpuacctStat.User)
		}
		if memoryStat, err := cgroup.ReadMemoryStat("/sys/fs/cgroup/memory/docker/" + rs.DockerID + "/memory.stat"); err != nil {
			glog.V(3).Info("Couldn't read MemoryStat:", err)
		} else {
			metrics.GetOrRegisterGauge("MemoryStat.pgfault", containerRegistry).Update(memoryStat.Pgfault)
			metrics.GetOrRegisterGauge("MemoryStat.rss", containerRegistry).Update(memoryStat.Rss)
		}
	}
}

// Fills out the metric consumer format.
func (sr StatsReporter) gatherStats(t time.Time) []containerStat {
	stats := []containerStat{}
	for key, registry := range registries {
		reg, _ := registry.(*metrics.StandardRegistry)
		reg.Each(func(name string, i interface{}) {
			metric := i.(metrics.Gauge)
			tagmap := make(map[string]string)
			if key.serviceID != "" {
				tagmap["controlplane_service_id"] = key.serviceID
				tagmap["controlplane_instance_id"] = string(key.instanceID)
			}
			tagmap["controlplane_host_id"] = hostID
			stats = append(stats, containerStat{name, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
		})
	}
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
	resp, reqerr := http.DefaultClient.Do(statsreq)
	if reqerr != nil {
		glog.V(3).Info("Couldn't post stats: ", reqerr)
		return reqerr
	}
	if strings.Contains(resp.Status, "200 OK") == false {
		glog.Warningf("couldn't post stats: ", resp.Status)
		return nil
	}
	resp.Body.Close()
	return nil
}
