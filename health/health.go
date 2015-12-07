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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/node"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

var (
	// Map of ServiceID -> InstanceID -> HealthCheckName -> healthStatus
	healthStatuses  = make(map[string]map[string]map[string]*domain.HealthCheckStatus)
	cpDao           dao.ControlPlane
	runningServices []dao.RunningService
	exitChannel     = make(chan bool)
	lock            = &sync.RWMutex{}
)

func init() {
	// Initialize the fake isvc application.
	healthStatuses["isvc-internalservices"] = map[string]map[string]*domain.HealthCheckStatus{
		"0": {
			"alive": &domain.HealthCheckStatus{
				Status:    "passed",
				Timestamp: time.Now().UTC().Unix(),
				Interval:  3.156e9, // One century in seconds.
			},
		},
	}
}

type messagePacket struct {
	Timestamp int64
	Statuses  map[string]map[string]map[string]*domain.HealthCheckStatus
}

type healthStatusMap struct {
	sync.RWMutex
	statuses map[string]map[string]map[string]*domain.HealthCheckStatus
}

func NewHealthStatuses() healthStatusMap {
	return healthStatusMap{
		statuses: make(map[string]map[string]map[string]*domain.HealthCheckStatus),
	}
}

// Returns Map of InstanceID -> HealthCheckName -> healthStatus for a given serviceID.
func (m *healthStatusMap) GetHealthStatusesForService(serviceID string) map[string]map[string]domain.HealthCheckStatus {
	m.RLock()
	defer m.RUnlock()
	//make a copy of healthStatuses[serviceID] and store the HealthCheckStatus values instead of pointers
	result := make(map[string]map[string]domain.HealthCheckStatus, len(healthStatuses[serviceID]))
	for instanceID, healthChecks := range m.statuses[serviceID] {
		result[instanceID] = make(map[string]domain.HealthCheckStatus, len(healthChecks))
		for hcName, hcStatus := range healthChecks {
			result[instanceID][hcName] = *hcStatus
		}
	}
	return result

}

func GetHealthStatusesForService(serviceID string) map[string]map[string]domain.HealthCheckStatus {
	lock.RLock()
	defer lock.RUnlock()

	//make a copy of healthStatuses[serviceID] and store the HealthCheckStatus values instead of pointers
	result := make(map[string]map[string]domain.HealthCheckStatus, len(healthStatuses[serviceID]))
	for instanceID, healthChecks := range healthStatuses[serviceID] {
		result[instanceID] = make(map[string]domain.HealthCheckStatus, len(healthChecks))
		for hcName, hcStatus := range healthChecks {
			result[instanceID][hcName] = *hcStatus
		}
	}
	return result
}

func getService(serviceID, instanceID string) *dao.RunningService {
	for _, svc := range runningServices {
		if svc.ServiceID == serviceID && strconv.Itoa(svc.InstanceID) == instanceID {
			return &svc
		}
	}
	return nil
}

func isService(serviceID string) bool {
	for _, svc := range runningServices {
		if svc.ServiceID == serviceID {
			return true
		}
	}
	return false
}

// CleanupOnce removes services that are no longer running and updates service
// start time.
func CleanupOnce() {
	var empty interface{}
	if cpDao == nil {
		return
	}
	err := cpDao.GetRunningServices(&empty, &runningServices)
	if err != nil {
		glog.Warningf("Error acquiring running services: %v", err)
		return
	}
	lock.Lock()
	for serviceID, instances := range healthStatuses {
		if strings.HasPrefix(serviceID, "isvc-") {
			continue
		}
		if !isService(serviceID) {
			delete(healthStatuses, serviceID)
			continue
		}
		for instanceID, healthChecks := range instances {
			svc := getService(serviceID, instanceID)
			if svc == nil {
				delete(instances, instanceID)
				continue
			}
			for _, check := range healthChecks {
				if check.StartedAt == 0 {
					check.StartedAt = svc.StartedAt.Unix()
				}
			}
		}
	}
	lock.Unlock()
}

// Cleanup runs CleanupOnce on a five-second cycle.
func Cleanup(shutdown <-chan interface{}) {
	for {
		select {
		case <-shutdown:
			return
		case <-time.After(time.Second * 5):
			CleanupOnce()
		}
	}
}

// Stores the dao.ControlPlane object created in daemon.go for use in this module.
func SetDao(d dao.ControlPlane) {
	cpDao = d
}

// RestGetHealthStatus writes a JSON response with the health status of all services that have health checks.
func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	lock.RLock()
	defer lock.RUnlock()
	isvcNames := isvcs.Mgr.GetServiceNames()
	for _, name := range isvcNames {
		iname := "isvc-" + name
		status, err := isvcs.Mgr.GetHealthStatus(name)
		if err != nil {
			glog.Warningf("Error acquiring health status for %s: %s", name, err)
			continue
		}
		healthStatuses[iname] = map[string]map[string]*domain.HealthCheckStatus{}
		healthStatuses[iname]["0"] = map[string]*domain.HealthCheckStatus{}
		for _, status2 := range status.HealthStatuses {
			healthStatuses[iname]["0"][status2.Name] = &status2
		}
	}
	packet := messagePacket{time.Now().UTC().Unix(), healthStatuses}
	w.WriteJson(&packet)
}

// RegisterHealthCheck updates the healthStatus and healthTime structures with a health check result.
func RegisterHealthCheck(serviceID string, instanceID string, name string, passed string, f facade.FacadeInterface) {
	lock.Lock()
	defer lock.Unlock()
	serviceStatus, ok := healthStatuses[serviceID]
	if !ok {
		serviceStatus = make(map[string]map[string]*domain.HealthCheckStatus)
		healthStatuses[serviceID] = serviceStatus
	}
	instanceStatus, ok := serviceStatus[instanceID]
	if !ok {
		instanceStatus = make(map[string]*domain.HealthCheckStatus)
		serviceStatus[instanceID] = instanceStatus
	}
	if name == "__instance_shutdown" {
		delete(serviceStatus, instanceID)
		return
	}
	thisStatus, ok := instanceStatus[name]
	if !ok {
		healthChecks, err := f.GetHealthChecksForService(datastore.Get(), serviceID)
		if err != nil {
			glog.Errorf("Unable to acquire health checks: %+v", err)
			return
		}
		for iname, icheck := range healthChecks {
			_, ok = instanceStatus[iname]
			if !ok {
				instanceStatus[iname] = &domain.HealthCheckStatus{name, "unknown", 0, icheck.Interval.Seconds(), time.Now().Unix(), ""}
			}
		}
	}
	thisStatus, ok = instanceStatus[name]
	if !ok {
		glog.Warningf("ignoring %s health status %s, not found in service %s", passed, name, serviceID)
		return
	}
	thisStatus.Status = passed
	thisStatus.Timestamp = time.Now().UTC().Unix()
}
