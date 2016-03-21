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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/metrics"
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
	InstanceID      uint
	Instances       uint
	Name            string
	ParentServiceID string
	PoolID          string
	ServiceID       string
	StartedAt       string
	RAMMax          int64
	RAMLast         int64
	RAMAverage      int64
	HealthChecks    []domain.HealthCheckStatus
}

func memoryKey(serviceID, instanceID string) string {
	return fmt.Sprintf("%s-%d", serviceID, instanceID)
}

func getAllMemoryStats(f *facade.Facade, instances ...dao.RunningService) (stats map[string]metrics.MemoryUsageStats) {
	// Request memory stats for the last hour
	startTime := time.Now().Add(-time.Hour * 24)
	// Query the stats
	response, err := f.GetInstanceMemoryStats(startTime, instances...)
	if err != nil {
		glog.Errorf("Unable to query memory stats")
		return
	}
	for _, mus := range response {
		stats[memoryKey(mus.ServiceID, mus.InstanceID)] = mus
	}
	return
}

func getAllServiceStatuses(f *facade.Facade, ctx datastore.Context) (statuses []*ConciseServiceStatus, err error) {
	services, err := f.GetRunningServices(ctx)
	if err != nil {
		return nil, err
	}
	if services == nil {
		glog.V(3).Info("Nil service list returned")
		services = []dao.RunningService{}
	}

	// Append internal services to the list
	services = append(services, getIRS()...)

	// Look up memory stats
	memoryStats := getAllMemoryStats(f)
	fmt.Println(memoryStats)

	// Look up health check statuses

	for _, svc := range services {
		stat := &ConciseServiceStatus{}
		statuses = append(statuses, stat)
		fmt.Println(svc)
	}
	return
}

func restGetConciseServiceStatus(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	statuses, err := getAllServiceStatuses(nil, nil)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
	}
	fmt.Println(statuses)
}
