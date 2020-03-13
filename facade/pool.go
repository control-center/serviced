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
	"github.com/control-center/serviced/audit"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/validation"
	"github.com/zenoss/glog"

	"errors"
	"fmt"
	"time"
)

const (
	beforePoolUpdate = beforeEvent("BeforePoolUpdate")
	afterPoolUpdate  = afterEvent("AfterPoolUpdate")
	beforePoolAdd    = beforeEvent("BeforePoolAdd")
	afterPoolAdd     = afterEvent("AfterPoolAdd")
	beforePoolDelete = beforeEvent("BeforePoolDelete")
	afterPoolDelete  = afterEvent("AfterPoolDelete")
)

var (
	// ErrPoolExists a pool this name already exists
	ErrPoolExists = errors.New("facade: resource pool exists")

	// ErrPoolNotExists the name pool not does exist
	ErrPoolNotExists = errors.New("facade: resource pool does not exist")

	// ErrIPExists the IP address already exists in the resource pool
	ErrIPExists = errors.New("facade: ip exists in resource pool")

	// ErrIPNotExists the IP address does not exist
	ErrIPNotExists = errors.New("facade: ip does not exist in resource pool")

	// ErrDefaultPool the default resource pool cannot be deleted
	ErrDefaultPool = errors.New("facade: cannot delete default resource pool")
)

// AddResourcePool adds a new resource pool
func (f *Facade) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AddResourcePool"))

	glog.Infof("Adding Resource Pool %s", entity.ID)

	alog := f.auditLogger.Message(ctx, "Adding Resource Pool").
		Action(audit.Add).Entity(entity)

	if err := f.DFSLock(ctx).LockWithTimeout("add resource pool", userLockTimeout); err != nil {
		glog.Warningf("Cannot add resource pool: %s", err)
		return alog.Error(err)
	}
	defer f.DFSLock(ctx).Unlock()

	return alog.Error(f.addResourcePool(ctx, entity))
}

func (f *Facade) addResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	if pool, err := f.GetResourcePool(ctx, entity.ID); err != nil {
		return err
	} else if pool != nil {
		return ErrPoolExists
	}

	vips := entity.VirtualIPs
	entity.VirtualIPs = []pool.VirtualIP{}
	// TODO: Get rid of me when we have front-end functionality of pool realms
	if entity.Realm == "" {
		entity.Realm = defaultRealm
	}
	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now

	// Add the pool
	evtctx := newEventCtx()
	err := f.beforeEvent(beforePoolAdd, evtctx, entity)
	defer f.afterEvent(afterPoolAdd, evtctx, entity, err)
	if err != nil {
		return err
	} else if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return err
	} else if err = f.zzk.UpdateResourcePool(entity); err != nil {
		return err
	}

	if vips != nil && len(vips) > 0 {
		entity.VirtualIPs = vips
		return f.updateResourcePool(ctx, entity)
	}

	f.poolCache.SetDirty()

	return nil
}

// UpdateResourcePool updates an existing resource pool
func (f *Facade) UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.UpdateResourcePool"))
	alog := f.auditLogger.Message(ctx, "Updating Resource Pool").
		Action(audit.Update).Entity(entity)
	if err := f.DFSLock(ctx).LockWithTimeout("update resource pool", userLockTimeout); err != nil {
		glog.Warningf("Cannot update resource pool: %s", err)
		return alog.Error(err)
	}
	defer f.DFSLock(ctx).Unlock()

	return alog.Error(f.updateResourcePool(ctx, entity))
}

