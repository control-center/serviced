// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"errors"
	"fmt"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"time"
	"github.com/zenoss/serviced/datastore/context"
)

var (
	beforePoolUpdate beforeEvent = "BeforePoolUpdate"
	afterPoolUpdate  afterEvent  = "AfterPoolUpdate"
	beforePoolAdd    beforeEvent = "BeforePoolAdd"
	afterPoolAdd     afterEvent  = "AfterPoolAdd"
	beforePoolDelete beforeEvent = "BeforePoolDelete"
	afterPoolDelete  afterEvent  = "AfterPoolDelete"
)

type PoolIPs struct {
	PoolID  string
	HostIPs []host.HostIPResource
}

// GetPoolIPs gets all IPs available to a Pool
func (f *Facade) GetPoolIPs(ctx context.Context, poolID string) (*PoolIPs, error) {
	hosts, err := f.FindHostsInPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	hostIPs := make([]host.HostIPResource, 0)
	for _, h := range hosts {
		hostIPs = append(hostIPs, h.IPs...)
	}

	return &PoolIPs{PoolID: poolID, HostIPs: hostIPs}, nil
}

// AddResourcePool add resource pool to index
func (f *Facade) AddResourcePool(ctx context.Context, entity *pool.ResourcePool) error {
	glog.V(0).Infof("Facade.AddResourcePool: %+v", entity)
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
	defer f.afterEvent(afterPoolAdd, ec, entity, err)
	return err
}

// GetResourcePool returns  an ResourcePool ip id. nil if not found
func (f *Facade) GetResourcePool(ctx context.Context, id string) (*pool.ResourcePool, error) {
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

// UpdateHost information for a registered host
func (f *Facade) UpdateResourcePool(ctx context.Context, entity *pool.ResourcePool) error {
	glog.V(2).Infof("Facade.UpdateResourcePool: %+v", entity)
	ec := newEventCtx()
	err := f.beforeEvent(beforePoolUpdate, ec, entity)
	if err == nil {
		now := time.Now()
		entity.UpdatedAt = now
		err = f.poolStore.Put(ctx, pool.Key(entity.ID), entity)
	}
	defer f.afterEvent(afterPoolUpdate, ec, entity, err)
	return err
}

// RemoveResourcePool removes a ResourcePool
func (f *Facade) RemoveResourcePool(ctx context.Context, id string) error {
	glog.V(2).Infof("Facade.RemoveResourcePool: %s", id)

	if hosts, err := f.FindHostsInPool(ctx, id); err != nil {
		return fmt.Errorf("error verifying no hosts in pool: %v", err)
	} else if len(hosts) > 0 {
		return errors.New("cannot delete resource pool with hosts")
	}

	return f.delete(ctx, f.poolStore, pool.Key(id), beforePoolDelete, afterPoolDelete)
}

//GetResourcePools Returns a list of all ResourcePools
func (f *Facade) GetResourcePools(ctx context.Context) ([]*pool.ResourcePool, error) {
	return f.poolStore.GetResourcePools(ctx)
}

//CreateDefaultPool creates the default pool if it does not exists, it is idempotent
func (f *Facade) CreateDefaultPool(ctx context.Context) error {
	entity, err := f.GetResourcePool(ctx, DEFAULT_POOLID)
	if err != nil {
		return fmt.Errorf("could not create default pool: %v", err)
	}
	if entity != nil {
		glog.V(4).Infof("'%s' resource pool already exists", DEFAULT_POOLID)
		return nil
	}

	glog.V(0).Infof("'%s' resource pool not found; creating...", DEFAULT_POOLID)
	entity = pool.New(DEFAULT_POOLID)
	return f.AddResourcePool(ctx, entity)
}

var DEFAULT_POOLID string = "default"
