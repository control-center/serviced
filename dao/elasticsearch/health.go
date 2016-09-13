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

package elasticsearch

import (

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/isvcs"
)

func (this *ControlPlaneDao) ServicedHealthCheck(IServiceNames []string, results *[]dao.IServiceHealthResult) error {
	if len(IServiceNames) == 0 {
		IServiceNames = isvcs.Mgr.GetServiceNames()
	}

	healthStatuses := make([]dao.IServiceHealthResult, len(IServiceNames))
	for i, name := range IServiceNames {
		status, err := isvcs.Mgr.GetHealthStatus(name)
		if err != nil {
			return err
		}

		healthStatuses[i] = status
	}

	*results = healthStatuses
	return nil
}

// GetServicesHealth returns health checks for all services.
func (this *ControlPlaneDao) GetServicesHealth(unused int, results *map[string]map[int]map[string]health.HealthStatus) (err error) {
	*results, err = this.facade.GetServicesHealth(datastore.Get())
	return
}

// ReportHealthStatus sends an update to the health check status cache.
func (this *ControlPlaneDao) ReportHealthStatus(req dao.HealthStatusRequest, unused *int) error {
	this.facade.ReportHealthStatus(req.Key, req.Value, req.Expires)
	return nil
}

// ReportInstanceDead removes stopped instances from the health check status cache.
func (this *ControlPlaneDao) ReportInstanceDead(req dao.ServiceInstanceRequest, unused *int) error {
	this.facade.ReportInstanceDead(req.ServiceID, req.InstanceID)
	return nil
}