func (f *Facade) updateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	current, err := f.GetResourcePool(ctx, entity.ID)
	if err != nil {
		return err
	} else if current == nil {
		return ErrPoolNotExists
	}

	currentVIPs := make(map[string]pool.VirtualIP)
	for _, vip := range current.VirtualIPs {
		currentVIPs[vip.IP] = vip
	}

	vips := []pool.VirtualIP{}

	// Add the virtual ips that do not already exist
	for _, vip := range entity.VirtualIPs {
		if _, ok := currentVIPs[vip.IP]; !ok {
			if err := f.addVirtualIP(ctx, &vip); err != nil {
				glog.Warningf("Could not add virtual ip %s: %s", vip.IP, err)
				continue
			}
		} else {
			delete(currentVIPs, vip.IP)
		}
		vips = append(vips, vip)
	}

	// Delete the remaining virtual ips
	for _, vip := range currentVIPs {
		if err := f.removeVirtualIP(ctx, vip.PoolID, vip.IP); err != nil {
			glog.Warningf("Could not remove virtual ip %s: %s", vip.IP, err)
			vips = append(vips, vip)
		}
	}

	entity.VirtualIPs = vips
	entity.UpdatedAt = time.Now()

	evtctx := newEventCtx()
	err = f.beforeEvent(beforePoolUpdate, evtctx, entity)
	defer f.afterEvent(afterPoolUpdate, evtctx, entity, err)
	if err != nil {
		return err
	} else if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return err
	} else if err = f.zzk.UpdateResourcePool(entity); err != nil {
		return err
	}

	// Update dfs permissions of the hosts that belong to the pool if
	// the pool's dfs permissions have changed
	if entity.HasDfsAccess() != current.HasDfsAccess() {
		dfsClients, err := f.FindHostsInPool(ctx, entity.ID)
		if err != nil {
			msg := "Error retrieving pool's hosts"
			plog.WithError(err).WithField("poolID", entity.ID).Warning(msg)
		} else {
			var clientsError error
			if entity.HasDfsAccess() { // Enable dfs access
				plog.WithField("poolID", entity.ID).Debug("DFS Access enabled for pool")
				clientsError = f.zzk.RegisterDfsClients(dfsClients...)
			} else { // Disable dfs access
				plog.WithField("poolID", entity.ID).Debug("DFS Access disabled for pool")
				clientsError = f.zzk.UnregisterDfsClients(dfsClients...)
			}
			if clientsError != nil {
				msg := "Could not update dfs clients in zk after changing pool permissions"
				plog.WithError(err).WithField("poolID", entity.ID).Warning(msg)
			}
		}
	}

	f.poolCache.SetDirty()

	return nil
}

// RestoreResourcePools restores a bulk of resource pools, usually from a backup.
func (f *Facade) RestoreResourcePools(ctx datastore.Context, pools []pool.ResourcePool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RestoreResourcePools"))
	// Do not DFSLock here, ControlPlaneDao does that
	var alog audit.Logger
	for _, pool := range pools {
		alog = f.auditLogger.Message(ctx, "Adding ResourcePool").Action(audit.Add).Entity(&pool)
		pool.DatabaseVersion = 0
		if err := f.addResourcePool(ctx, &pool); err != nil {
			if err == ErrPoolExists {
				if err := f.updateResourcePool(ctx, &pool); err != nil {
					glog.Errorf("Could not restore resource pool %s via update: %s", pool.ID, err)
					return alog.Error(err)
				}
			} else {
				glog.Errorf("Could not restore resource pool %s via add: %s", pool.ID, err)
				return alog.Error(err)
			}
		}
		alog.Succeeded()
	}
	return nil
}

// HasIP checks if a pool uses a particular IP address
func (f *Facade) HasIP(ctx datastore.Context, poolID string, ipAddr string) (bool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.HasIP"))
	if exists, err := f.poolStore.HasVirtualIP(ctx, poolID, ipAddr); err != nil {
		glog.Errorf("Could not look up ip %s for pool %s: %s", ipAddr, poolID, err)
		return false, err
	} else if exists {
		return true, nil
	}

	if host, err := f.GetHostByIP(ctx, ipAddr); err != nil {
		glog.Errorf("Could not look up static host by ip %s: %s", ipAddr, err)
		return false, err
	} else if host != nil && host.PoolID == poolID {
		return true, nil
	}

	return false, nil
}

