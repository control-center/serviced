// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/addressassignment"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"

	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// AddService add a service. Return error if service already exists
func (this *ControlPlaneDao) AddService(svc service.Service, serviceId *string) error {
	glog.V(0).Infof("ControlPlaneDao.AddService: %+v", svc)
	store := service.NewStore()

	id := strings.TrimSpace(svc.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}
	svc.Id = id

	found := service.Service{}
	if err := store.Get(datastore.Get(), service.Key(svc.Id), &found); err != nil && !datastore.IsErrNoSuchEntity(err) {
		return err
	} else if err == nil {
		return fmt.Errorf("error adding service; %v already exists", id)
	}

	err := store.Put(datastore.Get(), service.Key(svc.Id), &svc)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.AddService: %+v", err)
		return err
	}
	*serviceId = id
	glog.V(0).Infof("ControlPlaneDao.AddService: id %+v; return id %v", id, serviceId)

	return this.zkDao.AddService(&svc)
}

// updateService internal method to use when service has been validated
func (this *ControlPlaneDao) updateService(svc *service.Service) error {
	id := strings.TrimSpace(svc.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}
	svc.Id = id
	//add assignment info to service
	for idx := range svc.Endpoints {
		assignment, err := this.getEndpointAddressAssignments(svc.Id, svc.Endpoints[idx].Name)
		if err != nil {
			glog.Errorf("ControlPlaneDao.UpdateService Error looking up address assignments: %v", err)
			return err
		}
		if assignment != nil {
			//assignment exists
			glog.V(4).Infof("ControlPlaneDao.UpdateService setting address assignment on endpoint: %s, %v", svc.Endpoints[idx].Name, assignment)
			svc.Endpoints[idx].SetAssignment(assignment)
		} else {
			svc.Endpoints[idx].RemoveAssignment()
		}
	}

	store := service.NewStore()
	ctx := datastore.Get()
	if err := store.Put(ctx, service.Key(id), svc); err != nil {
		return err
	}
	return this.zkDao.UpdateService(svc)
}

//
func (this *ControlPlaneDao) UpdateService(svc service.Service, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateService: %+v", svc)
	//cannot update service without validating it.
	if svc.DesiredState == service.SVCRun {
		if err := this.validateServicesForStarting(&svc, nil); err != nil {
			return err
		}

	}
	return this.updateService(&svc)
}

//
func (this *ControlPlaneDao) RemoveService(id string, unused *int) error {
	//TODO: should services already be stopped before removing to prevent half running service in case of error while deleting?

	err := this.walkServices(id, func(svc *service.Service) error {
		this.zkDao.RemoveService(svc.Id)
		return nil
	})

	if err != nil {
		//TODO: should we put them back?
		return err
	}

	store := service.NewStore()
	ctx := datastore.Get()

	err = this.walkServices(id, func(svc *service.Service) error {
		err := store.Delete(ctx, service.Key(svc.Id))
		if err != nil {
			glog.Errorf("Error removing service %s	 %s ", svc.Id, err)
		}
		return err
	})
	if err != nil {
		return err
	}
	//TODO: remove AddressAssignments with this Service
	return nil
}

//
func (this *ControlPlaneDao) GetService(id string, myService *service.Service) error {
	glog.V(3).Infof("ControlPlaneDao.GetService: id=%s", id)
	store := service.NewStore()
	request := service.Service{}
	err := store.Get(datastore.Get(), service.Key(id), &request)
	glog.V(3).Infof("ControlPlaneDao.GetService: id=%s, service=%+v, err=%s", id, request, err)
	*myService = request
	return err
}

//
func (this *ControlPlaneDao) GetServices(request dao.EntityRequest, services *[]*service.Service) error {
	glog.V(3).Infof("ControlPlaneDao.GetServices")
	store := service.NewStore()
	results, err := store.GetServices(datastore.Get())
	if err != nil {
		glog.Error("ControlPlaneDao.GetServices: err=", err)
		return err
	}
	*services = results
	return nil
}

//
func (this *ControlPlaneDao) GetTaggedServices(request dao.EntityRequest, services *[]*service.Service) error {
	glog.V(3).Infof("ControlPlaneDao.GetTaggedServices")

	store := service.NewStore()
	switch v := request.(type) {
	case []string:
		results, err := store.GetTaggedServices(datastore.Get(), v...)
		if err != nil {
			glog.Error("ControlPlaneDao.GetTaggedServices: err=", err)
			return err
		}
		*services = results
		glog.V(2).Infof("ControlPlaneDao.GetTaggedServices: services=%v", services)
		return nil
	default:
		err := fmt.Errorf("Bad request type: %v", v)
		glog.V(2).Info("ControlPlaneDao.GetTaggedServices: err=", err)
		return err
	}
}

