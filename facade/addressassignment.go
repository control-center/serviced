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

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/audit"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
)

// GetServiceAddressAssignmentDetails provides details about address assignments
// for the specified service id as is presented to the front-end.
func (f *Facade) GetServiceAddressAssignmentDetails(ctx datastore.Context, serviceID string, children bool) ([]service.IPAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceAddressAssignmentDetails"))
	logger := plog.WithFields(log.Fields{
		"parentserviceid": serviceID,
		"children":        children,
	})

	// Get a list of all ip assignments in the entire system based on services with endpoints
	// (not records in the addressassignment store)
	allIPs, err := f.serviceStore.GetAllIPAssignments(ctx)
	if err != nil {
		logger.WithError(err).Error("Could not look up address assignments")
		return nil, err
	} else if len(allIPs) == 0 {
		return []service.IPAssignment{}, nil
	}

	// Build a map of those assignments which belong to the specified serviceID or any of its children
	serviceIPs := make(map[string]service.BaseIPAssignment)
	gs := func(id string) (*service.ServiceDetails, error) {
		return f.GetServiceDetails(ctx, id)
	}
	for _, ip := range allIPs {
		key := fmt.Sprintf("%s-%s", ip.ServiceID, ip.EndpointName)
		if _, ok := serviceIPs[key]; ok {
			continue
		}
		if ip.ServiceID == serviceID {
			serviceIPs[key] = ip
		} else if children {
			_, servicePath, err := f.serviceCache.GetServicePath(ip.ServiceID, gs)
			if err != nil {
				logger.WithError(err).WithField("serviceid", ip.ServiceID).Error("Could not find service")
				return nil, err
			}
			if strings.Contains(servicePath, serviceID) {
				serviceIPs[key] = ip
			}
		}
	}

	// For each service endpoint that needs an address assignement,
	// 	retrieve the corresponding assignment from the DB
	ipAssignments := []service.IPAssignment{}
	for _, ip := range serviceIPs {
		servicelogger := logger.WithFields(log.Fields{
			"serviceid":    ip.ServiceID,
			"servicename":  ip.ServiceName,
			"poolid":       ip.PoolID,
			"endpointname": ip.EndpointName,
			"children":     children,
		})
		ipassignment := service.IPAssignment{
			BaseIPAssignment: ip,
		}
		addr, err := f.addressassignmentStore.FindAssignmentByServiceEndpoint(ctx, ip.ServiceID, ip.EndpointName)
		if err != nil {
			err := fmt.Errorf("Can not find address assignment")
			servicelogger.WithFields(log.Fields{
				"endpointname": ip.EndpointName,
			}).WithError(err).Error("Address assignment lookup failed")
			return nil, err
		}

		if addr != nil {
			// Get the host info for the address assignment
			var hostID, hostName string
			if addr.AssignmentType == "virtual" {
				hostID, _ = f.zzk.GetVirtualIPHostID(ip.PoolID, addr.IPAddr)
			} else {
				hostID = addr.HostID
			}

			if hostID != "" {
				hst := &host.Host{}
				if err := f.hostStore.Get(ctx, host.Key(hostID), hst); err != nil {
					servicelogger.WithField("hostid", hostID).WithError(err).Debug("Could not look up host for address assignment")
					return nil, err
				}
				hostName = hst.Name
			}

			ipassignment.Type = addr.AssignmentType
			ipassignment.HostID = hostID
			ipassignment.HostName = hostName
			ipassignment.IPAddress = addr.IPAddr
			servicelogger.Debug("Set address assignment for service endpoint")
		} else {
			servicelogger.Debug("No address assignment available for service endpoint")
		}

		// Append a new record for the port
		ipAssignments = append(ipAssignments, ipassignment)
	}
	return ipAssignments, nil
}

// GetServiceAddressAssignments fills in all address assignments for the specified service id.
func (f *Facade) GetServiceAddressAssignments(ctx datastore.Context, serviceID string) ([]addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceAddressAssignments"))
	return f.addressassignmentStore.GetServiceAddressAssignments(ctx, serviceID)
}

// GetServiceAddressAssignmentsByPort returns address assignments by port
func (f *Facade) GetServiceAddressAssignmentsByPort(ctx datastore.Context, poolID string, port uint16) ([]addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceAddressAssignmentsByPort"))
	return f.addressassignmentStore.GetServiceAddressAssignmentsByPort(ctx, poolID, port)
}

// FindAssignmentByServiceEndpoint returns the address assignment by serviceID and endpoint name
func (f *Facade) FindAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpointName string) (*addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.FindAssignmentByServiceEndpoint"))
	return f.addressassignmentStore.FindAssignmentByServiceEndpoint(ctx, serviceID, endpointName)
}

// FindAssignmentByHostPort returns assignment by host and port
func (f *Facade) FindAssignmentByHostPort(ctx datastore.Context, poolID, ipAddr string, port uint16) (*addressassignment.AddressAssignment, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.FindAssignmentByHostPort"))
	return f.addressassignmentStore.FindAssignmentByHostPort(ctx, poolID, ipAddr, port)
}

// RemoveAddressAssignment Removes an AddressAssignment by id
func (f *Facade) RemoveAddressAssignment(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemoveAddressAssignment"))
	alog := f.auditLogger.Message(ctx, "Removing AddressAssignment").Action(audit.Remove).
		ID(id).Type(addressassignment.GetType())
	key := addressassignment.Key(id)

	var assignment addressassignment.AddressAssignment
	if err := f.addressassignmentStore.Get(ctx, key, &assignment); err != nil {
		return alog.Error(err)
	}
	alog = alog.WithFields(log.Fields{
		"ipaddr":         assignment.IPAddr,
		"port":           assignment.Port,
		"endpointname":   assignment.EndpointName,
		"assignmenttype": assignment.AssignmentType,
	})

	if err := f.addressassignmentStore.Delete(ctx, key); err != nil {
		return alog.Error(err)
	}

	alog.Succeeded()
	return nil
}

func (f *Facade) assign(ctx datastore.Context, assignment addressassignment.AddressAssignment) (string, error) {
	alog := f.auditLogger.Message(ctx, "Adding AddressAssignment").Action(audit.Add).Type(assignment.GetType())
	if err := assignment.ValidEntity(); err != nil {
		return "", alog.Error(err)
	}

	// Do not add if it already exists
	if exists, err := f.FindAssignmentByServiceEndpoint(ctx, assignment.ServiceID, assignment.EndpointName); err != nil {
		return "", alog.Error(err)
	} else if exists != nil {
		return "", alog.Error(fmt.Errorf("found assignment for %s at %s", assignment.EndpointName, assignment.ServiceID))
	}

	// Do not add if already assigned
	if exists, err := f.FindAssignmentByHostPort(ctx, assignment.PoolID, assignment.IPAddr, assignment.Port); err != nil {
		return "", alog.Error(err)
	} else if exists != nil {
		return "", alog.Error(fmt.Errorf("found assignment for port %d at %s", assignment.Port, assignment.IPAddr))
	}

	alog = alog.WithFields(log.Fields{
		"ipaddr":         assignment.IPAddr,
		"port":           assignment.Port,
		"endpointname":   assignment.EndpointName,
		"assignmenttype": assignment.AssignmentType,
	})

	var err error
	if assignment.ID, err = utils.NewUUID36(); err != nil {
		return "", alog.Error(err)
	}

	if err := f.addressassignmentStore.Put(ctx, addressassignment.Key(assignment.ID), &assignment); err != nil {
		return "", alog.ID(assignment.ID).Error(err)
	}

	alog.Succeeded()
	return assignment.ID, nil
}