// AddVirtualIP adds a virtual IP to a pool
func (f *Facade) AddVirtualIP(ctx datastore.Context, vip pool.VirtualIP) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AddVirtualIP"))
	entity, err := f.GetResourcePool(ctx, vip.PoolID)
	alog := f.auditLogger.Message(ctx, "Adding VirtualIP").
		Action(audit.Update).WithField("virtualip", vip.IP)
	if err != nil {
		return alog.Error(err)
	} else if entity == nil {
		return alog.Error(ErrPoolNotExists)
	}
	alog = alog.Entity(entity)
	if err := f.addVirtualIP(ctx, &vip); err != nil {
		return alog.Error(err)
	}
	entity.VirtualIPs = append(entity.VirtualIPs, vip)
	entity.UpdatedAt = time.Now()

	evtctx := newEventCtx()
	err = f.beforeEvent(beforePoolUpdate, evtctx, entity)
	defer f.afterEvent(afterPoolUpdate, evtctx, entity, err)
	if err != nil {
		return alog.Error(err)
	} else if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return alog.Error(err)
	} else if err = f.zzk.UpdateResourcePool(entity); err != nil {
		return alog.Error(err)
	}

	alog.Succeeded()
	return nil
}

func (f *Facade) addVirtualIP(ctx datastore.Context, vip *pool.VirtualIP) error {
	pool, err := f.GetResourcePool(ctx, vip.PoolID)
	if err != nil {
		return err
	} else if pool == nil {
		return ErrPoolNotExists
	}

	if err := validation.IsIP(vip.IP); err != nil {
		return err
	} else if err := validation.IsIP(vip.Netmask); err != nil {
		return err
	} else if err := validation.NotEmpty("Bind Interface", vip.BindInterface); err != nil {
		return err
	} else if err := validation.ValidVirtualIP(vip.BindInterface); err != nil {
		return err
	}

	if exists, err := f.HasIP(ctx, vip.PoolID, vip.IP); err != nil {
		return err
	} else if exists {
		return ErrIPExists
	}

	return nil
}

// RemoveVirtualIP removes a virtual IP from a pool
func (f *Facade) RemoveVirtualIP(ctx datastore.Context, vip pool.VirtualIP) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemoveVirtualIP"))
	alog := f.auditLogger.Message(ctx, "Removing VirtualIP").Action(audit.Update).
		WithField("virtualip", vip.IP)
	entity, err := f.GetResourcePool(ctx, vip.PoolID)
	if err != nil {
		return alog.Error(err)
	} else if entity == nil {
		return alog.Error(ErrPoolNotExists)
	}
	alog = alog.Entity(entity)
	if err := f.removeVirtualIP(ctx, vip.PoolID, vip.IP); err != nil {
		return alog.Error(err)
	}
	for i, currentVIP := range entity.VirtualIPs {
		if currentVIP.IP == vip.IP {
			entity.VirtualIPs = append(entity.VirtualIPs[:i], entity.VirtualIPs[i+1:]...)
			break
		}
	}

	// grab all services that are assigned to that virtual ip
	query := []string{fmt.Sprintf("Endpoints.AddressAssignment.IPAddr:%s", vip.IP)}
	services, err := f.GetTaggedServices(ctx, query)
	if err != nil {
		glog.Errorf("Failed to grab services with endpoints assigned to ip %s: %s", vip.IP, err)
		return alog.Error(err)
	}

	evtctx := newEventCtx()
	err = f.beforeEvent(beforePoolUpdate, evtctx, entity)
	defer f.afterEvent(afterPoolUpdate, evtctx, entity, err)
	if err != nil {
		return alog.Error(err)
	} else if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return alog.Error(err)
	} else if err = f.zzk.UpdateResourcePool(entity); err != nil {
		return alog.Error(err)
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
			alog.Error(err)
		}
	}
	alog.Succeeded()
	return nil
}

func (f *Facade) removeVirtualIP(ctx datastore.Context, poolID, ipAddr string) error {
	if exists, err := f.poolStore.HasVirtualIP(ctx, poolID, ipAddr); err != nil {
		return err
	} else if !exists {
		return ErrIPNotExists
	}

	return nil
}

