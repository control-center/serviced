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
	"strconv"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/isvcs"
	"github.com/zenoss/glog"
)

var DefaultExpiration = time.Minute

// DEPRECATED: use ReportHealthStatus or ReportInstanceDead instead
func (this *ControlPlaneDao) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	instanceID, err := strconv.Atoi(result.InstanceID)
	if err != nil {
		glog.Errorf("Could not parse instance ID %s: %s", result.InstanceID, err)
		return err
	}
	if result.Name == "__instance_shutdown" {
		this.facade.ReportInstanceDead(result.ServiceID, instanceID)
		return nil
	}
	key := health.HealthStatusKey{
		ServiceID:       result.ServiceID,
		InstanceID:      instanceID,
		HealthCheckName: result.Name,
	}
	startedAt, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", result.Timestamp)
	if err != nil {
		glog.Errorf("Could not parse timestamp %s: %s", result.Timestamp, err)
		return err
	}
	value := &health.HealthStatus{
		StartedAt: startedAt,
	}
	switch result.Passed {
	case "passed":
		value.Status = health.OK
	case "failed":
		value.Status = health.Failed
	default:
		value.Status = health.Unknown
	}
	this.facade.ReportHealthStatus(key, value, DefaultExpiration)
	return nil
}

func (this *ControlPlaneDao) ReportHealthStatus(req health.HealthStatusRequest, unused *int) error {
	this.facade.ReportHealthStatus(req.Key, req.Value, req.Expires)
	return nil
}

func (this *ControlPlaneDao) ReportInstanceDead(req dao.ServiceInstanceRequest, unused *int) error {
	this.facade.ReportInstanceDead(req.ServiceID, req.InstanceID)
	return nil
}

func (this *ControlPlaneDao) GetServicesHealth(_ dao.EntityRequest, res *map[string][]map[string]*health.HealthStatus) (err error) {
	*res, err = this.facade.GetServicesHealth(datastore.Get())
	return
}

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
