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
	WaitSingleService(*service.Service, service.DesiredState, <-chan interface{}) error
	ScheduleServiceBatch(datastore.Context, []*service.Service, string, service.DesiredState) (int, error)
	UpdateService(ctx datastore.Context, svc service.Service) error
}
