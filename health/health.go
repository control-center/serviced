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
	"errors"
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
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

var (
	healthMap                       *HealthStatusMap
	ErrHealthRegistryNotInitialized = errors.New("health registry not initialized")
)

func now() int64 {
	return time.Now().UTC().Unix()
}

type messagePacket struct {
	Timestamp int64
	Statuses  map[string]map[string]map[string]*domain.HealthCheckStatus
}

type RunningServices []dao.RunningService

func (r RunningServices) getService(serviceID, instanceID string) *dao.RunningService {
	for _, svc := range r {
		if svc.ServiceID == serviceID && strconv.Itoa(svc.InstanceID) == instanceID {
			return &svc
		}
	}
	return nil
}

func (r RunningServices) isService(serviceID string) bool {
	for _, svc := range r {
		if svc.ServiceID == serviceID {
			return true
		}
	}
	return false
}

// HealthStatusMap
type HealthStatusMap struct {
	sync.RWMutex
	cpDao        dao.ControlPlane
	facade       facade.FacadeInterface
	serviceLocks *utils.MutexMap
	statuses     map[string]map[string]map[string]*domain.HealthCheckStatus
}

func NewHealthStatuses(cpDao dao.ControlPlane, f facade.FacadeInterface) *HealthStatusMap {
	status := &HealthStatusMap{
		cpDao:        cpDao,
		facade:       f,
		statuses:     make(map[string]map[string]map[string]*domain.HealthCheckStatus),
		serviceLocks: utils.NewMutexMap(),
	}
	status.statuses["isvc-internalservices"] = map[string]map[string]*domain.HealthCheckStatus{
		"0": {
			"alive": &domain.HealthCheckStatus{
				Status:    "passed",
				Timestamp: now(),
				Interval:  3.156e9, // One century in seconds.
			},
		},
	}
	return status
}

// deleteServiceInstance deletes a single instance from the map
func (m *HealthStatusMap) deleteServiceInstance(serviceID, instanceID string) {
	m.serviceLocks.LockKey(serviceID)
	defer m.serviceLocks.UnlockKey(serviceID)
	serviceStatus, ok := m.statuses[serviceID]
	if !ok {
		serviceStatus = make(map[string]map[string]*domain.HealthCheckStatus)
		m.statuses[serviceID] = serviceStatus
	}
	delete(serviceStatus, instanceID)
}

// deleteServiceInstance deletes an entire service from the map
func (m *HealthStatusMap) deleteService(serviceID string) {
	m.serviceLocks.LockKey(serviceID)
	defer m.serviceLocks.UnlockKey(serviceID)
	delete(m.statuses, serviceID)
}

// getOrCreateServiceInstance looks up a service instance in the map. If it
// does not exist, it creates the necessary data structures populated with
// default values.
func (m *HealthStatusMap) getOrCreateServiceInstance(serviceID, instanceID string) (instanceStatus map[string]*domain.HealthCheckStatus) {
	var ok bool
	m.serviceLocks.LockKey(serviceID)
	defer m.serviceLocks.UnlockKey(serviceID)
	serviceStatus, ok := m.statuses[serviceID]
	if !ok {
		serviceStatus = make(map[string]map[string]*domain.HealthCheckStatus)
		m.statuses[serviceID] = serviceStatus
	}
	instanceStatus, ok = serviceStatus[instanceID]
	if !ok {
		instanceStatus = make(map[string]*domain.HealthCheckStatus)
		serviceStatus[instanceID] = instanceStatus
		healthChecks, err := m.facade.GetHealthChecksForService(datastore.Get(), serviceID)
		if err != nil {
			glog.Errorf("Unable to acquire health checks: %+v", err)
			return
		}
		for iname, icheck := range healthChecks {
			instanceStatus[iname] = &domain.HealthCheckStatus{
				iname,
				"unknown",
				0,
				icheck.Interval.Seconds(),
				now(),
				"",
			}
		}
	}
	return
}

// SetHealthStatus sets the status for a health check in the map.
func (m *HealthStatusMap) SetHealthStatus(serviceID, instanceID, healthCheckName, status string) {
	if healthCheckName == "__instance_shutdown" {
		m.deleteServiceInstance(serviceID, instanceID)
		return
	}
	instanceStatus := m.getOrCreateServiceInstance(serviceID, instanceID)
	thisStatus, ok := instanceStatus[healthCheckName]
	if !ok {
		glog.Warningf("ignoring %s health status %s, not found in service %s", status, healthCheckName, serviceID)
		return
	}
	newTime := now()
	// This is technically a race, but so unlikely a race that it's not worth
	// locking the entire service
	if newTime >= thisStatus.Timestamp {
		thisStatus.Status = status
		thisStatus.Timestamp = newTime
	}
}

