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

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
)

// GetServiceAddressAssignmentDetails provides details about address assignments
// for the specified service id as is presented to the front-end.
func (f *Facade) GetServiceAddressAssignmentDetails(ctx datastore.Context, serviceID string, children bool) ([]service.IPAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceAddressAssignmentDetails"))
	// get the service
	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	return f.getServiceAddressAssignmentDetails(ctx, *svc, children)
}

func (f *Facade) getServiceAddressAssignmentDetails(ctx datastore.Context, svc service.Service, children bool) ([]service.IPAssignment, error) {
	logger := plog.WithFields(log.Fields{
		"serviceid":   svc.ID,
		"servicename": svc.Name,
		"poolid":      svc.PoolID,
	})
	store := addressassignment.NewStore()

	// Initialize the object and the potential address.  The assumption is that
	// all address assignments for a service must point to the same ip.
	addrs := []service.IPAssignment{}
	addr := service.IPAssignment{
		ServiceID:   svc.ID,
		ServiceName: svc.Name,
		PoolID:      svc.PoolID,
	}

	// Look for endpoints with address assignments
	for _, ep := range svc.Endpoints {
		if ep.AddressConfig.Port > 0 {

			if addr.IPAddress == "" {

				// Get the assignment if it exists
				assign, err := store.FindAssignmentByServiceEndpoint(ctx, svc.ID, ep.Name)
				if err != nil {
					logger.WithError(err).Debug("Could not look up assignment")
					return nil, err
				}

				if assign != nil {
					// Get the host info for the address assignment
					var hostID, hostName string

					if assign.AssignmentType == "virtual" {
						hostID, _ = f.zzk.GetVirtualIPHostID(svc.PoolID, assign.IPAddr)
					} else {
						hostID = assign.HostID
					}

					if hostID != "" {
						hst := &host.Host{}
						if err := f.hostStore.Get(ctx, host.HostKey(hostID), hst); err != nil {
							logger.WithField("hostid", hostID).WithError(err).Debug("Could not look up host for address assignment")
							return nil, err
						}
						hostName = hst.Name
					}

					addr.Type = assign.AssignmentType
					addr.HostID = hostID
					addr.HostName = hostName
					addr.IPAddress = assign.IPAddr
					logger.Debug("Set address assignment for service")
				}
			}

			// Append a new record for the port
			addr.Port = ep.AddressConfig.Port
			addrs = append(addrs, addr)
			logger.WithField("port", ep.AddressConfig.Port).Debug("Added port")
		}
	}

	if children {
		svcs, err := f.serviceStore.GetChildServices(ctx, svc.ID)
		if err != nil {
			logger.WithError(err).Debug("Could not look up children of service")
			return nil, err
		}
		for _, svc := range svcs {
			childAddrs, err := f.getServiceAddressAssignmentDetails(ctx, svc, children)
			if err != nil {
				return nil, err
			}
			addrs = append(addrs, childAddrs...)
		}
	}

	return addrs, nil
}

// GetServiceAddressAssignments fills in all address assignments for the specified service id.
func (f *Facade) GetServiceAddressAssignments(ctx datastore.Context, serviceID string) ([]addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceAddressAssignments"))
	store := addressassignment.NewStore()
	return store.GetServiceAddressAssignments(ctx, serviceID)
}

func (f *Facade) GetServiceAddressAssignmentsByPort(ctx datastore.Context, port uint16) ([]addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceAddressAssignmentsByPort"))
	store := addressassignment.NewStore()
	return store.GetServiceAddressAssignmentsByPort(ctx, port)
}

// GetAddressAssignmentsByEndpoint returns the address assignment by serviceID and endpoint name
func (f *Facade) FindAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpointName string) (*addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("FindAssignmentByServiceEndpoint"))
	store := addressassignment.NewStore()
	return store.FindAssignmentByServiceEndpoint(ctx, serviceID, endpointName)
}

func (f *Facade) FindAssignmentByHostPort(ctx datastore.Context, ipAddr string, port uint16) (*addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("FindAssignmentByHostPort"))
	store := addressassignment.NewStore()
	return store.FindAssignmentByHostPort(ctx, ipAddr, port)
}

// RemoveAddressAssignment Removes an AddressAssignment by id
func (f *Facade) RemoveAddressAssignment(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("RemoveAddressAssignment"))
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
