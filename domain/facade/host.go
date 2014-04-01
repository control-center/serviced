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
func (f *Facade) beforeHostRemove(session session, hostID string) error {
	return nil
}

// afterHostRemove called after removing a host, if there was an error removing the host err will be non nil. The same
// session instance is passed here and the corresponding beforeHostRemove
func (f *Facade) afterHostRemove(session session, hostID string, err error) {
	//TODO: remove AddressAssignments with this host
}

//---------------------------------------------------------------------------
// Host CRUD

// AddHost register a host with serviced. Returns an error if host already exists
func (f *Facade) AddHost(ctx datastore.Context, h *host.Host) error {
	glog.V(2).Infof("Facade.AddHost: %v", h)
	glog.Infof("Facade.AddHost: %v", h)
	exists, err := f.GetHost(ctx, h.ID)
	glog.Infof("Facade.AddHost: after gethost %v", exists)
	if err != nil {
		return err
	}
	if exists != nil {
		return fmt.Errorf("host with ID %s already exists", h.ID)
	}

	// validate Pool exists

	s := newSession()
	err = f.beforeHostAdd(s, h)
	now := time.Now()
	h.CreatedAt = now
	h.UpdatedAt = now
	if err == nil {
		err = f.hostStore.Put(ctx, host.HostKey(h.ID), h)
	}
	defer f.afterHostAdd(s, h, err)
	return err

}

// UpdateHost information for a registered host
func (f *Facade) UpdateHost(ctx datastore.Context, h *host.Host) error {
	glog.V(2).Infof("Facade.UpdateHost: %+v", h)
	//TODO: make sure pool exists
	s := newSession()
	err := f.beforeHostUpdate(s, h)
	now := time.Now()
	h.UpdatedAt = now
	if err == nil {
		err = f.hostStore.Put(ctx, host.HostKey(h.ID), h)
	}
	defer f.afterHostUpdate(s, h, err)
	return err
}

// RemoveHost removes  a Host from serviced
func (f *Facade) RemoveHost(ctx datastore.Context, hostID string) error {
	glog.V(2).Infof("Facade.RemoveHost: %s", hostID)
	s := newSession()
	err := f.beforeHostRemove(s, hostID)
	if err == nil {
		err = f.hostStore.Delete(ctx, host.HostKey(hostID))
	}
	defer f.afterHostRemove(s, hostID, err)
	return err
}

// GetHost gets a host by id. Returns nil if host not found
func (f *Facade) GetHost(ctx datastore.Context, hostID string) (*host.Host, error) {
	glog.V(2).Infof("Facade.GetHost: id=%s", hostID)

	glog.Infof("Facade.GetHost: id=%s", hostID)
	var value host.Host
	err := f.hostStore.Get(ctx, host.HostKey(hostID), &value)
	glog.Infof("Facade.GetHost: get error %v", err)
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
	return nil, nil
}
