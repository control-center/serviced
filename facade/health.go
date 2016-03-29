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

package facade

import (
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/health"
	"github.com/zenoss/glog"
)

// ReportHealthStatus writes the status of a health check to the cache.
func (f *Facade) ReportHealthStatus(key health.HealthStatusKey, value health.HealthStatus, expires time.Duration) {
	f.hcache.Set(key, value, expires)
}

// ReportInstanceDead removes all health checks of a particular instance from
// the cache.
func (f *Facade) ReportInstanceDead(serviceID string, instanceID int) {
	f.hcache.DeleteInstance(serviceID, instanceID)
}

// GetServicesHealth returns the status of all services health instances.
func (f *Facade) GetServicesHealth(ctx datastore.Context) (map[string]map[int]map[string]health.HealthStatus, error) {
	store := f.serviceStore
	svcs, err := store.GetServices(ctx)
	if err != nil {
		glog.Errorf("Could not look up services: %s", err)
		return nil, err
	}
	stats := make(map[string]map[int]map[string]health.HealthStatus)
	for _, svc := range svcs {
		if stats[svc.ID], err = f.getServiceHealth(svc); err != nil {
			return nil, err
		}
	}
	return stats, nil
}

// GetServiceHealth returns the status of all health instances.
func (f *Facade) GetServiceHealth(ctx datastore.Context, serviceID string) (map[int]map[string]health.HealthStatus, error) {
	store := f.serviceStore
	svc, err := store.Get(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not look up service %s: %s", serviceID, err)
		return nil, err
	}
	return f.getServiceHealth(*svc)
}

func (f *Facade) getServiceHealth(svc service.Service) (map[int]map[string]health.HealthStatus, error) {
	var states []servicestate.ServiceState
	if err := f.zzk.GetServiceStates(svc.PoolID, &states, svc.ID); err != nil {
		glog.Errorf("Could not get service states for service %s (%s): %s", svc.Name, svc.ID, err)
		return nil, err
	}
	stateMap := make(map[int]servicestate.ServiceState)
	for _, state := range stateMap {
		stateMap[state.InstanceID] = state
	}
	status := make(map[int]map[string]health.HealthStatus)
	for i := 0; i < svc.Instances; i++ {
		stats := make(map[string]health.HealthStatus)
		for name, hc := range svc.HealthChecks {
			key := health.HealthStatusKey{
				ServiceID:       svc.ID,
				InstanceID:      i,
				HealthCheckName: name,
			}
			result, ok := f.hcache.Get(key)
			if ok {
				stats[name] = result
			} else if stateMap[i].Uptime() == 0 {
				stats[name] = hc.NotRunning()
			} else {
				stats[name] = hc.Unknown()
			}
		}
		status[i] = stats
	}
	return status, nil
}
