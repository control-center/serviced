// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scheduler

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/facade"
)

// Facade wrapper for synchronizers
type Facade struct {
	facade *facade.Facade
	ctx    datastore.Context
}

// GetResourcePools returns all of the resource pools.
// Implements LocalSyncInterface
func (f *Facade) GetResourcePools() ([]pool.ResourcePool, error) {
	return f.facade.GetResourcePools(f.ctx)
}

// GetHosts returns hosts for a particular poolID.
// Implements LocalSyncInterface
func (f *Facade) GetHosts(poolID string) ([]host.Host, error) {
	return f.facade.FindHostsInPool(f.ctx, poolID)
}

// GetServices returns services for a particular poolID.
// Implements LocalSyncInterface
func (f *Facade) GetServices(poolID string) ([]service.Service, error) {
	return f.facade.GetServicesByPool(f.ctx, poolID)
}