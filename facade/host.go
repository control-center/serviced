// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/host"

	"fmt"
	"time"
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

// AddHost register a host with serviced. Returns an error if host already exists
func (f *Facade) AddHost(ctx datastore.Context, entity *host.Host) error {
	glog.V(2).Infof("Facade.AddHost: %v", entity)
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

	ec := newEventCtx()
	err = f.beforeEvent(beforeHostAdd, ec, entity)
	if err == nil {
		now := time.Now()
		entity.CreatedAt = now
		entity.UpdatedAt = now
		err = f.hostStore.Put(ctx, host.HostKey(entity.ID), entity)
	}
	defer f.afterEvent(afterHostAdd, ec, entity, err)
	return err

}

// UpdateHost information for a registered host
func (f *Facade) UpdateHost(ctx datastore.Context, entity *host.Host) error {
	glog.V(2).Infof("Facade.UpdateHost: %+v", entity)
	//TODO: make sure pool exists
	ec := newEventCtx()
	err := f.beforeEvent(beforeHostAdd, ec, entity)
	if err == nil {
		now := time.Now()
		entity.UpdatedAt = now
		err = f.hostStore.Put(ctx, host.HostKey(entity.ID), entity)
	}
	defer f.afterEvent(afterHostAdd, ec, entity, err)
	return err
}

// RemoveHost removes a Host from serviced
func (f *Facade) RemoveHost(ctx datastore.Context, hostID string) error {
	glog.V(2).Infof("Facade.RemoveHost: %s", hostID)
	ec := newEventCtx()
	err := f.beforeEvent(beforeHostDelete, ec, hostID)
	if err == nil {
		err = f.hostStore.Delete(ctx, host.HostKey(hostID))
	}
	defer f.afterEvent(afterHostDelete, ec, hostID, err)
	return err
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
func (f *Facade) GetHosts(ctx datastore.Context) ([]*host.Host, error) {
	return f.hostStore.GetN(ctx, 10000)
}

// FindHostsInPool returns a list of all hosts with poolID
func (f *Facade) FindHostsInPool(ctx datastore.Context, poolID string) ([]*host.Host, error) {
	return f.hostStore.FindHostsWithPoolID(ctx, poolID)
}
