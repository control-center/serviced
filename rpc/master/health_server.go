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

package master

import (
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/isvcs"

	"time"
)

// HealthStatusRequest sends health status data to the health status cache.
type HealthStatusRequest struct {
	Key     health.HealthStatusKey
	Value   health.HealthStatus
	Expires time.Duration
}

type ReportDeadInstanceRequest struct {
	ServiceID  string
	InstanceID int
}

// GetISvcsHealth returns health status for a list of isvcs
func (s *Server) GetISvcsHealth(IServiceNames []string, results *[]isvcs.IServiceHealthResult) error {
	if len(IServiceNames) == 0 {
		IServiceNames = isvcs.Mgr.GetServiceNames()
	}

	healthStatuses := make([]isvcs.IServiceHealthResult, len(IServiceNames))
	for i, name := range IServiceNames {
		status, err := isvcs.Mgr.GetHealthStatus(name, isvcs.HEALTH_STATUS_INDEX_ALL)
		if err != nil {
			return err
		}

		healthStatuses[i] = status
	}

	*results = healthStatuses
	return nil
}

// GetServicesHealth returns health checks for all services.
func (s *Server) GetServicesHealth(unused struct{}, results *map[string]map[int]map[string]health.HealthStatus) error {
	if healthStatuses, err := s.f.GetServicesHealth(s.context()); err != nil {
		return err
	} else {
		*results = healthStatuses
	}
	return nil
}

// ReportHealthStatus sends an update to the health check status cache.
func (s *Server) ReportHealthStatus(request HealthStatusRequest, _ *struct{}) error {
	s.f.ReportHealthStatus(request.Key, request.Value, request.Expires)
	return nil
}

// ReportInstanceDead removes stopped instances from the health check status cache.
func (s *Server) ReportInstanceDead(request ReportDeadInstanceRequest, _ *struct{}) error {
	s.f.ReportInstanceDead(request.ServiceID, request.InstanceID)
	return nil
}
