// Copyright 2014 The Serviced Authors.
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
	"sync"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	zkpool "github.com/control-center/serviced/zzk/scheduler"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

func doRemoteSync(shutdown <-chan interface{}, f *facade.Facade, r *registry.EndpointRegistry) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		doPoolSync(shutdown, f)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		doEndpointSync(shutdown, r)
	}()

	wg.Wait()
}

func connectAll(shutdown <-chan interface{}) (remote client.Connection, local client.Connection) {
	var (
		remoteChan = zzk.Connect("/", zzk.GetRemoteConnection)
		localChan  = zzk.Connect("/", zzk.GetLocalConnection)
	)

	for remote == nil || local == nil {
		select {
		case remote = <-remoteChan:
			if remote == nil {
				remoteChan = zzk.Connect("/", zzk.GetRemoteConnection)
			}
		case local = <-localChan:
			if local == nil {
				localChan = zzk.Connect("/", zzk.GetLocalConnection)
			}
		case <-shutdown:
			return
		}
	}
	return remote, local
}

func doPoolSync(shutdown <-chan interface{}, f *facade.Facade) {
	rs := new(remoteSync).init(f)
	for {
		remote, local := connectAll(shutdown)
		select {
		case <-shutdown:
			return
		default:
			if remote == nil || local == nil {
				glog.Fatalf("Could not start up remote synchronizer")
				return
			}
		}
		// Set up pool listener
		poolSyncListener := zkpool.NewPoolSyncListener(remote, rs)

		// Add the host listener
		poolSyncListener.AddListener(func(conn client.Connection, poolID string) zzk.Listener {
			return zkservice.NewHostSyncListener(conn, rs, poolID)
		})

		// Add the service listener
		poolSyncListener.AddListener(func(conn client.Connection, poolID string) zzk.Listener {
			return zkservice.NewServiceSyncListener(conn, rs, poolID)
		})
		zzk.Listen(shutdown, make(chan error, 1), poolSyncListener)
	}
}

func doEndpointSync(shutdown <-chan interface{}, r *registry.EndpointRegistry) {
	for {
		remote, local := connectAll(shutdown)
		select {
		case <-shutdown:
			return
		default:
			if remote == nil || local == nil {
				glog.Fatalf("Could not start up remote synchronizer")
				return
			}
		}

		// Set up the endpoint registry listener
		erSyncListener := registry.NewEndpointRegistrySyncListener(remote, local, r)

		// Add the endpoint listener
		erSyncListener.AddListener(func(conn client.Connection, nodeID string) zzk.Listener {
			return registry.NewEndpointSyncListener(conn, local, r, nodeID)
		})
		zzk.Listen(shutdown, make(chan error, 1), erSyncListener)
	}
}

type remoteSync struct {
	f   *facade.Facade
	ctx datastore.Context
}

func (r *remoteSync) init(f *facade.Facade) *remoteSync {
	r = &remoteSync{
		f:   f,
		ctx: datastore.Get(),
	}
	return r
}

func (r *remoteSync) GetPathBasedConnection(path string) (client.Connection, error) {
	return zzk.GetRemoteConnection(path)
}

func (r *remoteSync) GetResourcePools() ([]*pool.ResourcePool, error) {
	return r.f.GetResourcePools(r.ctx)
}

func (r *remoteSync) AddOrUpdateResourcePool(pool *pool.ResourcePool) error {
	if p, err := r.f.GetResourcePool(r.ctx, pool.ID); err != nil {
		return err
	} else if p == nil {
		return r.f.AddResourcePool(r.ctx, pool)
	}

	return r.f.UpdateResourcePool(r.ctx, pool)
}

func (r *remoteSync) RemoveResourcePool(poolID string) error {
	return r.f.RemoveResourcePool(r.ctx, poolID)
}

func (r *remoteSync) GetServicesByPool(poolID string) ([]*service.Service, error) {
	return r.f.GetServicesByPool(r.ctx, poolID)
}

func (r *remoteSync) AddOrUpdateService(service *service.Service) error {
	if s, err := r.f.GetService(r.ctx, service.ID); err != nil {
		return err
	} else if s == nil {
		return r.f.AddService(r.ctx, *service)
	}

	return r.f.UpdateService(r.ctx, *service)
}

func (r *remoteSync) RemoveService(serviceID string) error {
	return r.f.RemoveService(r.ctx, serviceID)
}

func (r *remoteSync) GetHostsByPool(poolID string) ([]*host.Host, error) {
	return r.f.FindHostsInPool(r.ctx, poolID)
}

func (r *remoteSync) AddOrUpdateHost(host *host.Host) error {
	if h, err := r.f.GetHost(r.ctx, host.ID); err != nil {
		return err
	} else if h == nil {
		return r.f.AddHost(r.ctx, host)
	}

	return r.f.UpdateHost(r.ctx, host)
}

func (r *remoteSync) RemoveHost(hostID string) error {
	return r.f.RemoveHost(r.ctx, hostID)
}
