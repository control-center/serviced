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
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
)

// a lightweight Service object with enough data to support status polling even if frequent
type ServiceHealth struct {
	ID                string
	Name              string
	PoolID            string
	Instances         int
	DesiredState      int
	HealthChecks      map[string]health.HealthCheck
	EmergencyShutdown bool
	RAMCommitment     utils.EngNotation
	datastore.VersionedEntity
}

// Validation for Service ServiceDetails entity
func (sh *ServiceHealth) ValidEntity() error {
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("ID", sh.ID))
	violations.Add(validation.NotEmpty("Name", sh.Name))
	violations.Add(validation.NotEmpty("PoolID", sh.PoolID))

	if len(violations.Errors) > 0 {
		return violations
	}

	return nil
}

func BuildServiceHealth(svc Service) *ServiceHealth {
	sh := &ServiceHealth{
		ID:           svc.ID,
		Name:         svc.Name,
		PoolID:       svc.PoolID,
		Instances:    svc.Instances,
		DesiredState: svc.DesiredState,
		HealthChecks: make(map[string]health.HealthCheck),
	}

	for key, value := range svc.HealthChecks {
		sh.HealthChecks[key] = value
	}

	return sh
}
