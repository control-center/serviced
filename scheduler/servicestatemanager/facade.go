// Copyright 2017 The Serviced Authors.
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

package servicestatemanager

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
)

// Facade provides access to functions for scheduling
type Facade interface {
	// WaitSingleService blocks until the service has reached the desired state, or the channel is closed
	WaitSingleService(*service.Service, service.DesiredState, <-chan interface{}) error
	// ScheduleServiceBatch changes the desired state of a set of services, and returns a list of IDs of services that could not be scheduled
	ScheduleServiceBatch(datastore.Context, []*CancellableService, string, service.DesiredState) ([]string, error)
	// UpdateService modifies a service
	UpdateService(ctx datastore.Context, svc service.Service) error
	// GetTenantIDs gets a list of all tenant IDs
	GetTenantIDs(ctx datastore.Context) ([]string, error)
	// GetServiceLite looks up the latest service object with all of the information necessary to schedule it
	GetServicesForScheduling(ctx datastore.Context, ids []string) []*service.Service
	// SetServicesCurrentState updates the service's current state in the service store
	SetServicesCurrentState(ctx datastore.Context, currentState service.ServiceCurrentState, serviceIDs ...string)
}
