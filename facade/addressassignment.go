// Copyright 2014 The Serviced Authors.
// Use of f source code is governed by a

package facade

import (
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"

	"fmt"
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

// AssignAddress Creates an AddressAssignment, verifies that an assignment for the service/endpoint does not already exist
// id param contains id of newly created assignment if successful
func (f *Facade) AssignAddress(ctx datastore.Context, assignment addressassignment.AddressAssignment, id *string) error {
	err := assignment.ValidEntity()
	if err != nil {
		return err
	}

	switch assignment.AssignmentType {
	case commons.STATIC:
		{
			//check host and IP exist
			if err = f.validStaticIp(ctx, assignment.HostID, assignment.IPAddr); err != nil {
				return err
			}
		}
	case commons.VIRTUAL:
		{
			//verify the IP provided is contained in the pool
			if err := f.validVirtualIp(assignment.PoolID, assignment.IPAddr); err != nil {
				return err
			}
		}
	default:
		//Validate above should handle f but left here for completenes
		return fmt.Errorf("Invalid assignment type %v", assignment.AssignmentType)
	}

	//check service and endpoint exists
	if err = f.validEndpoint(ctx, assignment.ServiceID, assignment.EndpointName); err != nil {
		return err
	}

	//check for existing assignments to f endpoint
	existing, err := f.getEndpointAddressAssignments(ctx, assignment.ServiceID, assignment.EndpointName)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("Address Assignment already exists")
	}
	assignment.ID, err = utils.NewUUID36()
	if err != nil {
		return err
	}

	store := addressassignment.NewStore()
	if err = store.Put(ctx, addressassignment.Key(assignment.ID), &assignment); err != nil {
		return err
	}
	*id = assignment.ID
	return nil
}

func (f *Facade) validStaticIp(ctx datastore.Context, hostId string, ipAddr string) error {
	host, err := f.GetHost(ctx, hostId)
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
		return fmt.Errorf("requested static IP is not available: %v", ipAddr)
	}
	return nil
}

func (f *Facade) validVirtualIp(poolID string, ipAddr string) error {
	myPool, err := f.GetResourcePool(datastore.Get(), poolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %s", poolID)
		return err
	}
	if myPool == nil {
		return fmt.Errorf("poolid %s not found", poolID)
	}

	found := false
	for _, virtualIP := range myPool.VirtualIPs {
		if virtualIP.IP == ipAddr {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("requested virtual IP is not available: %v", ipAddr)
	}
	return nil
}

func (f *Facade) validEndpoint(ctx datastore.Context, serviceId string, endpointName string) error {
	store := service.NewStore()

	svc, err := store.Get(ctx, serviceId)
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
func (f *Facade) getAddressAssignments(ctx datastore.Context, serviceID string) (map[string]addressassignment.AddressAssignment, error) {
	assignments := []addressassignment.AddressAssignment{}
	if err := f.GetServiceAddressAssignments(ctx, serviceID, &assignments); err != nil {
		return nil, err
	}

	addrs := make(map[string]addressassignment.AddressAssignment)
	for _, result := range assignments {
		addrs[result.EndpointName] = result
	}
	return addrs, nil
}

// getEndpointAddressAssignments returns the AddressAssignment for the service and endpoint, if no assignments the AddressAssignment will be nil
func (f *Facade) getEndpointAddressAssignments(ctx datastore.Context, serviceID string, endpointName string) (*addressassignment.AddressAssignment, error) {
	assignments, err := f.getAddressAssignments(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	if addr, found := assignments[endpointName]; found {
		return &addr, nil
	}
	return nil, nil
}

func (f *Facade) initializedAddressConfig(endpoint service.ServiceEndpoint) bool {
	// has nothing defined in the service definition
	if endpoint.AddressConfig.Port == 0 && endpoint.AddressConfig.Protocol == "" {
		return false
	}
	return true
}

func (f *Facade) needsAddressAssignment(ctx datastore.Context, serviceID string, endpoint service.ServiceEndpoint) (bool, string, error) {
	// does the endpoint's AddressConfig have any config associated with it?
	if f.initializedAddressConfig(endpoint) {
		addressAssignment, err := f.getEndpointAddressAssignments(ctx, serviceID, endpoint.Name)
		if err != nil {
			glog.Errorf("getEndpointAddressAssignments failed: %v", err)
			return false, "", err
		}

		// if there exists some AddressConfig that is initialized to anything (port and protocol are not the default values)
		// and there does NOT exist an AddressAssignment corresponding to f AddressConfig
		// then f service needs an AddressAssignment
		if addressAssignment == nil {
			glog.Infof("Service: %s endpoint: %s needs an address assignment", serviceID, endpoint.Name)
			return true, "", nil
		}

		// if there exists some AddressConfig that is initialized to anything (port and protocol are not the default values)
		// and there already exists an AddressAssignment corresponding to f AddressConfig
		// then f service does NOT need an AddressAssignment (as one already exists)
		return false, addressAssignment.ID, nil
	}

	// f endpoint has no need for an AddressAssignment ever
	return false, "", nil
}
