// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"fmt"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/utils"
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

func (f *Facade) GetServiceAddressAssignmentsByPort(ctx datastore.Context, port uint16) ([]addressassignment.AddressAssignment, error) {
	store := addressassignment.NewStore()
	return store.GetServiceAddressAssignmentsByPort(ctx, port)
}

// GetAddressAssignmentsByEndpoint returns the address assignment by serviceID and endpoint name
func (f *Facade) FindAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpointName string) (*addressassignment.AddressAssignment, error) {
	store := addressassignment.NewStore()
	return store.FindAssignmentByServiceEndpoint(ctx, serviceID, endpointName)
}

func (f *Facade) FindAssignmentByHostPort(ctx datastore.Context, ipAddr string, port uint16) (*addressassignment.AddressAssignment, error) {
	store := addressassignment.NewStore()
	return store.FindAssignmentByHostPort(ctx, ipAddr, port)
}

// RemoveAddressAssignment Removes an AddressAssignment by id
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

	return nil
}

func (f *Facade) assign(ctx datastore.Context, assignment addressassignment.AddressAssignment) (string, error) {
	if err := assignment.ValidEntity(); err != nil {
		return "", err
	}

	// Do not add if it already exists
	if exists, err := f.FindAssignmentByServiceEndpoint(ctx, assignment.ServiceID, assignment.EndpointName); err != nil {
		return "", err
	} else if exists != nil {
		return "", fmt.Errorf("found assignment for %s at %s", assignment.EndpointName, assignment.ServiceID)
	}

	// Do not add if already assigned
	if exists, err := f.FindAssignmentByHostPort(ctx, assignment.IPAddr, assignment.Port); err != nil {
		return "", err
	} else if exists != nil {
		return "", fmt.Errorf("found assignment for port %d at %s", assignment.Port, assignment.IPAddr)
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
