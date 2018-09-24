// Copyright 2014-2018 The Serviced Authors.
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

package service

import (
	"fmt"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/validation"
)

//ValidEntity validate that Service has all required fields
func (s *Service) ValidEntity() error {

	vErr := validation.NewValidationError()
	vErr.Add(validation.NotEmpty("ID", s.ID))
	vErr.Add(validation.NotEmpty("Name", s.Name))
	vErr.Add(validation.NotEmpty("PoolID", s.PoolID))

	vErr.Add(validation.StringIn(s.Launch, commons.AUTO, commons.MANUAL))
	vErr.Add(validation.IntIn(s.DesiredState, int(SVCRun), int(SVCStop), int(SVCPause)))

	// Validate the min/max/default instances
	vErr.Add(s.InstanceLimits.Validate())
	if s.Instances != 0 {
		if s.InstanceLimits.Max != 0 {
			if s.Instances < s.InstanceLimits.Min || s.Instances > s.InstanceLimits.Max {
				vErr.Add(fmt.Errorf("Instance count (%d) must be in InstanceLimits range [%d-%d]", s.Instances, s.InstanceLimits.Min, s.InstanceLimits.Max))
			}
		} else if s.Instances < s.InstanceLimits.Min {
			vErr.Add(fmt.Errorf("Instance count (%d) must be greater than InstanceLimits min %d", s.Instances, s.InstanceLimits.Min))
		}
	}

	// validate the monitoring profile
	vErr.Add(s.MonitoringProfile.ValidEntity())

	for _, ep := range s.Endpoints {
		vErr.Add(ep.ValidEntity())
	}

	for _, hc := range s.HealthChecks {
		vErr.Add(hc.ValidEntity())
	}

	if vErr.HasError() {
		return vErr
	}
	return nil
}

// ValidEntity ensures the enpoint has valid values, does not check vhosts, public ports and assignments
func (endpoint ServiceEndpoint) ValidEntity() error {
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("endpoint.Name", endpoint.Name))
	violations.Add(validation.StringIn(endpoint.Purpose, "export", "import", "import_all"))

	if endpoint.Protocol != "" {
		violations.Add(validation.StringIn(endpoint.Protocol, "tcp", "udp"))
		violations.Add(validation.ValidPort(int(endpoint.PortNumber)))
	}

	violations.Add(validation.NotEmpty("endpoint.Application", endpoint.Application))

	if violations.HasError() {
		return violations
	}
	return nil
}
