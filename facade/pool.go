// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/validation"

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
	exists, err := f.GetResourcePool(ctx, entity.ID)
	if err != nil {
		return err
	}
	if exists != nil {
		return fmt.Errorf("pool already exists: %s", entity.ID)
	}

	ec := newEventCtx()
	err = f.beforeEvent(beforePoolAdd, ec, entity)
	if err == nil {
		now := time.Now()
		entity.CreatedAt = now
		entity.UpdatedAt = now
		err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity)
	}
	f.afterEvent(afterPoolAdd, ec, entity, err)
	return err
}

// UpdateResourcePool updates a ResourcePool
func (f *Facade) UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
	glog.V(2).Infof("Facade.UpdateResourcePool: %+v", entity)
	ec := newEventCtx()
	err := f.beforeEvent(beforePoolUpdate, ec, entity)
	if err == nil {
		now := time.Now()
		entity.UpdatedAt = now
		err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity)
	}
	f.afterEvent(afterPoolUpdate, ec, entity, err)
	return err
}

// RemoveResourcePool removes a ResourcePool
func (f *Facade) RemoveResourcePool(ctx datastore.Context, id string) error {
	glog.V(2).Infof("Facade.RemoveResourcePool: %s", id)

	if hosts, err := f.FindHostsInPool(ctx, id); err != nil {
		return fmt.Errorf("error verifying no hosts in pool: %v", err)
	} else if len(hosts) > 0 {
		return errors.New("cannot delete resource pool with hosts")
	}

	return f.delete(ctx, f.poolStore, pool.Key(id), beforePoolDelete, afterPoolDelete)
}

//GetResourcePools Returns a list of all ResourcePools
func (f *Facade) GetResourcePools(ctx datastore.Context) ([]*pool.ResourcePool, error) {
	pools, err := f.poolStore.GetResourcePools(ctx)

	if err != nil {
		return nil, fmt.Errorf("Could not load pools: %v", err)
	}

	for _, pool := range pools {
		f.calcPoolCapacity(ctx, pool)
		f.calcPoolCommitment(ctx, pool)
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
func (f *Facade) CreateDefaultPool(ctx datastore.Context) error {
	entity, err := f.GetResourcePool(ctx, defaultPoolID)
	if err != nil {
		return fmt.Errorf("could not create default pool: %v", err)
	}
	if entity != nil {
		glog.V(4).Infof("'%s' resource pool already exists", defaultPoolID)
		return nil
	}

	glog.V(4).Infof("'%s' resource pool not found; creating...", defaultPoolID)
	entity = pool.New(defaultPoolID)
	return f.AddResourcePool(ctx, entity)
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

func virtualIPExists(proposedVirtualIP pool.VirtualIP, poolIPs *PoolIPs) bool {
	for _, virtualIP := range poolIPs.VirtualIPs {
		// the IP address is unique
		// TODO: Is an IP address unique to just a pool? Suppose virtual IP X. Can pools X and Y both contain X?
		// if so, we need to check PoolID as well
		if proposedVirtualIP.IP == virtualIP.IP {
			return true
		}
	}

	for _, staticIP := range poolIPs.HostIPs {
		if proposedVirtualIP.IP == staticIP.IPAddress {
			return true
		}
	}

	return false
}

func validIP(aString string) error {
	violations := validation.NewValidationError()
	violations.Add(validation.IsIP(aString))
	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
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

	if err := validIP(requestedVirtualIP.IP); err != nil {
		return err
	}
	if err := validIP(requestedVirtualIP.Netmask); err != nil {
		return err
	}

	poolIPs, err := f.GetPoolIPs(ctx, requestedVirtualIP.PoolID)
	if err != nil {
		glog.Errorf("GetPoolIps failed: %v", err)
		return err
	}

	ipAddressAlreadyExists := virtualIPExists(requestedVirtualIP, poolIPs)
	if ipAddressAlreadyExists {
		errMsg := fmt.Sprintf("Cannot add requested virtual IP address: %v as it already exists in pool: %v", requestedVirtualIP, requestedVirtualIP.PoolID)
		return errors.New(errMsg)
	}

	myPool.VirtualIPs = append(myPool.VirtualIPs, requestedVirtualIP)
	if err := f.UpdateResourcePool(ctx, myPool); err != nil {
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
			// delete the current VirtualIP
			myPool.VirtualIPs = append(myPool.VirtualIPs[:virtualIPIndex], myPool.VirtualIPs[virtualIPIndex+1:]...)
			if err := f.UpdateResourcePool(ctx, myPool); err != nil {
				return err
			}
			glog.Infof("Removed virtual IP: %v from pool: %v", virtualIP.IP, requestedVirtualIP.PoolID)
			return nil
		}
	}

	errMsg := fmt.Sprintf("Cannot remove requested virtual IP address: %v (does not exist)", requestedVirtualIP.IP)
	return errors.New(errMsg)
}

var defaultPoolID = "default"