// Returns Map of InstanceID -> HealthCheckName -> healthStatus for a given serviceID.
func (m *HealthStatusMap) GetHealthStatusesForService(serviceID string) map[string]map[string]domain.HealthCheckStatus {
	m.serviceLocks.RLockKey(serviceID)
	defer m.serviceLocks.RUnlockKey(serviceID)
	// Make a copy of m.statuses[serviceID] and store the HealthCheckStatus values instead of pointers
	result := make(map[string]map[string]domain.HealthCheckStatus, len(m.statuses[serviceID]))
	for instanceID, healthChecks := range m.statuses[serviceID] {
		result[instanceID] = make(map[string]domain.HealthCheckStatus, len(healthChecks))
		for hcName, hcStatus := range healthChecks {
			result[instanceID][hcName] = *hcStatus
		}
	}
	return result

}

func (m *HealthStatusMap) getRunningServices() (*RunningServices, error) {
	var empty interface{}
	rs := []dao.RunningService{}
	if err := m.cpDao.GetRunningServices(&empty, &rs); err != nil {
		return nil, err
	}
	r := RunningServices(rs)
	return &r, nil
}

// CleanupOnce removes services that are no longer running and updates service
// start time.
func (m *HealthStatusMap) CleanupOnce() {
	runningServices, err := m.getRunningServices()
	if err != nil {
		glog.Warningf("Error acquiring running services: %v", err)
		return
	}
	for serviceID, instances := range m.statuses {
		if strings.HasPrefix(serviceID, "isvc-") {
			continue
		}
		if !runningServices.isService(serviceID) {
			m.deleteService(serviceID)
			continue
		}
		for instanceID, healthChecks := range instances {
			svc := runningServices.getService(serviceID, instanceID)
			if svc == nil {
				m.deleteServiceInstance(serviceID, instanceID)
				continue
			}
			for _, check := range healthChecks {
				if check.StartedAt == 0 {
					check.StartedAt = svc.StartedAt.Unix()
				}
			}
		}
	}
}

// Cleanup runs CleanupOnce on a five-second cycle.
func (m *HealthStatusMap) Cleanup(shutdown <-chan interface{}) {
	for {
		select {
		case <-shutdown:
			return
		case <-time.After(time.Second * 5):
			m.CleanupOnce()
		}
	}
}

// RestGetHealthStatus writes a JSON response with the health status of all services that have health checks.
func (m *HealthStatusMap) RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	isvcNames := isvcs.Mgr.GetServiceNames()
	m.RLock()
	defer m.RUnlock()
	for _, name := range isvcNames {
		iname := "isvc-" + name
		status, err := isvcs.Mgr.GetHealthStatus(name)
		if err != nil {
			glog.Warningf("Error acquiring health status for %s: %s", name, err)
			continue
		}
		m.statuses[iname] = map[string]map[string]*domain.HealthCheckStatus{}
		m.statuses[iname]["0"] = map[string]*domain.HealthCheckStatus{}
		for _, status2 := range status.HealthStatuses {
			m.statuses[iname]["0"][status2.Name] = &status2
		}
	}
	packet := messagePacket{now(), m.statuses}
	w.WriteJson(&packet)
}

// Stores the dao.ControlPlane object created in daemon.go for use in this module.
func Initialize(d dao.ControlPlane, f facade.FacadeInterface, shutdown <-chan interface{}) {
	healthMap = NewHealthStatuses(d, f)
	go healthMap.Cleanup(shutdown)
}

func RegisterHealthCheck(serviceID, instanceID, name, passed string) error {
	if healthMap == nil {
		return ErrHealthRegistryNotInitialized
	}
	healthMap.SetHealthStatus(serviceID, instanceID, name, passed)
	return nil
}

func GetHealthStatusesForService(serviceID string) (map[string]map[string]domain.HealthCheckStatus, error) {
	if healthMap == nil {
		return nil, ErrHealthRegistryNotInitialized
	}
	return healthMap.GetHealthStatusesForService(serviceID), nil
}

func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	healthMap.RestGetHealthStatus(w, r, client)
}
