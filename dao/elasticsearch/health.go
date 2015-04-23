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
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/isvcs"
)

func (this *ControlPlaneDao) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	health.RegisterHealthCheck(result.ServiceID, result.InstanceID, result.Name, result.Passed, this.facade)
	return nil
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
