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
	"strings"
)

// GetServiceAddressAssignmentDetails provides details about address assignments
// for the specified service id as is presented to the front-end.
func (f *Facade) GetServiceAddressAssignmentDetails(ctx datastore.Context, serviceID string, children bool) ([]service.IPAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceAddressAssignmentDetails"))
	return f.getServiceAddressAssignmentDetails(ctx, serviceID, children)
}

func (f *Facade) getServiceAddressAssignmentDetails(ctx datastore.Context, serviceID string, children bool) ([]service.IPAssignment, error) {
	logger := plog.WithFields(log.Fields{
		"parentserviceid":  serviceID,
		"children":    children,
	})
	store := addressassignment.NewStore()
	allAddrs, err := store.GetAllAddressAssignments(ctx)
	if err != nil {
		logger.WithError(err).Error("Could not look up address assignments")
		return nil, err
	} else if len(allAddrs) == 0 {
		return []service.IPAssignment{}, nil
	}

	// Build a map of serviceID+endpointName
	addrMap := make(map[string]addressassignment.AddressAssignment)
	for _, addr := range(allAddrs) {
		key := fmt.Sprintf("%s-%s", addr.ServiceID, addr.EndpointName)
		addrMap[key] = addr
	}

	// Build a list of service IDs for all child services that have address assignments
	serviceIDs := make(map[string]string)
	if children {
		gs := func(id string) (*service.ServiceDetails, error) {
			return f.GetServiceDetails(ctx, id)
		}

		for _, addr := range(allAddrs) {
			if _, ok := serviceIDs[addr.ServiceID]; ok {
				continue
			}
			_, servicePath, err := f.serviceCache.GetServicePath(addr.ServiceID, gs)
			if err != nil {
				logger.WithError(err).WithField("serviceid", addr.ServiceID).Error("Could not find service")
				return nil, err
			}
			if strings.Contains(servicePath, serviceID) {
				serviceIDs[addr.ServiceID] = addr.ServiceID
			}
		}
	} else {
		serviceIDs[serviceID] = serviceID
	}

	ipAssignments := []service.IPAssignment{}
	for _, id := range(serviceIDs) {
		svc, err := f.serviceStore.Get(ctx, id)
		if err != nil {
			logger.WithError(err).WithField("serviceid", id).Error("Could not find service")
			return nil, err
		}
		servicelogger := logger.WithFields(log.Fields{
			"serviceid":   svc.ID,
			"servicename": svc.Name,
			"poolid":      svc.PoolID,
			"children":    children,
		})
		assign := service.IPAssignment{
			ServiceID:   svc.ID,
			ServiceName: svc.Name,
			PoolID:      svc.PoolID,
		}
		for _, ep := range svc.Endpoints {
			if ep.AddressConfig.Port > 0 {
				if assign.IPAddress == "" {
					key := fmt.Sprintf("%s-%s", svc.ID, ep.Name)
					addr, ok := addrMap[key]
					if !ok {
						err := fmt.Errorf("Can not find address assignment")
						servicelogger.WithFields(log.Fields{
							"endpointname": ep.Name,
						}).WithError(err).Error("Address assignment lookup failed")
						return nil, err
					}
					// Get the host info for the address assignment
					var hostID, hostName string
					if addr.AssignmentType == "virtual" {
						hostID, _ = f.zzk.GetVirtualIPHostID(svc.PoolID, addr.IPAddr)
					} else {
						hostID = addr.HostID
					}

					if hostID != "" {
						hst := &host.Host{}
						if err := f.hostStore.Get(ctx, host.HostKey(hostID), hst); err != nil {
							servicelogger.WithField("hostid", hostID).WithError(err).Debug("Could not look up host for address assignment")
							return nil, err
						}
						hostName = hst.Name
					}

					assign.Type = addr.AssignmentType
					assign.HostID = hostID
					assign.HostName = hostName
					assign.IPAddress = addr.IPAddr
					servicelogger.Debug("Set address assignment for service")

				}
				// Append a new record for the port
				assign.Port = ep.AddressConfig.Port
				assign.Application = ep.Application
				ipAssignments = append(ipAssignments, assign)
				servicelogger.WithField("port", ep.AddressConfig.Port).Debug("Added port")
			}
		}
	}
	return ipAssignments, nil
}

// GetServiceAddressAssignments fills in all address assignments for the specified service id.
func (f *Facade) GetServiceAddressAssignments(ctx datastore.Context, serviceID string) ([]addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceAddressAssignments"))
	store := addressassignment.NewStore()
	return store.GetServiceAddressAssignments(ctx, serviceID)
}

func (f *Facade) GetServiceAddressAssignmentsByPort(ctx datastore.Context, port uint16) ([]addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceAddressAssignmentsByPort"))
	store := addressassignment.NewStore()
	return store.GetServiceAddressAssignmentsByPort(ctx, port)
}

// GetAddressAssignmentsByEndpoint returns the address assignment by serviceID and endpoint name
func (f *Facade) FindAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpointName string) (*addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.FindAssignmentByServiceEndpoint"))
	store := addressassignment.NewStore()
	return store.FindAssignmentByServiceEndpoint(ctx, serviceID, endpointName)
}

func (f *Facade) FindAssignmentByHostPort(ctx datastore.Context, ipAddr string, port uint16) (*addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.FindAssignmentByHostPort"))
	store := addressassignment.NewStore()
	return store.FindAssignmentByHostPort(ctx, ipAddr, port)
}

// RemoveAddressAssignment Removes an AddressAssignment by id
func (f *Facade) RemoveAddressAssignment(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemoveAddressAssignment"))
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
