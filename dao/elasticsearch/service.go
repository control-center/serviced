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
	"github.com/zenoss/serviced/domain/serviceconfigfile"
	"github.com/zenoss/serviced/domain/servicestate"

	"errors"
	"fmt"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"math/rand"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// AddService add a service. Return error if service already exists
func (this *ControlPlaneDao) AddService(svc service.Service, serviceId *string) error {
	glog.V(2).Infof("ControlPlaneDao.AddService: %+v", svc)
	store := service.NewStore()

	id := strings.TrimSpace(svc.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}
	svc.Id = id

	_, err := store.Get(datastore.Get(), svc.Id)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		return err
	} else if err == nil {
		return fmt.Errorf("error adding service; %v already exists", id)
	}

	err = store.Put(datastore.Get(), &svc)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.AddService: %+v", err)
		return err
	}
	*serviceId = id
	glog.V(2).Infof("ControlPlaneDao.AddService: id %+v; return id %v", id, serviceId)

	return this.zkDao.AddService(&svc)
}

func (this *ControlPlaneDao) fillOutService(ctx datastore.Context, svc *service.Service) error {
	if err := this.fillServiceAddr(svc); err != nil {
		return err
	}
	if err := this.fillServiceConfigs(ctx, svc); err != nil {
		return err
	}
	return nil
}

func (this *ControlPlaneDao) fillOutServices(ctx datastore.Context, svcs []*service.Service) error {
	for _, svc := range svcs {
		if err := this.fillOutService(ctx, svc); err != nil {
			return err
		}
	}
	return nil
}

func (this *ControlPlaneDao) fillServiceConfigs(ctx datastore.Context, svc *service.Service) error {
	glog.V(0).Infof("fillServiceConfigs for %s", svc.Id)
	tenantID, servicePath, err := this.getTenantIdAndPath(svc.Id)
	if err != nil {
		return err
	}
	glog.V(0).Infof("service %v; tenantid=%s; path=%s", svc.Id, tenantID, servicePath)

	configStore := serviceconfigfile.NewStore()
	existingConfs, err := configStore.GetConfigFiles(ctx, tenantID, servicePath)
	if err != nil {
		return err
	}

	//found confs are the modified confs for this service
	foundConfs := make(map[string]*servicedefinition.ConfigFile)
	for _, svcConfig := range existingConfs {
		foundConfs[svcConfig.ConfFile.Filename] = &svcConfig.ConfFile
	}

	//replace with stored service config only if it is an existing config
	for name, conf := range foundConfs {
		if _, found := svc.ConfigFiles[name]; found {
			svc.ConfigFiles[name] = *conf
		}
	}
	return nil
}

func (this *ControlPlaneDao) fillServiceAddr(svc *service.Service) error {
	addrs, err := this.getAddressAssignments(svc.Id)
	if err != nil {
		return err
	}
	for idx := range svc.Endpoints {
		if assignment, found := addrs[svc.Endpoints[idx].Name]; found {
			//assignment exists
			glog.V(4).Infof("setting address assignment on endpoint: %s, %v", svc.Endpoints[idx].Name, assignment)
			svc.Endpoints[idx].SetAssignment(assignment)
		} else {
			svc.Endpoints[idx].RemoveAssignment()
		}
	}
	return nil
}