// The tenant id is the root service uuid. Walk the service tree to root to find the tenant id.
func (this *ControlPlaneDao) GetTenantId(serviceId string, tenantId *string) (err error) {
	glog.V(2).Infof("ControlPlaneDao.GetTenantId: %s", serviceId)
	id := strings.TrimSpace(serviceId)
	if id == "" {
		return errors.New("empty serviceId not allowed")
	}

	var traverse func(string) (string, error)

	traverse = func(id string) (string, error) {
		var service service.Service
		if err := this.GetService(id, &service); err != nil {
			return "", err
		} else if service.ParentServiceId != "" {
			return traverse(service.ParentServiceId)
		} else {
			glog.V(1).Infof("parent service: %+v", service)
			return service.Id, nil
		}
	}

	*tenantId, err = traverse(id)
	return
}

// Get a service endpoint.
func (this *ControlPlaneDao) GetServiceEndpoints(serviceId string, response *map[string][]*dao.ApplicationEndpoint) (err error) {
	glog.V(2).Infof("ControlPlaneDao.GetServiceEndpoints serviceId=%s", serviceId)
	var myService service.Service
	err = this.GetService(serviceId, &myService)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceEndpoints service=%+v err=%s", myService, err)
		return
	}

	service_imports := myService.GetServiceImports()
	if len(service_imports) > 0 {
		glog.V(2).Infof("%+v service imports=%+v", myService, service_imports)

		var request dao.EntityRequest
		var servicesList []*service.Service
		err = this.GetServices(request, &servicesList)
		if err != nil {
			return
		}

		// Map all services by Id so we can construct a tree for the current service ID
		glog.V(2).Infof("ServicesList: %d", len(servicesList))
		topService := this.getServiceTree(serviceId, &servicesList)
		// We should now have the top-level service for the current service ID
		remoteEndpoints := make(map[string][]*dao.ApplicationEndpoint)

		//build 'OR' query to grab all service states with in "service" tree
		relatedServiceIds := walkTree(topService)
		var states []*servicestate.ServiceState
		err = this.zkDao.GetServiceStates(&states, relatedServiceIds...)
		if err != nil {
			return
		}

		// for each proxied port, find list of potential remote endpoints
		for _, endpoint := range service_imports {
			glog.V(2).Infof("Finding exports for import: %s %+v", endpoint.Application, endpoint)
			matchedEndpoint := false
			applicationRegex, err := regexp.Compile(fmt.Sprintf("^%s$", endpoint.Application))
			if err != nil {
				continue //Don't spam error message; it was reported at validation time
			}
			for _, ss := range states {
				hostPort, containerPort, protocol, match := ss.GetHostEndpointInfo(applicationRegex)
				if match {
					glog.V(1).Infof("Matched endpoint: %s.%s -> %s:%d (%s/%d)",
						myService.Name, endpoint.Application, ss.HostIp, hostPort, protocol, containerPort)
					// if port/protocol undefined in the import, use the export's values
					if endpoint.PortNumber != 0 {
						containerPort = endpoint.PortNumber
					}
					if endpoint.Protocol != "" {
						protocol = endpoint.Protocol
					}
					var ep dao.ApplicationEndpoint
					ep.ServiceId = ss.ServiceId
					ep.ContainerPort = containerPort
					ep.HostPort = hostPort
					ep.HostIp = ss.HostIp
					ep.ContainerIp = ss.PrivateIp
					ep.Protocol = protocol
					ep.VirtualAddress = endpoint.VirtualAddress

					key := fmt.Sprintf("%s:%d", protocol, containerPort)
					if _, exists := remoteEndpoints[key]; !exists {
						remoteEndpoints[key] = make([]*dao.ApplicationEndpoint, 0)
					}
					remoteEndpoints[key] = append(remoteEndpoints[key], &ep)
					matchedEndpoint = true
				}
			}
			if !matchedEndpoint {
				glog.V(1).Infof("Unmatched endpoint %s.%s", myService.Name, endpoint.Application)
			}
		}

		*response = remoteEndpoints
		glog.V(2).Infof("Return for %s is %+v", serviceId, remoteEndpoints)
	}
	return
}

