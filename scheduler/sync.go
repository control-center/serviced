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
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/facade"
	zkservice "github.com/control-center/serviced/zzk/service"
	zkvirtualips "github.com/control-center/serviced/zzk/virtualips"
)

// Facade wrapper for synchronizers
type Facade struct {
	facade *facade.Facade
	ctx    datastore.Context
}

// NewFacade instantiates a new facade object
func NewFacade(facade *facade.Facade, ctx datastore.Context) *Facade {
	return &Facade{facade, ctx}
}

// GetResourcePools returns all of the resource pools.
// Implements LocalSyncDatastore
func (f *Facade) GetResourcePools() ([]pool.ResourcePool, error) {
	return f.facade.GetResourcePools(f.ctx)
}

// GetHosts returns hosts for a particular poolID.
// Implements LocalSyncDatastore
func (f *Facade) GetHosts(poolID string) ([]host.Host, error) {
	return f.facade.FindHostsInPool(f.ctx, poolID)
}

// GetServices returns services for a particular poolID.
// Implements LocalSyncDatastore, RemoteSyncInterface
func (f *Facade) GetServices(poolID string) ([]service.Service, error) {
	return f.facade.GetServicesByPool(f.ctx, poolID)
}

// AddService creates a new service.
// Implements RemoteSyncInterface
func (f *Facade) AddService(svc *service.Service) error {
	return f.facade.AddService(f.ctx, *svc)
}

// UpdateService updates an existing service.
// Implements RemoteSyncInterface
func (f *Facade) UpdateService(svc *service.Service) error {
	return f.facade.UpdateService(f.ctx, *svc)
}

// RemoteService deletes an existing service.
// Implements RemoteSyncInterface
func (f *Facade) RemoveService(id string) error {
	return f.facade.RemoveService(f.ctx, id)
}

// CoordSync is the coordinator wrapper for synchronization.
type CoordSync struct {
	conn client.Connection
}

// NewCoordSync instantiates a new coordsync object
func NewCoordSync(conn client.Connection) *CoordSync {
	return &CoordSync{conn}
}

// SyncResourcePools synchronizes resource pools.
// Implements LocalSyncInterface
func (c *CoordSync) SyncResourcePools(pools []pool.ResourcePool) error {
	return zkservice.SyncPools(c.conn, pools)
}

// SyncVirtualIPs synchronizes virtual ips for a pool.
// Implements LocalSyncInterface
func (c *CoordSync) SyncVirtualIPs(poolID string, vips []pool.VirtualIP) error {
	return zkvirtualips.SyncVirtualIPs(c.conn, poolID, vips)
}

// SyncHosts synchronizes hosts for a pool.
// Implements LocalSyncInterface
func (c *CoordSync) SyncHosts(poolID string, hosts []host.Host) error {
	return zkservice.SyncHosts(c.conn, poolID, hosts)
}

// SyncServices synchronizes services for a resource pool.
// Implements LocalSyncInterface
func (c *CoordSync) SyncServices(poolID string, svcs []service.Service) error {
	return zkservice.SyncServices(c.conn, poolID, svcs)
}