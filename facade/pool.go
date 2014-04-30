// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
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
	return &entity, nil
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
	return f.poolStore.GetResourcePools(ctx)
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

var defaultPoolID = "default"

func VirtualIPExists(proposedVirtualIP pool.VirtualIP, poolsVirtualIPs []pool.VirtualIP) (bool, int) {
	for index, virtualIP := range poolsVirtualIPs {
		// TODO: What should determine the SAME virtual IP address? Perhaps just IP address?
		if proposedVirtualIP.PoolID == virtualIP.PoolID &&
			proposedVirtualIP.IP == virtualIP.IP &&
			proposedVirtualIP.Netmask == virtualIP.Netmask &&
			proposedVirtualIP.BindInterface == virtualIP.BindInterface {
			return true, index
		}
	}
	return false, -1
}

func ValidIP(aString string) error {
	violations := validation.NewValidationError()
	violations.Add(validation.IsIP(aString))
	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}

func (f *Facade) AddVirtualIP(ctx datastore.Context, requestedVirtualIP pool.VirtualIP) error {
	entity, err := f.GetResourcePool(ctx, requestedVirtualIP.PoolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %v", requestedVirtualIP.PoolID)
		return err
	}

	if err := ValidIP(requestedVirtualIP.IP); err != nil {
		return err
	}
	if err := ValidIP(requestedVirtualIP.Netmask); err != nil {
		return err
	}

	ipAddressAlreadyExists, position := VirtualIPExists(requestedVirtualIP, entity.VirtualIPs)
	if ipAddressAlreadyExists && position != -1 {
		errMsg := fmt.Sprintf("Cannot add requested virtual IP address: %v as it already exists in pool: %v", requestedVirtualIP, requestedVirtualIP.PoolID)
		return errors.New(errMsg)
	}

	// generate a UUID as a unique ID for the virtual IP
	virtualIPuuid, _ := dao.NewUuid()
	requestedVirtualIP.ID = virtualIPuuid

	entity.VirtualIPs = append(entity.VirtualIPs, requestedVirtualIP)
	if err := f.UpdateResourcePool(ctx, entity); err != nil {
		return err
	}

	return nil
}

func (f *Facade) RemoveVirtualIP(ctx datastore.Context, requestedVirtualIP pool.VirtualIP) error {
	entity, err := f.GetResourcePool(ctx, requestedVirtualIP.PoolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %v", requestedVirtualIP.PoolID)
		return err
	}

	virtualIPExists := false
	virtualIPToRemovePosition := 0
	for virtualIPIndex, virtualIP := range entity.VirtualIPs {
		if requestedVirtualIP.ID == virtualIP.ID {
			virtualIPExists = true
			virtualIPToRemovePosition = virtualIPIndex
		}
	}

	if !virtualIPExists {
		errMsg := fmt.Sprintf("Cannot remove requested virtual IP address: %v as it does not exist in pool: %v", requestedVirtualIP, requestedVirtualIP.PoolID)
		return errors.New(errMsg)
	}

	// delete the positionth element
	entity.VirtualIPs = append(entity.VirtualIPs[:virtualIPToRemovePosition], entity.VirtualIPs[virtualIPToRemovePosition+1:]...)
	if err := f.UpdateResourcePool(ctx, entity); err != nil {
		return err
	}

	return nil
}

// Retrieve pool IP address information (virtual and static)
func (f *Facade) RetrievePoolIPs(ctx datastore.Context, poolID string) ([]dao.IPInfo, error) {
	// get all the static IP addresses
	var IPsInfo []dao.IPInfo
	poolsIPInfo, err := f.GetPoolsIPInfo(ctx, poolID)
	if err != nil {
		fmt.Printf("GetPoolsIPInfo failed: %v", err)
		return nil, err
	}
	for _, ipInfo := range poolsIPInfo {
		IPsInfo = append(IPsInfo, dao.IPInfo{ipInfo.InterfaceName, ipInfo.IPAddress, "static"})
	}

	// get all the virtual IP addresses
	pool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %v", poolID)
		return nil, err
	}

	for _, virtualIP := range pool.VirtualIPs {
		// TODO: Fill in the interface name?
		IPsInfo = append(IPsInfo, dao.IPInfo{virtualIP.ID, virtualIP.IP, "virtual"})
	}

	return IPsInfo, nil
}

// Retrieve a pool's static IP addresses
func (f *Facade) GetPoolsIPInfo(ctx datastore.Context, poolID string) ([]host.HostIPResource, error) {
	var poolsIPInfo []host.HostIPResource
	// retrieve all the hosts that are in the requested pool
	poolHosts, err := f.FindHostsInPool(ctx, poolID)
	if err != nil {
		glog.Errorf("Could not get hosts for Pool %s: %v", poolID, err)
		return nil, err
	}

	for _, poolHost := range poolHosts {
		// retrieve the IPs of the hosts contained in the requested pool
		host, err := f.GetHost(ctx, poolHost.ID)
		if err != nil {
			glog.Errorf("Could not get host %s: %v", poolHost.ID, err)
			return nil, err
		}

		//aggregate all the IPResources from all the hosts in the requested pool
		for _, poolHostIPResource := range host.IPs {
			if poolHostIPResource.HostID != "" && poolHostIPResource.InterfaceName != "" && poolHostIPResource.IPAddress != "" {
				poolsIPInfo = append(poolsIPInfo, poolHostIPResource)
			}
		}
	}

	return poolsIPInfo, nil
}
