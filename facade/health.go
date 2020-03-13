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
	"github.com/control-center/serviced/health"
	zkservice "github.com/control-center/serviced/zzk/service"
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServicesHealth"))
	shs, err := f.serviceStore.GetAllServiceHealth(ctx)
	if err != nil {
		glog.Errorf("Could not look up services: %s", err)
		return nil, err
	}
	stats := make(map[string]map[int]map[string]health.HealthStatus)

	for _, svc := range shs {
		if stats[svc.ID], err = f.getServiceHealth(ctx, svc); err != nil {
			return nil, err
		}
	}
	return stats, nil
}

// GetServiceHealth returns the status of all health instances.
func (f *Facade) GetServiceHealth(ctx datastore.Context, serviceID string) (map[int]map[string]health.HealthStatus, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceHealth"))
	sh, err := f.serviceStore.GetServiceHealth(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not look up service %s: %s", serviceID, err)
		return nil, err
	}
	return f.getServiceHealth(ctx, *sh)
}

func (f *Facade) getServiceHealth(ctx datastore.Context, sh service.ServiceHealth) (map[int]map[string]health.HealthStatus, error) {
	states, err := f.zzk.GetServiceStates(ctx, sh.PoolID, sh.ID)
	if err != nil {
		glog.Errorf("Could not get service states for service %s (%s): %s", sh.Name, sh.ID, err)
		return nil, err
	}
	stateMap := make(map[int]zkservice.State)
	for _, state := range states {
		stateMap[state.InstanceID] = state
	}
	status := make(map[int]map[string]health.HealthStatus)
	for i := 0; i < sh.Instances; i++ {
		stats := make(map[string]health.HealthStatus)
		for name, hc := range sh.HealthChecks {
			key := health.HealthStatusKey{
				ServiceID:       sh.ID,
				InstanceID:      i,
				HealthCheckName: name,
			}
			result, ok := f.hcache.Get(key)
			var uptime time.Duration
			s := stateMap[i]
			if s.Started.After(s.Terminated) {
				uptime = time.Since(s.Started)
			}

			if ok {
				stats[name] = result
			} else if uptime == 0 {
				stats[name] = hc.NotRunning()
			} else {
				stats[name] = hc.Unknown()
			}
		}
		status[i] = stats
	}
	return status, nil
}
