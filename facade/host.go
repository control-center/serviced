// Copyright 2014 The Serviced Authors.
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

package facade

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

const (
	beforeHostUpdate = beforeEvent("BeforeHostUpdate")
	afterHostUpdate  = afterEvent("AfterHostUpdate")
	beforeHostAdd    = beforeEvent("BeforeHostAdd")
	afterHostAdd     = afterEvent("AfterHostAdd")
	beforeHostDelete = beforeEvent("BeforeHostDelete")
	afterHostDelete  = afterEvent("AfterHostDelete")
)

//---------------------------------------------------------------------------
// Host CRUD

// AddHost registers a host with serviced. Returns an error if host already
// exists or if the host's IP is a virtual IP.
func (f *Facade) AddHost(ctx datastore.Context, entity *host.Host) error {
	glog.V(2).Infof("Facade.AddHost: %v", entity)
	if err := f.DFSLock(ctx).LockWithTimeout("add host", userLockTimeout); err != nil {
		glog.Warningf("Cannot add host: %s", err)
		return err
	}
	defer f.DFSLock(ctx).Unlock()
	return f.addHost(ctx, entity)
}

func (f *Facade) addHost(ctx datastore.Context, entity *host.Host) error {
	exists, err := f.GetHost(ctx, entity.ID)
	if err != nil {
		return err
	}
	if exists != nil {
		return fmt.Errorf("host already exists: %s", entity.ID)
	}

	// validate Pool exists
	pool, err := f.GetResourcePool(ctx, entity.PoolID)
	if err != nil {
		return fmt.Errorf("error verifying pool exists: %v", err)
	}
	if pool == nil {
		return fmt.Errorf("error creating host, pool %s does not exists", entity.PoolID)
	}

	// verify that there are no virtual IPs with the given host IP(s)
	for _, ip := range entity.IPs {
		if exists, err := f.HasIP(ctx, pool.ID, ip.IPAddress); err != nil {
			return fmt.Errorf("error verifying ip %s exists: %v", ip.IPAddress, err)
		} else if exists {
			return fmt.Errorf("pool already has a virtual ip %s", ip.IPAddress)
		}
	}

	ec := newEventCtx()
	err = nil
	defer f.afterEvent(afterHostAdd, ec, entity, err)
	if err = f.beforeEvent(beforeHostAdd, ec, entity); err != nil {
		return err
	}

	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now

	if err = f.hostStore.Put(ctx, host.HostKey(entity.ID), entity); err != nil {
		return err
	}
	err = f.zzk.AddHost(entity)
	return err
}

// UpdateHost information for a registered host
func (f *Facade) UpdateHost(ctx datastore.Context, entity *host.Host) error {
	glog.V(2).Infof("Facade.UpdateHost: %+v", entity)
	if err := f.DFSLock(ctx).LockWithTimeout("update host", userLockTimeout); err != nil {
		glog.Warningf("Cannot update host: %s", err)
		return err
	}
	defer f.DFSLock(ctx).Unlock()

	// validate the host exists
	if host, err := f.GetHost(ctx, entity.ID); err != nil {
		return err
	} else if host == nil {
		return fmt.Errorf("host does not exist: %s", entity.ID)
	}

	// validate the pool exists
	if pool, err := f.GetResourcePool(ctx, entity.PoolID); err != nil {
		return err
	} else if pool == nil {
		return fmt.Errorf("pool does not exist: %s", entity.PoolID)
	}

	var err error
	ec := newEventCtx()
	defer f.afterEvent(afterHostAdd, ec, entity, err)

	if err = f.beforeEvent(beforeHostAdd, ec, entity); err != nil {
		return err
	}

	entity.UpdatedAt = time.Now()
	if err = f.hostStore.Put(ctx, host.HostKey(entity.ID), entity); err != nil {
		return err
	}

	err = f.zzk.UpdateHost(entity)
	return err
}

