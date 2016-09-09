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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
)

// ServiceDetails is the minimal data necessary to show for a service
type ServiceDetails struct {
	ID              string
	Name            string
	Description     string
	PoolID          string
	ParentServiceID string
	Instances       int
	InstanceLimits  domain.MinMax
	RAMCommitment   utils.EngNotation
	Startup         string
	datastore.VersionedEntity
}

// Validation for Service ServiceDetails entity
func (d *ServiceDetails) ValidEntity() error {
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("ServiceDetails.ID", d.ID))
	violations.Add(validation.NotEmpty("ServiceDetails.Name", d.Name))

	if len(violations.Errors) > 0 {
		return violations
	}

	return nil
}