// updateService internal method to use when service has been validated
func (this *ControlPlaneDao) updateService(svc *service.Service) error {
	id := strings.TrimSpace(svc.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}
	svc.Id = id
	//add assignment info to service so it is availble in zk
	this.fillServiceAddr(svc)

	svcStore := service.NewStore()
	ctx := datastore.Get()

	//Deal with Service Config Files
	oldSvc, err := svcStore.Get(ctx, svc.Id)
	if err != nil {
		return err
	}
	//For now always make sure originalConfigs stay the same, essentially they are immutable
	svc.OriginalConfigs = oldSvc.OriginalConfigs

	//check if config files haven't changed
	if !reflect.DeepEqual(oldSvc.OriginalConfigs, svc.ConfigFiles) {
		//lets validate Service before doing more work....
		if err := svc.ValidEntity(); err != nil {
			return err
		}

		tenantID, servicePath, err := this.getTenantIdAndPath(svc.Id)
		if err != nil {
			return err
		}

		newConfs := make(map[string]*serviceconfigfile.SvcConfigFile)
		//config files are different, for each one that is different validate and add to newConfs
		for key, oldConf := range oldSvc.OriginalConfigs {
			if conf, found := svc.ConfigFiles[key]; found {
				if !reflect.DeepEqual(oldConf, conf) {
					newConf, err := serviceconfigfile.New(tenantID, servicePath, conf)
					if err != nil {
						return err
					}
					newConfs[key] = newConf
				}
			}
		}

		//Get current stored conf files and replace as needed
		configStore := serviceconfigfile.NewStore()
		existingConfs, err := configStore.GetConfigFiles(ctx, tenantID, servicePath)
		if err != nil {
			return err
		}
		foundConfs := make(map[string]*serviceconfigfile.SvcConfigFile)
		for _, svcConfig := range existingConfs {
			foundConfs[svcConfig.ConfFile.Filename] = svcConfig
		}
		//add or replace stored service config
		for _, newConf := range newConfs {
			if existing, found := foundConfs[newConf.ConfFile.Filename]; found {
				newConf.ID = existing.ID
				//delete it from stored confs, left overs will be deleted from DB
				delete(foundConfs, newConf.ConfFile.Filename)
			}
			configStore.Put(ctx, serviceconfigfile.Key(newConf.ID), newConf)
		}
		//remove leftover non-updated stored confs, conf was probably reverted to original or no longer exists
		for _, confToDelete := range foundConfs {
			configStore.Delete(ctx, serviceconfigfile.Key(confToDelete.ID))
		}
	}

	if err := svcStore.Put(ctx, svc); err != nil {
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
		err := store.Delete(ctx, svc.Id)
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

//getService is an internal method that returns a Service without filling in all related service data like address assignments
//and modified config files
func (this *ControlPlaneDao) getService(id string) (*service.Service, error) {
	glog.V(3).Infof("ControlPlaneDao.getService: id=%s", id)
	store := service.NewStore()
	return store.Get(datastore.Get(), id)
}

//
func (this *ControlPlaneDao) GetService(id string, myService *service.Service) error {
	glog.V(3).Infof("ControlPlaneDao.GetService: id=%s", id)
	store := service.NewStore()
	svc, err := store.Get(datastore.Get(), id)
	if err != nil {
		return err
	}
	if err = this.fillOutService(datastore.Get(), svc); err != nil {
		return err
	}
	glog.V(3).Infof("ControlPlaneDao.GetService: id=%s, service=%+v, err=%s", id, svc, err)
	*myService = *svc
	return nil
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
	if err = this.fillOutServices(datastore.Get(), results); err != nil {
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
		if err = this.fillOutServices(datastore.Get(), results); err != nil {
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
func (this *ControlPlaneDao) GetTenantId(serviceID string, tenantId *string) error {
	glog.V(2).Infof("ControlPlaneDao.GetTenantId: %s", serviceID)
	var err error
	*tenantId, _, err = this.getTenantIdAndPath(serviceID)
	return err
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
		relatedServiceIDs := walkTree(topService)
		var states []*servicestate.ServiceState
		err = this.zkDao.GetServiceStates(&states, relatedServiceIDs...)
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
						myService.Name, endpoint.Application, ss.HostIP, hostPort, protocol, containerPort)
					// if port/protocol undefined in the import, use the export's values
					if endpoint.PortNumber != 0 {
						containerPort = endpoint.PortNumber
					}
					if endpoint.Protocol != "" {
						protocol = endpoint.Protocol
					}
					var ep dao.ApplicationEndpoint
					ep.ServiceID = ss.ServiceID
					ep.ContainerPort = containerPort
					ep.HostPort = hostPort
					ep.HostIP = ss.HostIP
					ep.ContainerIP = ss.PrivateIP
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
	err := this.GetService(assignmentRequest.ServiceID, &myService)
	if err != nil {
		return err
	}

	// populate poolsIpInfo
	poolIPs, err := this.facade.GetPoolIPs(datastore.Get(), myService.PoolID)
	if err != nil {
		glog.Errorf("GetPoolsIPInfo failed: %v", err)
		return err
	}
	poolsIpInfo := poolIPs.HostIPs
	if len(poolsIpInfo) < 1 {
		msg := fmt.Sprintf("No IP addresses are available in pool %s.", myService.PoolID)
		return errors.New(msg)
	}
	glog.Infof("Pool %v contains %v available IP(s)", myService.PoolID, len(poolsIpInfo))

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
			if assignmentRequest.IPAddress == hostIPResource.IPAddress {
				// WHAT HAPPENS IF THERE EXISTS THE SAME IP ON MORE THAN ONE HOST???
				validIp = true
				ipIndex = index
				break
			}
		}

		if !validIp {
			msg := fmt.Sprintf("The requested IP address: %s is not contained in pool %s.", assignmentRequest.IPAddress, myService.PoolID)
			return errors.New(msg)
		}
	}
	assignmentRequest.IPAddress = poolsIpInfo[ipIndex].IPAddress
	selectedHostID := poolsIpInfo[ipIndex].HostID
	glog.Infof("Attempting to set IP address(es) to %s", assignmentRequest.IPAddress)

	assignments := []*addressassignment.AddressAssignment{}
	this.GetServiceAddressAssignments(assignmentRequest.ServiceID, &assignments)
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
				assignment.HostID = selectedHostID
				assignment.PoolID = myService.PoolID
				assignment.IPAddr = assignmentRequest.IPAddress
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
	err = this.walkServices(assignmentRequest.ServiceID, visitor)
	if err != nil {
		return err
	}

	glog.Infof("All services requiring an explicit IP address (at this moment) from service: %v and down ... have been assigned: %s", assignmentRequest.ServiceID, assignmentRequest.IPAddress)
	return nil
}

func (this *ControlPlaneDao) getTenantIdAndPath(serviceID string) (string, string, error) {
	id := strings.TrimSpace(serviceID)
	if id == "" {
		return "", "", errors.New("empty serviceId not allowed")
	}

	var traverse func(string) (string, error)
	svcPath := make([]string, 0)

	traverse = func(id string) (string, error) {
		if svc, err := this.getService(id); err != nil {
			return "", err
		} else if svc.ParentServiceID != "" {
			svcPath = append(svcPath, svc.Name)
			return traverse(svc.ParentServiceID)
		} else {
			glog.V(1).Infof("parent service: %+v", svc)
			svcPath = append(svcPath, svc.Name)
			return svc.Id, nil
		}
	}

	tenantId, err := traverse(id)
	if err != nil {
		return "", "", err
	}
	var svcPathName string
	for i, part := range svcPath {
		if i == 0 {
			svcPathName = part
		} else {
			svcPathName = fmt.Sprintf("%s/%s", part, svcPath)
		}
	}
	svcPathName = fmt.Sprintf("/%s", svcPathName)

	return tenantId, svcPathName, nil
}

// traverse all the services (including the children of the provided service)
func (this *ControlPlaneDao) walkServices(serviceID string, visitFn service.Visit) error {

	store := service.NewStore()
	ctx := datastore.Get()

	getChildren := func(parentID string) ([]*service.Service, error) {
		return store.GetChildServices(ctx, parentID)
	}
	getService := func(svcID string) (service.Service, error) {
		svc, err := store.Get(ctx, svcID)
		if err != nil {
			return service.Service{}, err
		}
		return *svc, nil
	}

	return service.Walk(serviceID, visitFn, getService, getChildren)
}

// walkTree returns a list of ids for all services in a hierarchy rooted by node
func walkTree(node *treenode) []string {
	if len(node.children) == 0 {
		return []string{node.id}
	}
	relatedServiceIDs := make([]string, 0)
	for _, childNode := range node.children {
		for _, childId := range walkTree(childNode) {
			relatedServiceIDs = append(relatedServiceIDs, childId)
		}
	}
	return append(relatedServiceIDs, node.id)
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
			svc.ParentServiceID,
			[]*treenode{},
		}
	}

	// second time through builds our tree
	root := treenode{"root", "", []*treenode{}}
	for _, svc := range *servicesList {
		node := servicesMap[svc.Id]
		parent, found := servicesMap[svc.ParentServiceID]
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
