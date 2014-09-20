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

package health

import (
	"sync"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/node"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

type healthStatus struct {
	Status    string
	Timestamp int64
	Interval  float64
	StartedAt int64
}

type messagePacket struct {
	Timestamp int64
	Statuses  map[string]map[string]*healthStatus
}

var healthStatuses = make(map[string]map[string]*healthStatus)
var exitChannel = make(chan bool)
var lock = &sync.Mutex{}

func init() {
	foreverHealthy := &healthStatus{
		Status:    "passed",
		Timestamp: time.Now().UTC().Unix(),
		Interval:  3.156e9, // One century in seconds.
	}
	healthStatuses["isvc-internalservices"] = map[string]*healthStatus{"alive": foreverHealthy}
	healthStatuses["isvc-elasticsearch-logstash"] = map[string]*healthStatus{"alive": foreverHealthy}
	healthStatuses["isvc-elasticsearch-serviced"] = map[string]*healthStatus{"alive": foreverHealthy}
	healthStatuses["isvc-zookeeper"] = map[string]*healthStatus{"alive": foreverHealthy}
	healthStatuses["isvc-opentsdb"] = map[string]*healthStatus{"alive": foreverHealthy}
	healthStatuses["isvc-logstash"] = map[string]*healthStatus{"alive": foreverHealthy}
	healthStatuses["isvc-celery"] = map[string]*healthStatus{"alive": foreverHealthy}
	healthStatuses["isvc-dockerRegistry"] = map[string]*healthStatus{"alive": foreverHealthy}
}

// RestGetHealthStatus writes a JSON response with the health status of all services that have health checks.
func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	packet := messagePacket{time.Now().UTC().Unix(), healthStatuses}
	w.WriteJson(&packet)
}

// RegisterHealthCheck updates the healthStatus and healthTime structures with a health check result.
func RegisterHealthCheck(serviceID string, name string, passed string, d dao.ControlPlane, f *facade.Facade) {
	lock.Lock()
	defer lock.Unlock()

	serviceStatus, ok := healthStatuses[serviceID]
	if !ok {
		// healthStatuses[serviceID]
		serviceStatus = make(map[string]*healthStatus)
		healthStatuses[serviceID] = serviceStatus
	}
	thisStatus, ok := serviceStatus[name]
	if !ok {
		healthChecks, err := f.GetHealthChecksForService(datastore.Get(), serviceID)
		if err != nil {
			glog.Errorf("Unable to acquire health checks: %+v", err)
			return
		}
		for iname, icheck := range healthChecks {
			_, ok = serviceStatus[iname]
			if !ok {
				serviceStatus[name] = &healthStatus{"unknown", 0, icheck.Interval.Seconds(), time.Now().Unix()}
			}
		}
	}
	thisStatus, ok = serviceStatus[name]
	if !ok {
		glog.Warningf("ignoring health status, not found in service: %s %s %s", serviceID, name, passed)
		return
	}
	thisStatus.Status = passed
	thisStatus.Timestamp = time.Now().UTC().Unix()

	if thisStatus.StartedAt == 0 {
		var runningServices []dao.RunningService
		err := d.GetRunningServicesForService(serviceID, &runningServices)
		if err == nil && len(runningServices) > 0 {
			thisStatus.StartedAt = runningServices[0].StartedAt.Unix()
		}
	}
}
