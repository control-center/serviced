// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/addressassignment"
	"github.com/zenoss/serviced/domain/service"

	"fmt"
)

// GetServiceAddressAssignments fills in all AddressAssignments for the specified serviced id.
func (this *ControlPlaneDao) GetServiceAddressAssignments(serviceID string, assignments *[]*addressassignment.AddressAssignment) error {
	store := addressassignment.NewStore()

	results, err := store.GetServiceAddressAssignments(datastore.Get(), serviceID)
	if err != nil {
		return err
	}
	*assignments = results
	return nil
}

// RemoveAddressAssignemnt Removes an AddressAssignment by id
func (this *ControlPlaneDao) RemoveAddressAssignment(id string, _ *struct{}) error {
	store := addressassignment.NewStore()
	key := addressassignment.Key(id)

	var assignment addressassignment.AddressAssignment
	if err := store.Get(datastore.Get(), key, &assignment); err != nil {
		return err
	}

	if err := store.Delete(datastore.Get(), key); err != nil {
		return err
	}

	var svc service.Service
	if err := this.GetService(assignment.ServiceID, &svc); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetService service=%+v err=%s", assignment.ServiceID, err)
		return err
	}

	if err := this.updateService(&svc); err != nil {
		glog.V(2).Infof("ControlPlaneDao.updateService service=%+v err=%s", assignment.ServiceID, err)
		return err
	}

	return nil
}

// AssignAddress Creates an AddressAssignment, verifies that an assignment for the service/endpoint does not already exist
// id param contains id of newly created assignment if successful
func (this *ControlPlaneDao) AssignAddress(assignment addressassignment.AddressAssignment, id *string) error {
	err := assignment.ValidEntity()
	if err != nil {
		return err
	}

	switch assignment.AssignmentType {
	case "static":
		{
			//check host and IP exist
			if err = this.validStaticIp(assignment.HostID, assignment.IPAddr); err != nil {
				return err
			}
		}
	case "virtual":
		{
			// TODO: need to check if virtual IP exists
			return fmt.Errorf("Not yet supported type %v", assignment.AssignmentType)
		}
	default:
		//Validate above should handle this but left here for completenes
		return fmt.Errorf("Invalid assignment type %v", assignment.AssignmentType)
	}

	//check service and endpoint exists
	if err = this.validEndpoint(assignment.ServiceID, assignment.EndpointName); err != nil {
		return err
	}

	//check for existing assignments to this endpoint
	existing, err := this.getEndpointAddressAssignments(assignment.ServiceID, assignment.EndpointName)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("Address Assignment already exists")
	}
	assignment.ID, err = dao.NewUuid()
	if err != nil {
		return err
	}

	store := addressassignment.NewStore()
	if err = store.Put(datastore.Get(), addressassignment.Key(assignment.ID), &assignment); err != nil {
		return err
	}
	*id = assignment.ID
	return nil
}

func (this *ControlPlaneDao) validStaticIp(hostId string, ipAddr string) error {

	host, err := this.facade.GetHost(datastore.Get(), hostId)
	if err != nil {
		return err
	}
	if host == nil {
		return fmt.Errorf("host not found: %v", hostId)
	}
	found := false
	for _, ip := range host.IPs {
		if ip.IPAddress == ipAddr {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Requested static IP is not available: %v", ipAddr)
	}
	return nil
}

func (this *ControlPlaneDao) validEndpoint(serviceId string, endpointName string) error {
	store := service.NewStore()

	svc, err := store.Get(datastore.Get(), serviceId)
	if err != nil {
		return err
	}
	found := false
	for _, endpoint := range svc.Endpoints {
		if endpointName == endpoint.Name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Endpoint %v not found on service %v", endpointName, serviceId)
	}
	return nil
}

// getEndpointAddressAssignments returns the AddressAssignment for the service and endpoint, if no assignments the AddressAssignment will be nil
func (this *ControlPlaneDao) getEndpointAddressAssignments(serviceId string, endpointName string) (*addressassignment.AddressAssignment, error) {
	//TODO: this can probably be done w/ a query
	assignments := []*addressassignment.AddressAssignment{}
	err := this.GetServiceAddressAssignments(serviceId, &assignments)
	if err != nil {
		return nil, err
	}

	for _, result := range assignments {
		if result.EndpointName == endpointName {
			return result, nil
		}
	}
	return nil, nil
}

func (this *ControlPlaneDao) initializedAddressConfig(endpoint service.ServiceEndpoint) bool {
	// has nothing defined in the service definition
	if endpoint.AddressConfig.Port == 0 && endpoint.AddressConfig.Protocol == "" {
		return false
	}
	return true
}

func (this *ControlPlaneDao) needsAddressAssignment(serviceID string, endpoint service.ServiceEndpoint) (bool, string, error) {
	// does the endpoint's AddressConfig have any config associated with it?
	if this.initializedAddressConfig(endpoint) {
		addressAssignment, err := this.getEndpointAddressAssignments(serviceID, endpoint.Name)
		if err != nil {
			glog.Errorf("getEndpointAddressAssignments failed: %v", err)
			return false, "", err
		}

		// if there exists some AddressConfig that is initialized to anything (port and protocol are not the default values)
		// and there does NOT exist an AddressAssignment corresponding to this AddressConfig
		// then this service needs an AddressAssignment
		if addressAssignment == nil {
			glog.Infof("Service: %s endpoint: %s needs an address assignment", serviceID, endpoint.Name)
			return true, "", nil
		}

		// if there exists some AddressConfig that is initialized to anything (port and protocol are not the default values)
		// and there already exists an AddressAssignment corresponding to this AddressConfig
		// then this service does NOT need an AddressAssignment (as one already exists)
		return false, addressAssignment.ID, nil
	}

	// this endpoint has no need for an AddressAssignment ever
	return false, "", nil
}
