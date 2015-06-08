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
	"time"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

// LocalSyncDatastore contains the primary datastore information from which to
// sync.
type LocalSyncDatastore interface {
	// GetResourcePools returns all the resource pools
	GetResourcePools() ([]pool.ResourcePool, error)
	// GetHosts returns hosts for a particular resource pool
	GetHosts(poolID string) ([]host.Host, error)
	// GetServices returns services for a particular resource pool
	GetServices(poolID string) ([]service.Service, error)
}

// LocalSyncInterface is the endpoint where data is synced
type LocalSyncInterface interface {
	// SyncResourcePools synchronizes resource pools
	SyncResourcePools(pools []pool.ResourcePool) error
	// SyncVirtualIPs synchronizes virtual ips for a pool
	SyncVirtualIPs(poolID string, virtualIPs []pool.VirtualIP) error
	// SyncHosts synchronizes hosts for a pool
	SyncHosts(poolID string, hosts []host.Host) error
	// SyncServices synchronizes services for a resource pool
	SyncServices(poolID string, svcs []service.Service) error
}

// LocalSync performs synchronization from the primary datastore to the
// coordinator
type LocalSync struct {
	ds    LocalSyncDatastore
	iface LocalSyncInterface
}

// Purge performs the synchronization for services, hosts, pools, and virtual
// ip.
// Implements utils.TTL
func (sync *LocalSync) Purge(age time.Duration) (time.Duration, error) {
	// synchronize resource pools
	pools, err := sync.ds.GetResourcePools()
	if err != nil {
		glog.Errorf("Could not get resource pools: %s", err)
		return 0, err
	} else if err := sync.iface.SyncResourcePools(pools); err != nil {
		glog.Errorf("Could not synchronize resource pools: %s", err)
		return 0, err
	}

	for _, pool := range pools {
		// synchronize virtual ips
		if err := sync.iface.SyncVirtualIPs(pool.ID, pool.VirtualIPs); err != nil {
			glog.Errorf("Could not synchronize virtual ips in pool %s: %s", pool.ID, err)
			return 0, err
		}

		// synchronize hosts
		if hosts, err := sync.ds.GetHosts(pool.ID); err != nil {
			glog.Errorf("Could not get hosts in pool %s: %s", pool.ID, err)
			return 0, err
		} else if err := sync.iface.SyncHosts(pool.ID, hosts); err != nil {
			glog.Errorf("Could not synchronize hosts in pool %s: %s", pool.ID, err)
			return 0, err
		}

		// synchronize services
		if svcs, err := sync.ds.GetServices(pool.ID); err != nil {
			glog.Errorf("Could not get services in pool %s: %s", pool.ID, err)
			return 0, err
		} else if err := sync.iface.SyncServices(pool.ID, svcs); err != nil {
			glog.Errorf("Could not synchronize services in pool %s: %s", pool.ID, err)
			return 0, err
		}
	}

	return age, nil
}