// Copyright 2014 The Serviced Authors.
// Use of f source code is governed by a

package facade

import (
	"fmt"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

// GetServiceAddressAssignments fills in all AddressAssignments for the specified serviced id.
func (f *Facade) GetServiceAddressAssignments(ctx datastore.Context, serviceID string, assignments *[]addressassignment.AddressAssignment) error {
	store := addressassignment.NewStore()

	results, err := store.GetServiceAddressAssignments(ctx, serviceID)
	if err != nil {
		return err
	}
	*assignments = results
	return nil
}

// GetAddressAssignmentsByEndpoint returns the address assignment by serviceID and endpoint name
func (f *Facade) FindAddressAssignment(ctx datastore.Context, serviceID, endpointName string) (*addressassignment.AddressAssignment, error) {
	store := addressassignment.NewStore()
	return store.FindAddressAssignment(ctx, serviceID, endpointName)
}

// RemoveAddressAssignemnt Removes an AddressAssignment by id
func (f *Facade) RemoveAddressAssignment(ctx datastore.Context, id string) error {
	store := addressassignment.NewStore()
	key := addressassignment.Key(id)

	var assignment addressassignment.AddressAssignment
	if err := store.Get(ctx, key, &assignment); err != nil {
		return err
	}

	if err := store.Delete(ctx, key); err != nil {
		return err
	}

	var svc *service.Service
	var err error
	if svc, err = f.GetService(ctx, assignment.ServiceID); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetService service=%+v err=%s", assignment.ServiceID, err)
		return err
	}

	if err := f.updateService(ctx, svc); err != nil {
		glog.V(2).Infof("ControlPlaneDao.updateService service=%+v err=%s", assignment.ServiceID, err)
		return err
	}

	return nil
}

func (f *Facade) assign(ctx datastore.Context, assignment addressassignment.AddressAssignment) (string, error) {
	if err := assignment.ValidEntity(); err != nil {
		return "", err
	}

	// Do not add if it already exists
	if exists, err := f.FindAddressAssignment(ctx, assignment.ServiceID, assignment.EndpointName); err != nil {
		return "", err
	} else if exists != nil {
		return "", fmt.Errorf("found assignment for %s at %s", assignment.EndpointName, assignment.ServiceID)
	}

	var err error
	if assignment.ID, err = utils.NewUUID36(); err != nil {
		return "", err
	}

	store := addressassignment.NewStore()
	if err := store.Put(ctx, addressassignment.Key(assignment.ID), &assignment); err != nil {
		return "", err
	}

	return assignment.ID, nil
}
