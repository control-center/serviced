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
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
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
	ErrPoolExists    = errors.New("facade: resource pool exists")
	ErrPoolNotExists = errors.New("facade: resource pool does not exist")
	ErrIPExists      = errors.New("facade: ip exists in resource pool")
	ErrIPNotExists   = errors.New("facade: ip does not exist in resource pool")
)

//PoolIPs type for IP resources available in a ResourcePool
type PoolIPs struct {
	PoolID     string
	HostIPs    []host.HostIPResource
	VirtualIPs []pool.VirtualIP
}

// AddResourcePool adds a new resource pool
func (f *Facade) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
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
	} else if err = zkAPI(f).AddResourcePool(entity); err != nil {
		return err
	}

	if vips != nil && len(vips) > 0 {
		entity.VirtualIPs = vips
		return f.UpdateResourcePool(ctx, entity)
	}
	return nil
}

// UpdateResourcePool updates an existing resource pool
func (f *Facade) UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
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

	var newVIPs []pool.VirtualIP

	// Add the virtual ips that do not already exist
	for _, vip := range entity.VirtualIPs {
		if _, ok := currentVIPs[vip.IP]; ok {
			delete(currentVIPs, vip.IP)
		} else if err := f.addVirtualIP(ctx, &vip); err != nil {
			glog.Warningf("Could not add virtual ip %s: %s", vip.IP, err)
		} else {
			newVIPs = append(newVIPs, vip)
		}
	}

	// Delete the remaining virtual ips
	for _, vip := range currentVIPs {
		if err := f.removeVirtualIP(ctx, vip.PoolID, vip.IP); err != nil {
			glog.Warningf("Could not remove virtual ip %s: %s", vip.IP, err)
			newVIPs = append(newVIPs, vip)
		}
	}

	entity.VirtualIPs = newVIPs
	entity.UpdatedAt = time.Now()

	evtctx := newEventCtx()
	err = f.beforeEvent(beforePoolUpdate, evtctx, entity)
	defer f.afterEvent(afterPoolUpdate, evtctx, entity, err)
	if err != nil {
		return err
	} else if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return err
	} else if err = zkAPI(f).UpdateResourcePool(entity); err != nil {
		return err
	}

	return nil
}

// HasIP checks if a pool uses a particular IP address
func (f *Facade) HasIP(ctx datastore.Context, poolID string, ipAddr string) (bool, error) {
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

// AddVirtualIP adds a virtualIP to a pool
func (f *Facade) AddVirtualIP(ctx datastore.Context, vip pool.VirtualIP) error {
	entity, err := f.GetResourcePool(ctx, vip.PoolID)
	if err != nil {
		return err
	} else if entity == nil {
		return ErrPoolNotExists
	}

	if err := f.addVirtualIP(ctx, &vip); err != nil {
		return err
	}
	entity.VirtualIPs = append(entity.VirtualIPs, vip)
	entity.UpdatedAt = time.Now()

	evtctx := newEventCtx()
	err = f.beforeEvent(beforePoolUpdate, evtctx, entity)
	defer f.afterEvent(afterPoolUpdate, evtctx, entity, err)
	if err != nil {
		return err
	} else if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return err
	} else if err = zkAPI(f).UpdateResourcePool(entity); err != nil {
		return err
	}

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
	} else if validation.NotEmpty("Bind Interface", vip.BindInterface); err != nil {
		return err
	}

	if exists, err := f.HasIP(ctx, vip.PoolID, vip.IP); err != nil {
		return err
	} else if exists {
		return ErrIPExists
	}

	// add virtual ip to zookeeper
	return zkAPI(f).AddVirtualIP(vip)
}