// RemoveResourcePool removes a resource pool
func (f *Facade) RemoveResourcePool(ctx datastore.Context, id string) error {
	alog := f.auditLogger.
		Message(ctx, "Removing Resource Pool").
		Action(audit.Remove).ID(id).Type(pool.GetType())

	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemoveResourcePool"))
	glog.V(2).Infof("Facade.RemoveResourcePool: %s", id)
	if err := f.DFSLock(ctx).LockWithTimeout("remove resource pool", userLockTimeout); err != nil {
		glog.Warningf("Cannot remove resource pool: %s", err)
		return alog.Error(err)
	}
	defer f.DFSLock(ctx).Unlock()

	// CC-2024: do not delete the default resource pool
	if id == "default" {
		glog.Errorf("Cannot delete default resource pool")
		return alog.Error(ErrDefaultPool)
	}

	if hosts, err := f.FindHostsInPool(ctx, id); err != nil {
		return alog.Error(fmt.Errorf("could not verify hosts in pool %s: %s", id, err))
	} else if count := len(hosts); count > 0 {
		return alog.Error(fmt.Errorf("cannot delete pool %s: found %d hosts", id, count))
	}

	if svcs, err := f.GetServicesByPool(ctx, id); err != nil {
		return alog.Error(fmt.Errorf("could not verify services in pool %s: %s", id, err))
	} else if count := len(svcs); count > 0 {
		return alog.Error(fmt.Errorf("cannot delete pool %s: found %d services", id, count))
	}

	if err := f.delete(ctx, f.poolStore, pool.Key(id), beforePoolDelete, afterPoolDelete); err != nil {
		return alog.Error(err)
	}

	f.poolCache.SetDirty()

	return alog.Error(f.zzk.RemoveResourcePool(id))
}

// GetResourcePools returns a list of all resource pools
func (f *Facade) GetResourcePools(ctx datastore.Context) ([]pool.ResourcePool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetResourcePools"))
	pools, err := f.poolStore.GetResourcePools(ctx)

	if err != nil {
		return nil, fmt.Errorf("Could not load pools: %v", err)
	}

	for i := range pools {
		f.calcPoolCapacity(ctx, &pools[i])
		f.calcPoolCommitment(ctx, &pools[i])
	}

	return pools, err
}

// GetResourcePoolsByRealm returns a list of all resource pools by Realm
func (f *Facade) GetResourcePoolsByRealm(ctx datastore.Context, realm string) ([]pool.ResourcePool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetResourcePoolsByRealm"))
	pools, err := f.poolStore.GetResourcePoolsByRealm(ctx, realm)

	if err != nil {
		return nil, fmt.Errorf("Could not load pools: %v", err)
	}

	for i := range pools {
		f.calcPoolCapacity(ctx, &pools[i])
		f.calcPoolCommitment(ctx, &pools[i])
	}

	return pools, err
}