// start the provided service
func (this *ControlPlaneDao) StartService(serviceId string, unused *string) error {
	// this will traverse all the services
	err := this.validateService(serviceId)
	if err != nil {
		return err
	}

	visitor := func(svc *service.Service) error {
		//start this service
		svc.DesiredState = service.SVCRun
		err = this.updateService(svc)
		if err != nil {
			return err
		}
		return nil
	}

	// traverse all the services
	return this.walkServices(serviceId, visitor)
}
func (this *ControlPlaneDao) StopService(id string, unused *int) error {
	glog.V(0).Info("ControlPlaneDao.StopService id=", id)

	visitor := func(svc *service.Service) error {
		//start this service
		if svc.Launch == commons.MANUAL {
			return nil
		}
		svc.DesiredState = service.SVCStop
		if err := this.updateService(svc); err != nil {
			return err
		}
		return nil
	}

	// traverse all the services
	return this.walkServices(id, visitor)
}

// assign an IP address to a service (and all its child services) containing non default AddressResourceConfig
func (this *ControlPlaneDao) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) error {
	myService := service.Service{}
	err := this.GetService(assignmentRequest.ServiceId, &myService)
	if err != nil {
		return err
	}

	// populate poolsIpInfo
	poolIPs, err := this.facade.GetPoolIPs(datastore.Get(), myService.PoolId)
	if err != nil {
		glog.Errorf("GetPoolsIPInfo failed: %v", err)
		return err
	}
	poolsIpInfo := poolIPs.HostIPs
	if len(poolsIpInfo) < 1 {
		msg := fmt.Sprintf("No IP addresses are available in pool %s.", myService.PoolId)
		return errors.New(msg)
	}
	glog.Infof("Pool %v contains %v available IP(s)", myService.PoolId, len(poolsIpInfo))

	rand.Seed(time.Now().UTC().UnixNano())
	ipIndex := 0
	userProvidedIPAssignment := false

	if assignmentRequest.AutoAssignment {
		// automatic IP requested
		glog.Infof("Automatic IP Address Assignment")
		ipIndex = rand.Intn(len(poolsIpInfo))
	} else {
		// manual IP provided
		// verify that the user provided IP address is available in the pool
		glog.Infof("Manual IP Address Assignment")
		validIp := false
		userProvidedIPAssignment = true

		for index, hostIPResource := range poolsIpInfo {
			if assignmentRequest.IpAddress == hostIPResource.IPAddress {
				// WHAT HAPPENS IF THERE EXISTS THE SAME IP ON MORE THAN ONE HOST???
				validIp = true
				ipIndex = index
				break
			}
		}

		if !validIp {
			msg := fmt.Sprintf("The requested IP address: %s is not contained in pool %s.", assignmentRequest.IpAddress, myService.PoolId)
			return errors.New(msg)
		}
	}
	assignmentRequest.IpAddress = poolsIpInfo[ipIndex].IPAddress
	selectedHostId := poolsIpInfo[ipIndex].HostID
	glog.Infof("Attempting to set IP address(es) to %s", assignmentRequest.IpAddress)

	assignments := []*addressassignment.AddressAssignment{}
	this.GetServiceAddressAssignments(assignmentRequest.ServiceId, &assignments)
	if err != nil {
		glog.Errorf("controlPlaneDao.GetServiceAddressAssignments failed in anonymous function: %v", err)
		return err
	}

	visitor := func(myService *service.Service) error {
		// if this service is in need of an IP address, assign it an IP address
		for _, endpoint := range myService.Endpoints {
			needsAnAddressAssignment, addressAssignmentId, err := this.needsAddressAssignment(myService.Id, endpoint)
			if err != nil {
				return err
			}

			// if an address assignment is needed (does not yet exist) OR
			// if a specific IP address is provided by the user AND an address assignment already exists
			if needsAnAddressAssignment || (userProvidedIPAssignment && addressAssignmentId != "") {
				if addressAssignmentId != "" {
					glog.Infof("Removing AddressAssignment: %s", addressAssignmentId)
					err = this.RemoveAddressAssignment(addressAssignmentId, nil)
					if err != nil {
						glog.Errorf("controlPlaneDao.RemoveAddressAssignment failed in AssignIPs anonymous function: %v", err)
						return err
					}
				}
				assignment := addressassignment.AddressAssignment{}
				assignment.AssignmentType = "static"
				assignment.HostID = selectedHostId
				assignment.PoolID = myService.PoolId
				assignment.IPAddr = assignmentRequest.IpAddress
				assignment.Port = endpoint.AddressConfig.Port
				assignment.ServiceID = myService.Id
				assignment.EndpointName = endpoint.Name
				glog.Infof("Creating AddressAssignment for Endpoint: %s", assignment.EndpointName)

				var unusedStr string
				err = this.AssignAddress(assignment, &unusedStr)
				if err != nil {
					glog.Errorf("AssignAddress failed in AssignIPs anonymous function: %v", err)
					return err
				}

				err = this.updateService(myService)
				if err != nil {
					glog.Errorf("Failed to update service w/AssignAddressAssignment: %v", err)
					return err
				}

				glog.Infof("Created AddressAssignment: %s for Endpoint: %s", assignment.ID, assignment.EndpointName)
			}
		}
		return nil
	}

	// traverse all the services
	err = this.walkServices(assignmentRequest.ServiceId, visitor)
	if err != nil {
		return err
	}

	glog.Infof("All services requiring an explicit IP address (at this moment) from service: %v and down ... have been assigned: %s", assignmentRequest.ServiceId, assignmentRequest.IpAddress)
	return nil
}

