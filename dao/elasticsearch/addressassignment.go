// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/addressassignment"
)

// GetServiceAddressAssignments fills in all AddressAssignments for the specified serviced id.
func (this *ControlPlaneDao) GetServiceAddressAssignments(serviceID string, assignments *[]*addressassignment.AddressAssignment) error {
	return this.facade.GetServiceAddressAssignments(datastore.Get(), serviceID, assignments)
}

// RemoveAddressAssignemnt Removes an AddressAssignment by id
func (this *ControlPlaneDao) RemoveAddressAssignment(id string, _ *struct{}) error {
	return this.facade.RemoveAddressAssignment(datastore.Get(), id)
}

// AssignAddress Creates an AddressAssignment, verifies that an assignment for the service/endpoint does not already exist
// id param contains id of newly created assignment if successful
func (this *ControlPlaneDao) AssignAddress(assignment addressassignment.AddressAssignment, id *string) error {
	return this.facade.AssignAddress(datastore.Get(), assignment, id)
}