// GetResourcePool returns a resource pool, or nil if not found
func (f *Facade) GetResourcePool(ctx datastore.Context, id string) (*pool.ResourcePool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetResourcePool"))
	glog.V(2).Infof("Facade.GetResourcePool: id=%s", id)
	var entity pool.ResourcePool
	err := f.poolStore.Get(ctx, pool.Key(id), &entity)
	if datastore.IsErrNoSuchEntity(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	f.calcPoolCapacity(ctx, &entity)

	return &entity, nil
}

// CreateDefaultPool creates the default pool if it does not exist. It is idempotent.
func (f *Facade) CreateDefaultPool(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.CreateDefaultPool"))
	alog := f.auditLogger.Message(ctx, "Adding ResourcePool").
		ID(id).Type(pool.GetType()).Action(audit.Add)
	entity, err := f.GetResourcePool(ctx, id)
	if err != nil {
		alog.Error(err)
		return fmt.Errorf("could not create default pool %s: %v", id, err)
	}
	if entity != nil {
		glog.V(4).Infof("'%s' resource pool already exists", id)
		alog.Succeeded()
		return nil
	}

	glog.V(4).Infof("'%s' resource pool not found; creating...", id)
	entity = pool.New(id)
	entity.Realm = defaultRealm
	entity.Description = "Default Pool"
	entity.Permissions = pool.DFSAccess + pool.AdminAccess
	if err := f.AddResourcePool(ctx, entity); err != nil {
		return alog.Error(err)
	}
	alog.Succeeded()
	return nil
}

func (f *Facade) calcPoolCapacity(ctx datastore.Context, pool *pool.ResourcePool) error {
	hosts, err := f.hostStore.FindHostsWithPoolID(ctx, pool.ID)
	if err != nil {
		glog.Errorf("Unable to find hosts in %s: %v", pool.ID, err)
		return err
	}

	coreCapacity := 0
	memCapacity := uint64(0)
	for _, host := range hosts {
		coreCapacity = coreCapacity + host.Cores
		memCapacity = memCapacity + host.Memory
	}

	pool.CoreCapacity = coreCapacity
	pool.MemoryCapacity = memCapacity

	return nil
}

func (f *Facade) calcPoolCommitment(ctx datastore.Context, pool *pool.ResourcePool) error {
	services, err := f.serviceStore.GetServicesByPool(ctx, pool.ID)
	if err != nil {
		glog.Errorf("Unable to find services on %s: %v", pool.ID, err)
		return err
	}

	memCommitment := uint64(0)
	for _, service := range services {
		memCommitment = memCommitment + service.RAMCommitment.Value
	}

	pool.MemoryCommitment = memCommitment

	return nil
}

// GetPoolIPs gets all IPs available to a resource pool
func (f *Facade) GetPoolIPs(ctx datastore.Context, poolID string) (*pool.PoolIPs, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetPoolIPs"))
	glog.V(0).Infof("Facade.GetPoolIPs: %+v", poolID)
	hosts, err := f.FindHostsInPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	glog.V(0).Infof("Facade.GetPoolIPs: found hosts %v", hosts)

	// save off the static IP addresses
	hostIPs := make([]host.HostIPResource, 0)
	for _, h := range hosts {
		hostIPs = append(hostIPs, h.IPs...)
	}

	// save off the virtual IP addresses
	myPool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %v", poolID)
		return nil, err
	} else if myPool == nil {
		msg := fmt.Sprintf("Pool ID: %v could not be found", poolID)
		return nil, errors.New(msg)
	}
	virtualIPs := make([]pool.VirtualIP, 0)
	virtualIPs = append(virtualIPs, myPool.VirtualIPs...)

	return &pool.PoolIPs{PoolID: poolID, HostIPs: hostIPs, VirtualIPs: virtualIPs}, nil
}

var defaultRealm = "default"

// GetReadPools returns a list of simplified resource pools
func (f *Facade) GetReadPools(ctx datastore.Context) ([]pool.ReadPool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetReadPools"))

	var getPoolsFunc GetPoolsFunc = func() ([]pool.ReadPool, error) {
		pools, err := f.poolStore.GetResourcePools(ctx)

		if err != nil {
			return nil, fmt.Errorf("Could not load pools: %v", err)
		}

		readPools := []pool.ReadPool{}

		for i := range pools {
			f.calcPoolCapacity(ctx, &pools[i])
			f.calcPoolCommitment(ctx, &pools[i])

			readPools = append(readPools, pool.ReadPool{
				ID:                pools[i].ID,
				Description:       pools[i].Description,
				CreatedAt:         pools[i].CreatedAt,
				UpdatedAt:         pools[i].UpdatedAt,
				CoreCapacity:      pools[i].CoreCapacity,
				MemoryCapacity:    pools[i].MemoryCapacity,
				MemoryCommitment:  pools[i].MemoryCommitment,
				ConnectionTimeout: pools[i].ConnectionTimeout,
				Permissions:       pools[i].Permissions,
			})
		}
		return readPools, nil
	}

	return f.poolCache.GetPools(getPoolsFunc)
}