// traverse all the services (including the children of the provided service)
func (this *ControlPlaneDao) walkServices(serviceID string, visitFn service.Visit) error {

	store := service.NewStore()
	ctx := datastore.Get()

	getChildren := func(parentID string) ([]*service.Service, error) {
		return store.GetChildServices(ctx, parentID)
	}
	getService := func(svcID string) (service.Service, error) {
		svc := service.Service{}
		err := store.Get(ctx, service.Key(svcID), &svc)
		return svc, err
	}

	return service.Walk(serviceID, visitFn, getService, getChildren)
}

// walkTree returns a list of ids for all services in a hierarchy rooted by node
func walkTree(node *treenode) []string {
	if len(node.children) == 0 {
		return []string{node.id}
	}
	relatedServiceIds := make([]string, 0)
	for _, childNode := range node.children {
		for _, childId := range walkTree(childNode) {
			relatedServiceIds = append(relatedServiceIds, childId)
		}
	}
	return append(relatedServiceIds, node.id)
}

type treenode struct {
	id       string
	parent   string
	children []*treenode
}

// getServiceTree creates the service hierarchy tree containing serviceId, serviceList is used to create the tree.
// Returns a pointer the root of the service hierarchy
func (this *ControlPlaneDao) getServiceTree(serviceId string, servicesList *[]*service.Service) *treenode {
	glog.V(2).Infof(" getServiceTree = %s", serviceId)
	servicesMap := make(map[string]*treenode)
	for _, svc := range *servicesList {
		servicesMap[svc.Id] = &treenode{
			svc.Id,
			svc.ParentServiceId,
			[]*treenode{},
		}
	}

	// second time through builds our tree
	root := treenode{"root", "", []*treenode{}}
	for _, svc := range *servicesList {
		node := servicesMap[svc.Id]
		parent, found := servicesMap[svc.ParentServiceId]
		// no parent means this node belongs to root
		if !found {
			parent = &root
		}
		parent.children = append(parent.children, node)
	}

	// now walk up the tree, then back down capturing all siblings for this service ID
	topService := servicesMap[serviceId]
	for len(topService.parent) != 0 {
		topService = servicesMap[topService.parent]
	}
	return topService
}

// determine whether the services are ready for deployment
func (this *ControlPlaneDao) validateServicesForStarting(svc *service.Service, _ *struct{}) error {
	// ensure all endpoints with AddressConfig have assigned IPs
	for _, endpoint := range svc.Endpoints {
		needsAnAddressAssignment, addressAssignmentId, err := this.needsAddressAssignment(svc.Id, endpoint)
		if err != nil {
			return err
		}

		if needsAnAddressAssignment {
			msg := fmt.Sprintf("Service ID %s is in need of an AddressAssignment: %s", svc.Id, addressAssignmentId)
			return errors.New(msg)
		} else if addressAssignmentId != "" {
			glog.Infof("AddressAssignment: %s already exists", addressAssignmentId)
		}
	}

	if svc.RAMCommitment < 0 {
		return fmt.Errorf("service RAM commitment cannot be negative")
	}

	// add additional validation checks to the services
	return nil
}

// validate the provided service
func (this *ControlPlaneDao) validateService(serviceId string) error {
	//TODO: create map of IPs to ports and ensure that an IP does not have > 1 process listening on the same port
	visitor := func(service *service.Service) error {
		// validate the service is ready to start
		err := this.validateServicesForStarting(service, nil)
		if err != nil {
			glog.Errorf("Services failed validation for starting")
			return err
		}
		return nil
	}

	// traverse all the services
	return this.walkServices(serviceId, visitor)
}
