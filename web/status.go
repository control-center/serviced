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
	"fmt"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/node"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

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
	HealthChecks    map[string]health.HealthStatus
}

func memoryKey(serviceID, instanceID string) string {
	return fmt.Sprintf("%s-%s", serviceID, instanceID)
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
	var instances []dao.RunningService
	if err := client.GetRunningServices(&empty, &instances); err != nil {
		return nil, err
	}
	if instances == nil {
		glog.V(3).Info("Nil service list returned")
		instances = []dao.RunningService{}
	}
	// Append internal services to the list
	instances = append(instances, getIRS()...)

	// Look up memory stats for the last hour
	memoryStats := make(map[string]metrics.MemoryUsageStats)
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
			memoryStats[memoryKey(mus.ServiceID, mus.InstanceID)] = mus
		}
	}

	// Look up health check statuses
	healthChecks := make(map[string]map[string]health.HealthStatus)
	if len(instances) > 0 {
		results := make(map[string]map[int]map[string]health.HealthStatus)
		if err := client.GetServicesHealth(0, &results); err != nil {
			glog.Errorf("Unable to look up health check results (%s)", err)
		}
		for svcid, insts := range results {
			for instid, checks := range insts {
				healthChecks[memoryKey(svcid, string(instid))] = checks
			}
		}
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
		}
		key := memoryKey(instance.ServiceID, string(instance.InstanceID))
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
	statuses, err := getAllServiceStatuses(client)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
	}
	w.WriteJson(statuses)
}
