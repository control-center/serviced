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
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/node"
	"sync"
	"time"
)

type healthStatus struct {
	Status    string
	Timestamp int64
	Interval  float64
}

type messagePacket struct {
	Timestamp int64
	Statuses  map[string]map[string]*healthStatus
}

var healthStatuses = make(map[string]map[string]*healthStatus)
var exitChannel = make(chan bool)
var lock = &sync.Mutex{}

// RestGetHealthStatus writes a JSON response with the health status of all services that have health checks.
func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	packet := messagePacket{time.Now().UTC().Unix(), healthStatuses}
	w.WriteJson(&packet)
}

// RegisterHealthCheck updates the healthStatus and healthTime structures with a health check result.
func RegisterHealthCheck(serviceID string, name string, passed string, d dao.ControlPlane) {
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
		var service service.Service
		err := d.GetService(serviceID, &service)
		if err != nil {
			glog.Errorf("Unable to acquire services.")
			return
		}
		for iname, icheck := range service.HealthChecks {
			_, ok = serviceStatus[iname]
			if !ok {
				serviceStatus[name] = &healthStatus{"unknown", 0, icheck.Interval.Seconds()}
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
}