// RestoreHosts restores a list of hosts, typically from a backup
func (f *Facade) RestoreHosts(ctx datastore.Context, hosts []host.Host) error {
	// Do not DFSLock here, ControlPlaneDao does that
	var exists bool
	var err error

	for _, host := range hosts {
		host.DatabaseVersion = 0
		// check all the static ips on this host
		for _, ip := range host.IPs {
			if exists, err = f.HasIP(ctx, host.PoolID, ip.IPAddress); err != nil {
				glog.Errorf("Could no check ip %s in pool %s while restoring host %s: %s", ip.IPAddress, host.PoolID, ip.HostID, err)
				return err
			} else if exists {
				glog.Warningf("Could not restore host %s (%s): ip already exists", ip.HostID, ip.IPAddress)
				break
			}
		}
		if !exists {
			// check the primary ip on this host
			if exists, err := f.HasIP(ctx, host.PoolID, host.IPAddr); err != nil {
				glog.Errorf("Could not check ip %s in pool %s while restoring host %s: %s", host.IPAddr, host.PoolID, host.ID, err)
				return err
			} else if !exists {
				if err := f.addHost(ctx, &host); err != nil {
					glog.Errorf("Could not add host %s to pool %s: %s", host.ID, host.PoolID, err)
					return err
				}
				glog.Infof("Restored host %s (%s) to pool %s", host.ID, host.IPAddr, host.PoolID)
			} else {
				glog.Warningf("Could not restore host %s (%s): ip already exists", host.ID, host.IPAddr)
			}
		}
	}
	return nil
}

// RemoveHost removes a Host from serviced
func (f *Facade) RemoveHost(ctx datastore.Context, hostID string) (err error) {
	glog.V(2).Infof("Facade.RemoveHost: %s", hostID)
	if err := f.DFSLock(ctx).LockWithTimeout("remove host", userLockTimeout); err != nil {
		glog.Warningf("Cannot remove host: %s", err)
		return err
	}
	defer f.DFSLock(ctx).Unlock()

	//assert valid host
	var _host *host.Host
	if _host, err = f.GetHost(ctx, hostID); err != nil {
		return err
	} else if _host == nil {
		return fmt.Errorf("HostID %s does not exist", hostID)
	}

	ec := newEventCtx()
	defer f.afterEvent(afterHostDelete, ec, hostID, err)
	if err = f.beforeEvent(beforeHostDelete, ec, hostID); err != nil {
		return err
	}

	//remove host from zookeeper
	if err = f.zzk.RemoveHost(_host); err != nil {
		return err
	}

	//remove host from datastore
	if err = f.hostStore.Delete(ctx, host.HostKey(hostID)); err != nil {
		return err
	}

	//grab all services that are address assigned the host's IPs
	var services []service.Service
	for _, ip := range _host.IPs {
		query := []string{fmt.Sprintf("Endpoints.AddressAssignment.IPAddr:%s", ip.IPAddress)}
		svcs, err := f.GetTaggedServices(ctx, query)
		if err != nil {
			glog.Errorf("Failed to grab services with endpoints assigned to ip %s on host %s: %s", ip.IPAddress, _host.Name, err)
			return err
		}
		services = append(services, svcs...)
	}

	// update address assignments
	for _, svc := range services {
		request := addressassignment.AssignmentRequest{
			ServiceID:      svc.ID,
			IPAddress:      "",
			AutoAssignment: true,
		}
		if err = f.AssignIPs(ctx, request); err != nil {
			glog.Warningf("Failed assigning another ip to service %s: %s", svc.ID, err)
		}
	}

	return nil
}

// GetHost gets a host by id. Returns nil if host not found
func (f *Facade) GetHost(ctx datastore.Context, hostID string) (*host.Host, error) {
	glog.V(2).Infof("Facade.GetHost: id=%s", hostID)

	var value host.Host
	err := f.hostStore.Get(ctx, host.HostKey(hostID), &value)
	glog.V(4).Infof("Facade.GetHost: get error %v", err)
	if datastore.IsErrNoSuchEntity(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// GetHosts returns a list of all registered hosts
func (f *Facade) GetHosts(ctx datastore.Context) ([]host.Host, error) {
	return f.hostStore.GetN(ctx, 10000)
}

func (f *Facade) GetActiveHostIDs(ctx datastore.Context) ([]string, error) {
	hostids := []string{}
	pools, err := f.GetResourcePools(ctx)
	if err != nil {
		glog.Errorf("Could not get resource pools: %v", err)
		return nil, err
	}
	for _, p := range pools {
		var active []string
		if err := f.zzk.GetActiveHosts(p.ID, &active); err != nil {
			glog.Errorf("Could not get active host ids for pool: %v", err)
			return nil, err
		}
		hostids = append(hostids, active...)
	}
	return hostids, nil
}

// FindHostsInPool returns a list of all hosts with poolID
func (f *Facade) FindHostsInPool(ctx datastore.Context, poolID string) ([]host.Host, error) {
	return f.hostStore.FindHostsWithPoolID(ctx, poolID)
}

func (f *Facade) GetHostByIP(ctx datastore.Context, hostIP string) (*host.Host, error) {
	return f.hostStore.GetHostByIP(ctx, hostIP)
}
