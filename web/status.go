// Copyright 2016 The Serviced Authors.
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

package web

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

var (
	cachedvalue     atomic.Value
	cachetimeout    = time.Duration(5) * time.Second
	emptyvalue      = []byte{}
	isvcsRootHealth = map[string]health.HealthStatus{
		"alive": health.HealthStatus{
			Status:    health.OK,
			StartedAt: time.Now(),
		},
	}
)

// SetServiceStatsCacheTimeout sets the time in seconds for stats on
// running instances to be cached for the UI.
func SetServiceStatsCacheTimeout(seconds int) {
	cachetimeout = time.Duration(seconds) * time.Second
}

// ConciseServiceStatus contains the information necessary to display the
// status of a service in the UI. It joins running instances with health checks
// with memory stats. It does not include fields unnecessary for status
// display, such as config files or monitoring profiles.
type ConciseServiceStatus struct {
	DockerID        string
	HostID          string
	ID              string
	InSync          bool
	InstanceID      int
	Instances       int
	Name            string
	ParentServiceID string
	PoolID          string
	ServiceID       string
	StartedAt       time.Time
	RAMMax          int64
	RAMLast         int64
	RAMAverage      int64
	RAMCommitment   utils.EngNotation
	HealthChecks    map[string]health.HealthStatus
}

type memorykey struct {
	ServiceID  string
	InstanceID int
}

// convertInstancesToMetric converts dao.RunningServices into the structure the
// metrics API wants
func convertInstancesToMetric(instances []dao.RunningService) []metrics.ServiceInstance {
	svcInstances := make([]metrics.ServiceInstance, len(instances))
	for i, inst := range instances {
		svcInstances[i] = metrics.ServiceInstance{
			inst.ServiceID,
			inst.InstanceID,
		}
	}
	return svcInstances
}

func getAllServiceStatuses(client *node.ControlClient) (statuses []*ConciseServiceStatus, err error) {
	// Get all running service instances
	glog.V(2).Info("Retrieving statuses, memory and health checks for running services")
	var instances []dao.RunningService
	if err := client.GetRunningServices(&empty, &instances); err != nil {
		return nil, err
	}
	if instances == nil {
		glog.V(3).Info("Nil service list returned")
		instances = []dao.RunningService{}
	}

	// Look up memory stats for the last hour
	// Memory stats for isvcs don't work the same way, so we don't append
	// those services to this list yet
	memoryStats := make(map[memorykey]metrics.MemoryUsageStats)
	if len(instances) > 0 {
		response := []metrics.MemoryUsageStats{}
		query := dao.MetricRequest{
			StartTime: time.Now().Add(-time.Hour * 24),
			Instances: convertInstancesToMetric(instances),
		}
		if err := client.GetInstanceMemoryStats(query, &response); err != nil {
			glog.Errorf("Unable to look up instance memory stats (%s)", err)
		}
		for _, mus := range response {
			instID, err := strconv.Atoi(mus.InstanceID)
			if err != nil {
				glog.Errorf("Could not convert instance id %s", mus.InstanceID)
				continue
			}
			memoryStats[memorykey{ServiceID: mus.ServiceID, InstanceID: instID}] = mus
		}
	}

	// Look up health check statuses
	healthChecks := make(map[memorykey]map[string]health.HealthStatus)
	if len(instances) > 0 {
		results := make(map[string]map[int]map[string]health.HealthStatus)
		if err := client.GetServicesHealth(0, &results); err != nil {
			glog.Errorf("Unable to look up health check results (%s)", err)
		}
		for svcid, insts := range results {
			for instid, checks := range insts {
				healthChecks[memorykey{ServiceID: svcid, InstanceID: instid}] = checks
			}
		}
	}

	// Add isvcs to the mix
	isvcInstances := getIRS()
	for _, isvc := range isvcInstances {
		var checks map[string]health.HealthStatus
		instances = append(instances, isvc)
		if isvc.ServiceID == "isvc-internalservices" {
			checks = isvcsRootHealth
		} else {
			results, err := isvcs.Mgr.GetHealthStatus(strings.TrimPrefix(isvc.ServiceID, "isvc-"))
			if err != nil {
				glog.Warningf("Error acquiring health status for %s (%s)", isvc.ServiceID, err)
				continue
			}
			checks = convertDomainHealthToNewHealth(results.HealthStatuses)
		}
		healthChecks[memorykey{ServiceID: isvc.ServiceID, InstanceID: isvc.InstanceID}] = checks
	}
	// Create the concise service statuses
	for _, instance := range instances {
		stat := &ConciseServiceStatus{
			DockerID:        instance.DockerID,
			HostID:          instance.HostID,
			ID:              instance.ID,
			InSync:          instance.InSync,
			InstanceID:      instance.InstanceID,
			Instances:       instance.Instances,
			Name:            instance.Name,
			ParentServiceID: instance.ParentServiceID,
			PoolID:          instance.PoolID,
			ServiceID:       instance.ServiceID,
			StartedAt:       instance.StartedAt,
			RAMCommitment:   instance.RAMCommitment,
		}
		key := memorykey{ServiceID: instance.ServiceID, InstanceID: instance.InstanceID}
		if mem, ok := memoryStats[key]; ok {
			stat.RAMMax = mem.Max
			stat.RAMLast = mem.Last
			stat.RAMAverage = mem.Average
		}
		if hc, ok := healthChecks[key]; ok {
			stat.HealthChecks = hc
		}
		statuses = append(statuses, stat)
	}
	return
}

func restGetConciseServiceStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	f := func() ([]byte, error) {
		statuses, err := getAllServiceStatuses(client)
		if err != nil {
			glog.V(2).Infof("Error retrieving service statuses: (%s)", err)
			return nil, err
		}
		bytes, err := json.Marshal(statuses)
		if err != nil {
			glog.V(2).Infof("Error serializing service statuses: (%s)", err)
			return nil, err
		}
		return bytes, nil
	}
	w.Header().Set("content-type", "application/json")
	bytes, err := getCached(f)
	if err != nil {
		glog.Errorf("Error retrieving service statuses: %s", err)
		restServerError(w, err)
	}
	w.Write(bytes)
}

func getCached(f func() ([]byte, error)) ([]byte, error) {
	var err error
	val := cachedvalue.Load()
	if val == nil || len(val.([]byte)) == 0 {
		val, err = f()
		if err != nil {
			return nil, err
		}
		cachedvalue.Store(val)
		go func() {
			time.Sleep(cachetimeout)
			cachedvalue.Store(emptyvalue)
		}()
	}
	return val.([]byte), nil
}

// ConvertDomainHealthToNewHealth changes the domain health check status into
// a new-style health status for consumption by the UI or other interface.
// Sort of deprecated, in that it will be rendered useless once
// domain.HealthCheckStatus is no longer returned by isvcs.Mgr.
func convertDomainHealthToNewHealth(sources []domain.HealthCheckStatus) map[string]health.HealthStatus {
	var status string
	result := make(map[string]health.HealthStatus)
	for _, hcs := range sources {
		switch hcs.Status {
		case "passed":
			status = health.OK
		case "failed":
			status = health.Failed
		default:
			status = health.Unknown
		}
		// Not going to bother populating Duration, since it doesn't have an
		// analogue in domain.HealthCheckStatus.
		result[hcs.Name] = health.HealthStatus{
			Status:    status,
			StartedAt: time.Unix(hcs.Timestamp, 0),
		}
	}
	return result
}
