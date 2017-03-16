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

package service

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
)

// ServiceDetails is the minimal data necessary to show for a service
type ServiceDetails struct {
	ID                string
	Name              string
	Description       string
	PoolID            string
	ImageID           string
	ParentServiceID   string
	Parent            *ServiceDetails `json:",omitempty"` // optional, only needed when looking up service ancestry
	Instances         int
	InstanceLimits    domain.MinMax
	RAMCommitment     utils.EngNotation
	Startup           string
	HasChildren       bool
	DeploymentID      string
	DesiredState      int
	CurrentState      string
	Launch            string
	Tags              []string
	EmergencyShutdown bool
	UpdatedAt         time.Time
	CreatedAt         time.Time
	datastore.VersionedEntity
}

// Validation for Service ServiceDetails entity
func (d *ServiceDetails) ValidEntity() error {
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("ID", d.ID))
	violations.Add(validation.NotEmpty("Name", d.Name))
	violations.Add(validation.NotEmpty("PoolID", d.PoolID))
	violations.Add(d.InstanceLimits.Validate())
	if d.Instances != 0 {
		if d.InstanceLimits.Max != 0 {
			if d.Instances < d.InstanceLimits.Min || d.Instances > d.InstanceLimits.Max {
				violations.Add(fmt.Errorf("Instance count (%d) must be in InstanceLimits range [%d-%d]", d.Instances, d.InstanceLimits.Min, d.InstanceLimits.Max))
			}
		} else if d.Instances < d.InstanceLimits.Min {
			violations.Add(fmt.Errorf("Instance count (%d) must be greater than InstanceLimits min %d", d.Instances, d.InstanceLimits.Min))
		}
	}

	if len(violations.Errors) > 0 {
		return violations
	}

	return nil
}

// Equals returns true if two instances of ServiceDetails are the same
func (s *ServiceDetails) Equals(b *ServiceDetails) bool {
	if s.ID != b.ID {
		return false
	}
	if s.Name != b.Name {
		return false
	}
	if s.Startup != b.Startup {
		return false
	}
	if s.Description != b.Description {
		return false
	}
	if s.Instances != b.Instances {
		return false
	}
	if s.ImageID != b.ImageID {
		return false
	}
	if s.PoolID != b.PoolID {
		return false
	}
	if s.DesiredState != b.DesiredState {
		return false
	}
	if s.Launch != b.Launch {
		return false
	}
	if s.ParentServiceID != b.ParentServiceID {
		return false
	}
	if s.EmergencyShutdown != b.EmergencyShutdown {
		return false
	}
	return true
}
