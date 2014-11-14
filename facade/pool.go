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

//PoolIPs type for IP resources available in a ResourcePool
type PoolIPs struct {
	PoolID     string
	HostIPs    []host.HostIPResource
	VirtualIPs []pool.VirtualIP
}

// AddResourcePool add resource pool to index
func (f *Facade) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	glog.V(2).Infof("Facade.AddResourcePool: %+v", entity)
	if exists, err := f.GetResourcePool(ctx, entity.ID); err != nil {
		return err
	} else if exists != nil {
		return fmt.Errorf("pool already exists: %s", entity.ID)
	}

	var err error
	ec := newEventCtx()
	defer f.afterEvent(afterPoolAdd, ec, entity, err)

	if err = f.beforeEvent(beforePoolAdd, ec, entity); err != nil {
		return err
	}

	// TODO: Get rid of me when we have front-end functionality of pool realms
	if entity.Realm == "" {
		entity.Realm = defaultRealm
	}

	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now
	if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return err
	}
	err = zkAPI(f).AddResourcePool(entity)
	return err
}

func (f *Facade) hasVirtualIP(ctx datastore.Context, poolID string, ipAddr string) (bool, error) {
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

/*
Ensure that any new virtual IPs are valid.
Valid means that the strings representing the IP address and netmask are valid.
Valid means that the IP is NOT already in the pool (neither as a static IP nor a virtual IP)
*/
func (f *Facade) validateVirtualIPs(ctx datastore.Context, proposedPool *pool.ResourcePool) error {
	currentPool, err := f.GetResourcePool(ctx, proposedPool.ID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %v", proposedPool.ID)
		return err
	} else if currentPool == nil {
		msg := fmt.Sprintf("Pool ID: %v could not be found", proposedPool.ID)
		return errors.New(msg)
	}

	// are the virtual IPs the same?
	if !currentPool.VirtualIPsEqual(proposedPool) {
		currentVirtualIPs := make(map[string]pool.VirtualIP)
		for _, virtualIP := range currentPool.VirtualIPs {
			currentVirtualIPs[virtualIP.IP] = virtualIP
		}
		proposedVirtualIPs := make(map[string]pool.VirtualIP)
		for _, virtualIP := range proposedPool.VirtualIPs {
			if _, keyAlreadyExists := proposedVirtualIPs[virtualIP.IP]; keyAlreadyExists {
				return fmt.Errorf("duplicate virtual IP request: %v", virtualIP.IP)
			}
			proposedVirtualIPs[virtualIP.IP] = virtualIP
		}

		for key, proposedVirtualIP := range proposedVirtualIPs {
			// check to see if the proposedVirtualIP is a NEW one
			if _, keyExists := currentVirtualIPs[key]; !keyExists {
				// virtual IPs will be added, need to validate this virtual IP
				if err := validation.IsIP(proposedVirtualIP.IP); err != nil {
					return err
				}
				if err := validation.IsIP(proposedVirtualIP.Netmask); err != nil {
					return err
				}
				if err := validation.NotEmpty("Bind Interface", proposedVirtualIP.BindInterface); err != nil {
					return err
				}

				ipAddressAlreadyExists, err := f.hasVirtualIP(ctx, proposedPool.ID, proposedVirtualIP.IP)
				if err != nil {
					return err
				} else if ipAddressAlreadyExists {
					return fmt.Errorf("cannot add requested virtual IP address: %v as it already exists in pool: %v", proposedVirtualIP.IP, proposedVirtualIP.PoolID)
				}
			}
		}
	}
	return nil
}

// UpdateResourcePool updates a ResourcePool
func (f *Facade) UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	glog.V(2).Infof("Facade.UpdateResourcePool: %+v", entity)
	if err := f.validateVirtualIPs(ctx, entity); err != nil {
		return err
	}

	var err error
	ec := newEventCtx()
	defer f.afterEvent(afterPoolUpdate, ec, entity, err)

	if err = f.beforeEvent(beforePoolUpdate, ec, entity); err != nil {
		return err
	}

	now := time.Now()
	entity.UpdatedAt = now
	if err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity); err != nil {
		return err
	}
	err = zkAPI(f).UpdateResourcePool(entity)
	return err
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
		memCommitment = memCommitment + service.RAMCommitment
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

func (f *Facade) AddVirtualIP(ctx datastore.Context, requestedVirtualIP pool.VirtualIP) error {
	myPool, err := f.GetResourcePool(ctx, requestedVirtualIP.PoolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %v", requestedVirtualIP.PoolID)
		return err
	} else if myPool == nil {
		msg := fmt.Sprintf("Pool ID: %v could not be found", requestedVirtualIP.PoolID)
		return errors.New(msg)
	}

	myPool.VirtualIPs = append(myPool.VirtualIPs, requestedVirtualIP)
	if err := f.UpdateResourcePool(ctx, myPool); err != nil {
		return err
	}
	if err := zkAPI(f).AddVirtualIP(&requestedVirtualIP); err != nil {
		return err
	}
	return nil
}

func (f *Facade) RemoveVirtualIP(ctx datastore.Context, requestedVirtualIP pool.VirtualIP) error {
	myPool, err := f.GetResourcePool(ctx, requestedVirtualIP.PoolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %v", requestedVirtualIP.PoolID)
		return err
	} else if myPool == nil {
		msg := fmt.Sprintf("Pool ID: %v could not be found", requestedVirtualIP.PoolID)
		return errors.New(msg)
	}

	for virtualIPIndex, virtualIP := range myPool.VirtualIPs {
		if virtualIP.IP == requestedVirtualIP.IP {
			myPool.VirtualIPs = append(myPool.VirtualIPs[:virtualIPIndex], myPool.VirtualIPs[virtualIPIndex+1:]...)
			if err := f.UpdateResourcePool(ctx, myPool); err != nil {
				return err
			}
			if err := zkAPI(f).RemoveVirtualIP(&requestedVirtualIP); err != nil {
				return err
			}
			glog.Infof("Removed virtual IP: %v from pool: %v", virtualIP.IP, requestedVirtualIP.PoolID)
			return nil
		}
	}

	errMsg := fmt.Sprintf("Cannot remove requested virtual IP address: %v (does not exist)", requestedVirtualIP.IP)
	return errors.New(errMsg)
}

var defaultRealm = "default"