// RemoveVirtualIP removes a virtual ip from a pool
func (f *Facade) RemoveVirtualIP(ctx datastore.Context, vip pool.VirtualIP) error {
	entity, err := f.GetResourcePool(ctx, vip.PoolID)
	if err != nil {
		return err
	} else if entity == nil {
		return ErrPoolNotExists
	}

	if err := f.removeVirtualIP(ctx, vip.PoolID, vip.IP); err != nil {
		return err
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
		return err
	}

	evtctx := newEventCtx()
	err = f.beforeEvent(beforePoolUpdate, evtctx, entity)
	defer f.afterEvent(afterPoolUpdate, evtctx, entity, err)
	if err != nil {
		return err
	} else if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return err
	} else if err = zkAPI(f).UpdateResourcePool(entity); err != nil {
		return err
	}

	// update address assignments
	for _, svc := range services {
		request := dao.AssignmentRequest{
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

func (f *Facade) removeVirtualIP(ctx datastore.Context, poolID, ipAddr string) error {
	if exists, err := f.poolStore.HasVirtualIP(ctx, poolID, ipAddr); err != nil {
		return err
	} else if !exists {
		return ErrIPNotExists
	}

	return zkAPI(f).RemoveVirtualIP(&pool.VirtualIP{PoolID: poolID, IP: ipAddr})
}

// RemoveResourcePool removes a ResourcePool
func (f *Facade) RemoveResourcePool(ctx datastore.Context, id string) error {
	glog.V(2).Infof("Facade.RemoveResourcePool: %s", id)

	if hosts, err := f.FindHostsInPool(ctx, id); err != nil {
		return fmt.Errorf("could not verify hosts in pool %s: %s", id, err)
	} else if count := len(hosts); count > 0 {
		return fmt.Errorf("cannot delete pool %s: found %d hosts", id, count)
	}

	if svcs, err := f.GetServicesByPool(ctx, id); err != nil {
		return fmt.Errorf("could not verify services in pool %s: %s", id, err)
	} else if count := len(svcs); count > 0 {
		return fmt.Errorf("cannot delete pool %s: found %d services", id, count)
	}

	return f.delete(ctx, f.poolStore, pool.Key(id), beforePoolDelete, afterPoolDelete)
}

//GetResourcePools Returns a list of all ResourcePools
func (f *Facade) GetResourcePools(ctx datastore.Context) ([]pool.ResourcePool, error) {
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

//GetResourcePoolsByRealm Returns a list of all ResourcePools by Realm
func (f *Facade) GetResourcePoolsByRealm(ctx datastore.Context, realm string) ([]pool.ResourcePool, error) {
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

// GetResourcePool returns  an ResourcePool ip id. nil if not found
func (f *Facade) GetResourcePool(ctx datastore.Context, id string) (*pool.ResourcePool, error) {
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

//CreateDefaultPool creates the default pool if it does not exists, it is idempotent
func (f *Facade) CreateDefaultPool(ctx datastore.Context, id string) error {
	entity, err := f.GetResourcePool(ctx, id)
	if err != nil {
		return fmt.Errorf("could not create default pool %s: %v", id, err)
	}
	if entity != nil {
		glog.V(4).Infof("'%s' resource pool already exists", id)
		return nil
	}

	glog.V(4).Infof("'%s' resource pool not found; creating...", id)
	entity = pool.New(id)
	entity.Realm = defaultRealm
	entity.Description = "Default Pool"
	if err := f.AddResourcePool(ctx, entity); err != nil {
		return err
	}
	return nil
}

func (f *Facade) calcPoolCapacity(ctx datastore.Context, pool *pool.ResourcePool) error {
	hosts, err := f.hostStore.FindHostsWithPoolID(ctx, pool.ID)

	if err != nil {
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

	return err
}

func (f *Facade) calcPoolCommitment(ctx datastore.Context, pool *pool.ResourcePool) error {
	services, err := f.serviceStore.GetServicesByPool(ctx, pool.ID)

	if err != nil {
		return err
	}

	memCommitment := uint64(0)
	for _, service := range services {
		memCommitment = memCommitment + service.RAMCommitment.Value
	}

	pool.MemoryCommitment = memCommitment

	return err
}

// GetPoolIPs gets all IPs available to a Pool
func (f *Facade) GetPoolIPs(ctx datastore.Context, poolID string) (*PoolIPs, error) {
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

	return &PoolIPs{PoolID: poolID, HostIPs: hostIPs, VirtualIPs: virtualIPs}, nil
}

var defaultRealm = "default"
