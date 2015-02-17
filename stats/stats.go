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

// Package stats collects serviced metrics and posts them to the TSDB.

package stats

import (
	"github.com/control-center/go-procfs/linux"
	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/stats/cgroup"
	"github.com/control-center/serviced/utils"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/rcrowley/go-metrics"
	"github.com/zenoss/glog"

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// StatsReporter collects and posts serviced stats to the TSDB.
type StatsReporter struct {
	destination         string
	closeChannel        chan bool
	conn                coordclient.Connection
	containerRegistries map[registryKey]metrics.Registry
	hostID              string
	hostRegistry        metrics.Registry
	sync.Mutex
}

// Sample is a single metric measurement
type Sample struct {
	Metric    string            `json:"metric"`
	Value     string            `json:"value"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

type registryKey struct {
	serviceID  string
	instanceID int
}

// NewStatsReporter creates a new StatsReporter and kicks off the reporting goroutine.
func NewStatsReporter(destination string, interval time.Duration, conn coordclient.Connection) (*StatsReporter, error) {
	hostID, err := utils.HostID()
	if err != nil {
		glog.Errorf("Could not determine host ID.")
		return nil, err
	}
	if conn == nil {
		glog.Errorf("conn can not be nil")
		return nil, fmt.Errorf("conn can not be nil")
	}
	sr := StatsReporter{
		destination:         destination,
		closeChannel:        make(chan bool),
		conn:                conn,
		containerRegistries: make(map[registryKey]metrics.Registry),
		hostID:              hostID,
	}

	sr.hostRegistry = metrics.NewRegistry()
	go sr.report(interval)
	return &sr, nil
}

// getOrCreateContainerRegistry returns a registry for a given service id or creates it
// if it doesn't exist.
func (sr *StatsReporter) getOrCreateContainerRegistry(serviceID string, instanceID int) metrics.Registry {
	key := registryKey{serviceID, instanceID}
	if registry, ok := sr.containerRegistries[key]; ok {
		return registry
	}
	sr.Lock()
	defer sr.Unlock()
	sr.containerRegistries[key] = metrics.NewRegistry()
	return sr.containerRegistries[key]
}

func (sr *StatsReporter) removeStaleRegistries(running *[]dao.RunningService) {
	// First build a list of what's actually running
	keys := make(map[string][]int)
	for _, rs := range *running {
		if instances, ok := keys[rs.ServiceID]; !ok {
			instances = []int{rs.InstanceID}
			keys[rs.ServiceID] = instances
		} else {
			keys[rs.ServiceID] = append(keys[rs.ServiceID], rs.InstanceID)
		}
	}
	// Now remove any keys that are in the registry but are no longer running
	// on this host
	sr.Lock()
	defer sr.Unlock()
	for key, _ := range sr.containerRegistries {
		if instances, ok := keys[key.serviceID]; !ok {
			delete(sr.containerRegistries, key)
		} else {
			var seen bool
			for _, instanceid := range instances {
				if instanceid == key.instanceID {
					seen = true
					break
				}
			}
			if !seen {
				delete(sr.containerRegistries, key)
			}
		}
	}
}

// Close shuts down the reporting goroutine. Blocks waiting for the goroutine to signal that it
// is indeed shutting down.
func (sr *StatsReporter) Close() {
	sr.closeChannel <- true
	_ = <-sr.closeChannel
}

// Updates the default registry, fills out the metric consumer format, and posts
// the data to the TSDB. Stops when close signal is received on closeChannel.
func (sr *StatsReporter) report(d time.Duration) {
	tc := time.Tick(d)
	glog.Infof("collecting internal metrics at %s intervals", d)
	for {
		select {
		case _ = <-sr.closeChannel:
			glog.V(3).Info("Ceasing stat reporting.")
			sr.closeChannel <- true
			return
		case t := <-tc:
			glog.V(1).Info("Reporting container stats at:", t)
			sr.updateStats()
			stats := sr.gatherStats(t)
			err := Post(sr.destination, stats)
			if err != nil {
				glog.Errorf("Error reporting container stats: %v", err)
			}
		}
	}
}

func (sr *StatsReporter) updateHostStats() {

	loadavg, err := linux.ReadLoadavg()
	if err != nil {
		glog.Errorf("could not read loadavg: %s", err)
		return
	}
	metrics.GetOrRegisterGaugeFloat64("load.avg1m", sr.hostRegistry).Update(float64(loadavg.Avg1m))
	metrics.GetOrRegisterGaugeFloat64("load.avg5m", sr.hostRegistry).Update(float64(loadavg.Avg5m))
	metrics.GetOrRegisterGaugeFloat64("load.avg10m", sr.hostRegistry).Update(float64(loadavg.Avg10m))
	metrics.GetOrRegisterGauge("load.runningprocesses", sr.hostRegistry).Update(int64(loadavg.RunningProcesses))
	metrics.GetOrRegisterGauge("load.totalprocesses", sr.hostRegistry).Update(int64(loadavg.TotalProcesses))

	stat, err := linux.ReadStat()
	if err != nil {
		glog.Errorf("could not read stat: %s", err)
		return
	}
	metrics.GetOrRegisterGauge("cpu.user", sr.hostRegistry).Update(int64(stat.Cpu.User()))
	metrics.GetOrRegisterGauge("cpu.nice", sr.hostRegistry).Update(int64(stat.Cpu.Nice()))
	metrics.GetOrRegisterGauge("cpu.system", sr.hostRegistry).Update(int64(stat.Cpu.System()))
	metrics.GetOrRegisterGauge("cpu.idle", sr.hostRegistry).Update(int64(stat.Cpu.Idle()))
	metrics.GetOrRegisterGauge("cpu.iowait", sr.hostRegistry).Update(int64(stat.Cpu.Iowait()))
	metrics.GetOrRegisterGauge("cpu.irq", sr.hostRegistry).Update(int64(stat.Cpu.Irq()))
	metrics.GetOrRegisterGauge("cpu.softirq", sr.hostRegistry).Update(int64(stat.Cpu.Softirq()))
	var steal int64
	if stat.Cpu.StealSupported() {
		steal = int64(stat.Cpu.Steal())
	}
	metrics.GetOrRegisterGauge("cpu.steal", sr.hostRegistry).Update(steal)

	meminfo, err := linux.ReadMeminfo()
	if err != nil {
		glog.Errorf("could not read meminfo: %s", err)
		return
	}
	metrics.GetOrRegisterGauge("memory.total", sr.hostRegistry).Update(int64(meminfo.MemTotal))
	metrics.GetOrRegisterGauge("memory.free", sr.hostRegistry).Update(int64(meminfo.MemFree))
	metrics.GetOrRegisterGauge("memory.buffers", sr.hostRegistry).Update(int64(meminfo.Buffers))
	metrics.GetOrRegisterGauge("memory.cached", sr.hostRegistry).Update(int64(meminfo.Cached))
	actualFree := int64(meminfo.MemFree) + int64(meminfo.Buffers) + int64(meminfo.Cached)
	metrics.GetOrRegisterGauge("memory.actualfree", sr.hostRegistry).Update(actualFree)
	metrics.GetOrRegisterGauge("memory.actualused", sr.hostRegistry).Update(int64(meminfo.MemTotal) - actualFree)
	metrics.GetOrRegisterGauge("swap.total", sr.hostRegistry).Update(int64(meminfo.SwapTotal))
	metrics.GetOrRegisterGauge("swap.free", sr.hostRegistry).Update(int64(meminfo.SwapFree))

	vmstat, err := linux.ReadVmstat()
	if err != nil {
		glog.Errorf("could not read vmstat: %s", err)
		return
	}
	metrics.GetOrRegisterGauge("vmstat.pgfault", sr.hostRegistry).Update(int64(vmstat.Pgfault))
	metrics.GetOrRegisterGauge("vmstat.pgmajfault", sr.hostRegistry).Update(int64(vmstat.Pgmajfault))
	metrics.GetOrRegisterGauge("vmstat.pgpgout", sr.hostRegistry).Update(int64(vmstat.Pgpgout) * 1024)
	metrics.GetOrRegisterGauge("vmstat.pgpgin", sr.hostRegistry).Update(int64(vmstat.Pgpgin) * 1024)
	metrics.GetOrRegisterGauge("vmstat.pswpout", sr.hostRegistry).Update(int64(vmstat.Pswpout) * 1024)
	metrics.GetOrRegisterGauge("vmstat.pswpin", sr.hostRegistry).Update(int64(vmstat.Pswpin) * 1024)

	if openFileDescriptorCount, err := GetOpenFileDescriptorCount(); err != nil {
		glog.Warningf("Couldn't get open file descriptor count", err)
	} else {
		metrics.GetOrRegisterGauge("Serviced.OpenFileDescriptors", sr.hostRegistry).Update(openFileDescriptorCount)
	}
}

// Updates the default registry.
func (sr *StatsReporter) updateStats() {
	// Stats for host.
	sr.updateHostStats()
	// Stats for the containers.
	var running []dao.RunningService
	running, err := zkservice.LoadRunningServicesByHost(sr.conn, sr.hostID)
	if err != nil {
		glog.Errorf("updateStats: zkservice.LoadRunningServicesByHost (conn: %+v hostID: %v) failed: %v", sr.conn, sr.hostID, err)
	}

	for _, rs := range running {
		if rs.DockerID != "" {
			containerRegistry := sr.getOrCreateContainerRegistry(rs.ServiceID, rs.InstanceID)
			if cpuacctStat, err := cgroup.ReadCpuacctStat(cgroup.GetCgroupDockerStatsFilePath(rs.DockerID, cgroup.Cpuacct)); err != nil {
				glog.V(4).Infof("Couldn't read CpuacctStat:", err)
			} else {
				metrics.GetOrRegisterGauge("cgroup.cpuacct.system", containerRegistry).Update(cpuacctStat.System)
				metrics.GetOrRegisterGauge("cgroup.cpuacct.user", containerRegistry).Update(cpuacctStat.User)
			}
			if memoryStat, err := cgroup.ReadMemoryStat(cgroup.GetCgroupDockerStatsFilePath(rs.DockerID, cgroup.Memory)); err != nil {
				glog.V(4).Infof("Couldn't read MemoryStat:", err)
			} else {
				metrics.GetOrRegisterGauge("cgroup.memory.pgmajfault", containerRegistry).Update(memoryStat.Pgfault)
				metrics.GetOrRegisterGauge("cgroup.memory.totalrss", containerRegistry).Update(memoryStat.TotalRss)
				metrics.GetOrRegisterGauge("cgroup.memory.cache", containerRegistry).Update(memoryStat.Cache)
			}
		} else {
			glog.V(4).Infof("Skipping stats update for %s (%s), no container ID exists yet", rs.Name, rs.ServiceID)
		}
	}
	// Clean out old container registries
	sr.removeStaleRegistries(&running)
}

// Fills out the metric consumer format.
func (sr *StatsReporter) gatherStats(t time.Time) []Sample {
	stats := []Sample{}
	// Handle the host metrics.
	reg, _ := sr.hostRegistry.(*metrics.StandardRegistry)
	reg.Each(func(name string, i interface{}) {
		if metric, ok := i.(metrics.Gauge); ok {
			tagmap := make(map[string]string)
			tagmap["controlplane_host_id"] = sr.hostID
			stats = append(stats, Sample{name, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
		}
		if metricf64, ok := i.(metrics.GaugeFloat64); ok {
			tagmap := make(map[string]string)
			tagmap["controlplane_host_id"] = sr.hostID
			stats = append(stats, Sample{name, strconv.FormatFloat(metricf64.Value(), 'f', -1, 32), t.Unix(), tagmap})
		}
	})
	// Handle each container's metrics.
	for key, registry := range sr.containerRegistries {
		reg, _ := registry.(*metrics.StandardRegistry)
		reg.Each(func(name string, i interface{}) {
			if metric, ok := i.(metrics.Gauge); ok {
				tagmap := make(map[string]string)
				tagmap["controlplane_service_id"] = key.serviceID
				tagmap["controlplane_instance_id"] = strconv.FormatInt(int64(key.instanceID), 10)
				tagmap["controlplane_host_id"] = sr.hostID
				stats = append(stats, Sample{name, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
			}
		})
	}
	return stats
}

// Send the list of stats to the TSDB.
func Post(destination string, stats []Sample) error {
	payload := map[string][]Sample{"metrics": stats}
	data, err := json.Marshal(payload)
	if err != nil {
		glog.Warningf("Couldn't marshal stats: ", err)
		return err
	}
	statsreq, err := http.NewRequest("POST", destination, bytes.NewBuffer(data))
	if err != nil {
		glog.Warningf("Couldn't create stats request: ", err)
		return err
	}
	statsreq.Header["User-Agent"] = []string{"Zenoss Metric Publisher"}
	statsreq.Header["Content-Type"] = []string{"application/json"}
	resp, reqerr := http.DefaultClient.Do(statsreq)
	if reqerr != nil {
		glog.Warningf("Couldn't post stats: ", reqerr)
		return reqerr
	}
	if !strings.Contains(resp.Status, "200 OK") {
		glog.Warningf("couldn't post stats: ", resp.Status)
		return nil
	}
	resp.Body.Close()
	return nil
}
