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

type session map[string]interface{}

func newSession() session {
	return make(map[string]interface{})
}

// beforeHostUpdate called before updating a host. The same session instance is passed here and the corresponding
// afterHostUpdate. If an error is returned host will not be updated.
func (f *Facade) beforeHostUpdate(session session, host *host.Host) error {
	return nil
}

// afterHostUpdate called after updating a host, if there was an error updating the host err will be non nil. The same
// session instance is passed here and the corresponding beforeHostUpdate
func (f *Facade) afterHostUpdate(session session, host *host.Host, err error) {

}

// beforeHostAdd called before adding a host. The same session instance is passed here and the corresponding
// afterHostAdd. If an error is returned host will not be added.
func (f *Facade) beforeHostAdd(session session, host *host.Host) error {
	return nil
}

// afterHostUpdate called after adding a host, if there was an error adding the host err will be non nil. The same
// session instance is passed here and the corresponding beforeHostAdd
func (f *Facade) afterHostAdd(session session, host *host.Host, err error) {

}

// beforeHostRemove called before removing a host. The same session instance is passed here and the corresponding
// afterHostRemove. If an error is returned host will not be removed.
func (f *Facade) beforeHostRemove(session session, hostId string) error {
	return nil
}

// afterHostRemove called after removing a host, if there was an error removing the host err will be non nil. The same
// session instance is passed here and the corresponding beforeHostRemove
func (f *Facade) afterHostRemove(session session, hostId string, err error) {
	//TODO: remove AddressAssignments with this host

}

//---------------------------------------------------------------------------
// Host CRUD

// Register a host with serviced. Returns an error if host already exists
func (f *Facade) AddHost(ctx datastore.Context, host *host.Host) error {
	glog.V(2).Infof("Facade.AddHost: %+v", host)
	exists, err := f.GetHost(ctx, host.Id)
	if err != nil {
		return err
	}
	if exists != nil {
		return fmt.Errorf("Host with ID %s already exists", host.Id)
	}

	// validate Pool exists

	s := newSession()
	err = f.beforeHostAdd(s, host)
	now := time.Now()
	host.CreatedAt = now
	host.UpdatedAt = now
	if err == nil {
		err = f.hostStore.Put(ctx, host)
	}
	defer f.afterHostAdd(s, host, err)
	return err

}

// Update Host information for a registered host
func (f *Facade) UpdateHost(ctx datastore.Context, host *host.Host) error {
	glog.V(2).Infof("Facade.UpdateHost: %+v", host)
	//TODO: make sure pool exists
	s := newSession()
	err := f.beforeHostUpdate(s, host)
	now := time.Now()
	host.UpdatedAt = now
	if err == nil {
		err = f.hostStore.Put(ctx, host)
	}
	defer f.afterHostUpdate(s, host, err)
	return err
}

// Remove a Host from serviced
func (f *Facade) RemoveHost(ctx datastore.Context, hostId string) error {
	glog.V(2).Infof("Facade.RemoveHost: %s", hostId)
	s := newSession()
	err := f.beforeHostRemove(s, hostId)
	if err == nil {
		err = f.hostStore.Delete(ctx, hostId)
	}
	defer f.afterHostRemove(s, hostId, err)
	return err
}

// Get Host by id
func (f *Facade) GetHost(ctx datastore.Context, hostId string) (*host.Host, error) {
	glog.V(2).Infof("Facade.GetHost: id=%s", hostId)
	host, err := f.hostStore.Get(ctx, hostId)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		return nil, err
	}
	return host, nil
}

// GetHosts returns a list of all registered hosts
func (f *Facade) GetHosts(ctx datastore.Context) ([]host.Host, error) {
	return nil, nil
}
