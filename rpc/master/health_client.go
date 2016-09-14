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

// GetISvcsHealth returns health status for a list of isvcs
func (c *Client) GetISvcsHealth(IServiceNames []string) ([]isvcs.IServiceHealthResult, error) {
	results := []isvcs.IServiceHealthResult{}
	if err := c.call("GetISvcsHealth", IServiceNames, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetServicesHealth returns health checks for all services.
func (c *Client) GetServicesHealth() (map[string]map[int]map[string]health.HealthStatus, error) {
	results := make(map[string]map[int]map[string]health.HealthStatus)
	err := c.call("GetServicesHealth", empty, &results)
	return results, err
}

// ReportHealthStatus sends an update to the health check status cache.
func (c *Client) ReportHealthStatus(key health.HealthStatusKey, value health.HealthStatus, expires time.Duration) error {
	request := HealthStatusRequest{
		Key:     key,
		Value:   value,
		Expires: expires,
	}
	return c.call("ReportHealthStatus", request, empty)
}

// ReportInstanceDead removes stopped instances from the health check status cache.
func (c *Client) ReportInstanceDead(serviceID string, instanceID int) error {
	request := ReportDeadInstanceRequest{
		ServiceID:  serviceID,
		InstanceID: instanceID,
	}
	return c.call("ReportInstanceDead", request, empty)
}
