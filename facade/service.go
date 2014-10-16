// Copyright 2014 The Serviced Authors.
// Use of f source code is governed by a

package facade

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zenoss/glog"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicestate"
)

// AddService adds a service; return error if service already exists
func (f *Facade) AddService(ctx datastore.Context, svc service.Service) error {
	glog.V(2).Infof("Facade.AddService: %+v", svc)
	store := f.serviceStore

	_, err := store.Get(ctx, svc.ID)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		return err
	} else if err == nil {
		return fmt.Errorf("error adding service; %v already exists", svc.ID)
	}
	// Strip the database version; we already know this is a create
	svc.DatabaseVersion = 0

	// Save a copy for checking configs later
	svcCopy := svc

	err = store.Put(ctx, &svc)
	if err != nil {
		glog.V(2).Infof("Facade.AddService: %+v", err)
		return err
	}
	glog.V(2).Infof("Facade.AddService: id %+v", svc.ID)

	// Compare the incoming config files to see if there are modifications from
	// the original. If there are, we need to perform an update to add those
	// modifications to the service.
	if svcCopy.OriginalConfigs != nil && !reflect.DeepEqual(svcCopy.OriginalConfigs, svcCopy.ConfigFiles) {
		// Get the current service in order to get the database version. We
		// don't save this because it won't have any of the updated config
		// files, among other things.
		cursvc, err := store.Get(ctx, svc.ID)
		if err != nil {
			glog.V(2).Infof("Facade.AddService: %+v", err)
			return err
		}
		svcCopy.DatabaseVersion = cursvc.DatabaseVersion

		for key, _ := range svcCopy.OriginalConfigs {
			glog.V(2).Infof("Facade.AddService: calling updateService for %s due to OriginalConfigs of %+v", svc.Name, key)
		}
		return f.updateService(ctx, &svcCopy)
	}

	glog.V(2).Infof("Facade.AddService: calling zk.updateService for %s %d ConfigFiles", svc.Name, len(svc.ConfigFiles))
	return zkAPI(f).UpdateService(&svc)
}

//
func (f *Facade) UpdateService(ctx datastore.Context, svc service.Service) error {
	glog.V(2).Infof("Facade.UpdateService: %+v", svc)
	//cannot update service without validating it.
	if svc.DesiredState == service.SVCRun {
		if err := f.validateServicesForStarting(ctx, &svc); err != nil {
			return err
		}

		for _, ep := range svc.GetServiceVHosts() {
			for _, vh := range ep.VHosts {
				//check that vhosts aren't already started elsewhere
				if err := zkAPI(f).CheckRunningVHost(vh, svc.ID); err != nil {
					return err
				}
			}
		}
	}
	return f.updateService(ctx, &svc)
}

//
func (f *Facade) RemoveService(ctx datastore.Context, id string) error {
	//TODO: should services already be stopped before removing to prevent half running service in case of error while deleting?

	err := f.walkServices(ctx, id, func(svc *service.Service) error {
		zkAPI(f).RemoveService(svc)
		return nil
	})

	if err != nil {
		//TODO: should we put them back?
		return err
	}

	store := f.serviceStore

	err = f.walkServices(ctx, id, func(svc *service.Service) error {
		err := store.Delete(ctx, svc.ID)
		if err != nil {
			glog.Errorf("Error removing service %s	 %s ", svc.ID, err)
		}
		return err
	})
	if err != nil {
		return err
	}
	//TODO: remove AddressAssignments with f Service
	return nil
}

func (f *Facade) GetPoolForService(ctx datastore.Context, id string) (string, error) {
	glog.V(3).Infof("Facade.GetPoolForService: id=%s", id)
	store := f.serviceStore
	svc, err := store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return svc.PoolID, nil
}

func (f *Facade) GetHealthChecksForService(ctx datastore.Context, serviceID string) (map[string]domain.HealthCheck, error) {
	glog.V(3).Infof("Facade.GetHealthChecksForService: id=%s", serviceID)
	store := f.serviceStore
	svc, err := store.Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	return svc.HealthChecks, nil
}

func (f *Facade) GetService(ctx datastore.Context, id string) (*service.Service, error) {
	glog.V(3).Infof("Facade.GetService: id=%s", id)
	store := f.serviceStore
	svc, err := store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = f.fillOutService(ctx, svc); err != nil {
		return nil, err
	}
	glog.V(3).Infof("Facade.GetService: id=%s, service=%+v, err=%s", id, svc, err)
	return svc, nil
}

//
func (f *Facade) GetServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error) {
	glog.V(3).Infof("Facade.GetServices")
	store := f.serviceStore
	var services []service.Service
	var err error
	if request.(dao.ServiceRequest).UpdatedSince != 0 {
		services, err = store.GetUpdatedServices(ctx, request.(dao.ServiceRequest).UpdatedSince)
		if err != nil {
			glog.Error("Facade.GetServices: err=", err)
			return nil, err
		}
	} else {
		services, err = store.GetServices(ctx)
		if err != nil {
			glog.Error("Facade.GetServices: err=", err)
			return nil, err
		}
	}
	if err = f.fillOutServices(ctx, services); err != nil {
		return nil, err
	}

	switch v := request.(type) {
	case dao.ServiceRequest:
		glog.V(3).Infof("request: %+v", request)

		// filter by the name provided
		if request.(dao.ServiceRequest).NameRegex != "" {
			services, err = filterByNameRegex(request.(dao.ServiceRequest).NameRegex, services)
			if err != nil {
				glog.Error("Facade.GetTaggedServices: err=", err)
				return nil, err
			}
		}

		// filter by the tenantID provided
		if request.(dao.ServiceRequest).TenantID != "" {
			services, err = f.filterByTenantID(ctx, request.(dao.ServiceRequest).TenantID, services)
			if err != nil {
				glog.Error("Facade.GetTaggedServices: err=", err)
				return nil, err
			}
		}

		return services, nil
	default:
		err := fmt.Errorf("Bad request type %v: %+v", v, request)
		glog.V(2).Info("Facade.GetTaggedServices: err=", err)
		return nil, err
	}
	return services, nil
}

// GetServicesByPool looks up all services in a particular pool
func (f *Facade) GetServicesByPool(ctx datastore.Context, poolID string) ([]service.Service, error) {
	glog.V(3).Infof("Facade.GetServicesByPool")
	store := f.serviceStore
	results, err := store.GetServicesByPool(ctx, poolID)
	if err != nil {
		glog.Error("Facade.GetServicesByPool: err=", err)
		return results, err
	}
	if err = f.fillOutServices(ctx, results); err != nil {
		return results, err
	}
	return results, nil
}

//
func (f *Facade) GetTaggedServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error) {
	glog.V(3).Infof("Facade.GetTaggedServices")

	store := f.serviceStore
	switch v := request.(type) {
	case []string:
		results, err := store.GetTaggedServices(ctx, v...)
		if err != nil {
			glog.Error("Facade.GetTaggedServices: err=", err)
			return nil, err
		}
		if err = f.fillOutServices(ctx, results); err != nil {
			return nil, err
		}
		glog.V(2).Infof("Facade.GetTaggedServices: services=%v", results)
		return results, nil
	case dao.ServiceRequest:
		glog.V(3).Infof("request: %+v", request)

		// Get the tagged services
		services, err := store.GetTaggedServices(ctx, request.(dao.ServiceRequest).Tags...)
		if err != nil {
			glog.Error("Facade.GetTaggedServices: err=", err)
			return nil, err
		}
		if err = f.fillOutServices(ctx, services); err != nil {
			return nil, err
		}

		// filter by the name provided
		if request.(dao.ServiceRequest).NameRegex != "" {
			services, err = filterByNameRegex(request.(dao.ServiceRequest).NameRegex, services)
			if err != nil {
				glog.Error("Facade.GetTaggedServices: err=", err)
				return nil, err
			}
		}

		// filter by the tenantID provided
		if request.(dao.ServiceRequest).TenantID != "" {
			services, err = f.filterByTenantID(ctx, request.(dao.ServiceRequest).TenantID, services)
			if err != nil {
				glog.Error("Facade.GetTaggedServices: err=", err)
				return nil, err
			}
		}

		return services, nil
	default:
		err := fmt.Errorf("Bad request type: %v", v)
		glog.V(2).Info("Facade.GetTaggedServices: err=", err)
		return nil, err
	}
}

// The tenant id is the root service uuid. Walk the service tree to root to find the tenant id.
func (f *Facade) GetTenantID(ctx datastore.Context, serviceID string) (string, error) {
	glog.V(2).Infof("Facade.GetTenantId: %s", serviceID)
	gs := func(id string) (service.Service, error) {
		return f.getService(ctx, id)
	}
	return getTenantID(serviceID, gs)
}

// Get a service endpoint.
func (f *Facade) GetServiceEndpoints(ctx datastore.Context, serviceId string) (map[string][]dao.ApplicationEndpoint, error) {
	// TODO: this function is obsolete.  Remove it.
	result := make(map[string][]dao.ApplicationEndpoint)
	return result, fmt.Errorf("facade.GetServiceEndpoints is obsolete - do not use it")
}

// foundchild is an error used exclusively to short-circuit the service walking
// when an appropriate child has been found
type foundchild bool

// Satisfy the error interface
func (f foundchild) Error() string {
	return ""
}

// FindChildService walks services below the service specified by serviceId, checking to see
// if childName matches the service's name. If so, it returns it.
func (f *Facade) FindChildService(ctx datastore.Context, serviceId string, childName string) (*service.Service, error) {
	var child *service.Service

	visitor := func(svc *service.Service) error {
		if svc.Name == childName {
			child = svc
			// Short-circuit the rest of the walk
			return foundchild(true)
		}
		return nil
	}
	if err := f.walkServices(ctx, serviceId, visitor); err != nil {
		// If err is a foundchild we're just short-circuiting; otherwise it's a real err, pass it on
		if _, ok := err.(foundchild); !ok {
			return nil, err
		}
	}
	return child, nil
}

// start the provided service
func (f *Facade) StartService(ctx datastore.Context, serviceId string) error {
	glog.V(4).Infof("Facade.StartService %s", serviceId)
	// f will traverse all the services
	err := f.validateService(ctx, serviceId)
	glog.V(4).Infof("Facade.StartService validate service result %v", err)
	if err != nil {
		return err
	}

	visitor := func(svc *service.Service) error {
		// don't start the service if its Launch is 'manual' and it is a child
		if svc.Launch == commons.MANUAL && svc.ID != serviceId {
			return nil
		}
		svc.DesiredState = service.SVCRun
		err = f.updateService(ctx, svc)
		glog.V(4).Infof("Facade.StartService update service %v, %v: %v", svc.Name, svc.ID, err)
		if err != nil {
			return err
		}
		return nil
	}

	// traverse all the services
	return f.walkServices(ctx, serviceId, visitor)
}

func (f *Facade) RestartService(ctx datastore.Context, serviceID string) error {
	glog.V(4).Infof("Facade.RestartService %s", serviceID)
	// f wil traverse all the services
	err := f.validateService(ctx, serviceID)
	glog.V(4).Infof("Facade.RestartService validate service result %v", err)
	if err != nil {
		return err
	}

	visitor := func(svc *service.Service) error {
		// don't restart the service if its Launch is 'manual' and it is a child
		if svc.Launch == commons.MANUAL && svc.ID != serviceID {
			return nil
		}

		var states []servicestate.ServiceState
		if err := zkAPI(f).GetServiceStates(svc.PoolID, &states, svc.ID); err != nil {
			return err
		}

		for _, state := range states {
			if err := zkAPI(f).StopServiceInstance(svc.PoolID, state.HostID, state.ID); err != nil {
				return err
			}
		}

		svc.DesiredState = service.SVCRun
		err := f.updateService(ctx, svc)
		glog.V(4).Infof("Facade.RestartService update service %v, %v: %v", svc.Name, svc.ID, err)
		return err
	}

	// traverse all the services
	return f.walkServices(ctx, serviceID, visitor)
}

// pause the provided service
func (f *Facade) PauseService(ctx datastore.Context, serviceID string) error {
	glog.V(4).Infof("Facade.PauseService %s", serviceID)

	visitor := func(svc *service.Service) error {
		svc.DesiredState = service.SVCPause
		if err := f.updateService(ctx, svc); err != nil {
			glog.Errorf("could not update service %+v due to error %s", svc, err)
			return err
		}
		glog.V(4).Infof("Facade.PauseService update service %v, %v", svc.Name, svc.ID)
		return nil
	}

	// traverse all the services
	return f.walkServices(ctx, serviceID, visitor)
}

func (f *Facade) StopService(ctx datastore.Context, id string) error {
	glog.V(0).Info("Facade.StopService id=", id)

	visitor := func(svc *service.Service) error {
		// if it's not the target service and its Launch is 'manual',
		// then do not stop it
		if svc.Launch == commons.MANUAL && svc.ID != id {
			return nil
		}
		svc.DesiredState = service.SVCStop
		if err := f.updateService(ctx, svc); err != nil {
			return err
		}
		return nil
	}

	// traverse all the services
	return f.walkServices(ctx, id, visitor)
}

type assignIPInfo struct {
	IP     string
	IPType string
	HostID string
}

func (f *Facade) retrievePoolIPs(ctx datastore.Context, poolID string) ([]assignIPInfo, error) {
	assignIPInfoSlice := []assignIPInfo{}

	poolIPs, err := f.GetPoolIPs(ctx, poolID)
	if err != nil {
		glog.Errorf("GetPoolIPs failed: %v", err)
		return assignIPInfoSlice, err
	}

	for _, hostIPResource := range poolIPs.HostIPs {
		anAssignIPInfo := assignIPInfo{IP: hostIPResource.IPAddress, IPType: commons.STATIC, HostID: hostIPResource.HostID}
		assignIPInfoSlice = append(assignIPInfoSlice, anAssignIPInfo)
	}

	for _, virtualIP := range poolIPs.VirtualIPs {
		anAssignIPInfo := assignIPInfo{IP: virtualIP.IP, IPType: commons.VIRTUAL, HostID: ""}
		assignIPInfoSlice = append(assignIPInfoSlice, anAssignIPInfo)
	}

	return assignIPInfoSlice, nil
}

// assign an IP address to a service (and all its child services) containing non default AddressResourceConfig
func (f *Facade) AssignIPs(ctx datastore.Context, assignmentRequest dao.AssignmentRequest) error {
	myService, err := f.GetService(ctx, assignmentRequest.ServiceID)
	if err != nil {
		return err
	}

	assignIPInfoSlice, err := f.retrievePoolIPs(ctx, myService.PoolID)
	if err != nil {
		return err
	} else if len(assignIPInfoSlice) < 1 {
		return fmt.Errorf("no IPs available")
	}

	rand.Seed(time.Now().UTC().UnixNano())
	assignmentHostID := ""
	assignmentType := ""

	if assignmentRequest.AutoAssignment {
		// automatic IP requested
		glog.Infof("Automatic IP Address Assignment")
		randomIPIndex := rand.Intn(len(assignIPInfoSlice))

		assignmentRequest.IPAddress = assignIPInfoSlice[randomIPIndex].IP
		assignmentType = assignIPInfoSlice[randomIPIndex].IPType
		assignmentHostID = assignIPInfoSlice[randomIPIndex].HostID

		if assignmentType == "" {
			return fmt.Errorf("Assignment type could not be determined (virtual IP was likely not in the pool)")
		}
	} else {
		// manual IP provided
		// verify that the user provided IP address is available in the pool
		glog.Infof("Manual IP Address Assignment")

		for _, anAssignIPInfo := range assignIPInfoSlice {
			if assignmentRequest.IPAddress == anAssignIPInfo.IP {
				assignmentType = anAssignIPInfo.IPType
				assignmentHostID = anAssignIPInfo.HostID
			}
		}
		if assignmentType == "" {
			// IP was NOT contained in the pool
			return fmt.Errorf("requested IP address: %s is not contained in pool %s.", assignmentRequest.IPAddress, myService.PoolID)
		}
	}

	glog.Infof("Attempting to set IP address(es) to %s", assignmentRequest.IPAddress)

	assignments := []addressassignment.AddressAssignment{}
	if err := f.GetServiceAddressAssignments(ctx, assignmentRequest.ServiceID, &assignments); err != nil {
		glog.Errorf("controlPlaneDao.GetServiceAddressAssignments failed in anonymous function: %v", err)
		return err
	}

	visitor := func(myService *service.Service) error {
		// if f service is in need of an IP address, assign it an IP address
		for _, endpoint := range myService.Endpoints {
			needsAnAddressAssignment, addressAssignmentId, err := f.needsAddressAssignment(ctx, myService.ID, endpoint)
			if err != nil {
				return err
			}

			// if an address assignment is needed (does not yet exist) OR
			// if a specific IP address is provided by the user AND an address assignment already exists
			if needsAnAddressAssignment || addressAssignmentId != "" {
				if addressAssignmentId != "" {
					glog.Infof("Removing AddressAssignment: %s", addressAssignmentId)
					err = f.RemoveAddressAssignment(ctx, addressAssignmentId)
					if err != nil {
						glog.Errorf("controlPlaneDao.RemoveAddressAssignment failed in AssignIPs anonymous function: %v", err)
						return err
					}
				}
				assignment := addressassignment.AddressAssignment{}
				assignment.AssignmentType = assignmentType
				assignment.HostID = assignmentHostID
				assignment.PoolID = myService.PoolID
				assignment.IPAddr = assignmentRequest.IPAddress
				assignment.Port = endpoint.AddressConfig.Port
				assignment.ServiceID = myService.ID
				assignment.EndpointName = endpoint.Name
				glog.Infof("Creating AddressAssignment for Endpoint: %s", assignment.EndpointName)

				var unusedStr string
				if err := f.AssignAddress(ctx, assignment, &unusedStr); err != nil {
					glog.Errorf("AssignAddress failed in AssignIPs anonymous function: %v", err)
					return err
				}

				if err := f.updateService(ctx, myService); err != nil {
					glog.Errorf("Failed to update service w/AssignAddressAssignment: %v", err)
					return err
				}

				glog.Infof("Created AddressAssignment: %s for Endpoint: %s", assignment.ID, assignment.EndpointName)
			}
		}
		return nil
	}

	// traverse all the services
	err = f.walkServices(ctx, assignmentRequest.ServiceID, visitor)
	if err != nil {
		return err
	}

	glog.Infof("All services requiring an explicit IP address (at f moment) from service: %v and down ... have been assigned: %s", assignmentRequest.ServiceID, assignmentRequest.IPAddress)
	return nil
}

func (f *Facade) filterByTenantID(ctx datastore.Context, matchTenantID string, services []service.Service) ([]service.Service, error) {
	matches := []service.Service{}
	for _, service := range services {
		localTenantID, err := f.GetTenantID(ctx, service.ID)
		if err != nil {
			return nil, err
		}

		if localTenantID == matchTenantID {
			glog.V(5).Infof("    Keeping service ID: %v (tenant ID: %v)", service.ID, localTenantID)
			matches = append(matches, service)
		}
	}
	glog.V(2).Infof("Returning %d services from tenantID: %v", len(matches), matchTenantID)
	return matches, nil
}

func filterByNameRegex(nmregex string, services []service.Service) ([]service.Service, error) {
	r, err := regexp.Compile(nmregex)
	if err != nil {
		glog.Errorf("Bad name regexp :%s", nmregex)
		return nil, err
	}

	matches := []service.Service{}
	for _, service := range services {
		if r.MatchString(service.Name) {
			glog.V(5).Infof("    Keeping service ID: %v (service name: %v)", service.ID, service.Name)
			matches = append(matches, service)
		}
	}
	glog.V(2).Infof("Returning %d services from %v", len(matches), nmregex)
	return matches, nil
}

//getService is an internal method that returns a Service without filling in all related service data like address assignments
//and modified config files
func (f *Facade) getService(ctx datastore.Context, id string) (service.Service, error) {
	glog.V(3).Infof("Facade.getService: id=%s", id)
	store := f.serviceStore
	svc, err := store.Get(datastore.Get(), id)
	if err != nil || svc == nil {
		return service.Service{}, err
	}
	return *svc, err
}

//getServices is an internal method that returns all Services without filling in all related service data like address assignments
//and modified config files
func (f *Facade) getServices(ctx datastore.Context) ([]service.Service, error) {
	glog.V(3).Infof("Facade.GetServices")
	store := f.serviceStore
	results, err := store.GetServices(ctx)
	if err != nil {
		glog.Error("Facade.GetServices: err=", err)
		return results, err
	}
	return results, nil
}

//
func (f *Facade) getTenantIDAndPath(ctx datastore.Context, svc service.Service) (string, string, error) {
	gs := func(id string) (service.Service, error) {
		return f.getService(ctx, id)
	}

	tenantID, err := f.GetTenantID(ctx, svc.ID)
	if err != nil {
		return "", "", err
	}

	path, err := svc.GetPath(gs)
	if err != nil {
		return "", "", err
	}

	return tenantID, path, err
}

// traverse all the services (including the children of the provided service)
func (f *Facade) walkServices(ctx datastore.Context, serviceID string, visitFn service.Visit) error {
	store := f.serviceStore
	getChildren := func(parentID string) ([]service.Service, error) {
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
func (f *Facade) getServiceTree(serviceId string, servicesList *[]service.Service) *treenode {
	glog.V(2).Infof(" getServiceTree = %s", serviceId)
	servicesMap := make(map[string]*treenode)
	for _, svc := range *servicesList {
		servicesMap[svc.ID] = &treenode{
			svc.ID,
			svc.ParentServiceID,
			[]*treenode{},
		}
	}

	// second time through builds our tree
	root := treenode{"root", "", []*treenode{}}
	for _, svc := range *servicesList {
		node := servicesMap[svc.ID]
		parent, found := servicesMap[svc.ParentServiceID]
		// no parent means f node belongs to root
		if !found {
			parent = &root
		}
		parent.children = append(parent.children, node)
	}

	// now walk up the tree, then back down capturing all siblings for f service ID
	topService := servicesMap[serviceId]
	for len(topService.parent) != 0 {
		topService = servicesMap[topService.parent]
	}
	return topService
}

// determine whether the services are ready for deployment
func (f *Facade) validateServicesForStarting(ctx datastore.Context, svc *service.Service) error {
	// ensure all endpoints with AddressConfig have assigned IPs
	for _, endpoint := range svc.Endpoints {
		needsAnAddressAssignment, addressAssignmentId, err := f.needsAddressAssignment(ctx, svc.ID, endpoint)
		if err != nil {
			return err
		}

		if needsAnAddressAssignment {
			return fmt.Errorf("service ID %s is in need of an AddressAssignment: %s", svc.ID, addressAssignmentId)
		} else if addressAssignmentId != "" {
			glog.Infof("AddressAssignment: %s already exists", addressAssignmentId)
		}

		if len(endpoint.VHosts) > 0 {
			//check to see if this vhost is in use by another app
		}
	}

	if svc.RAMCommitment < 0 {
		return fmt.Errorf("service RAM commitment cannot be negative")
	}

	// add additional validation checks to the services
	return nil
}

// validate the provided service
func (f *Facade) validateService(ctx datastore.Context, serviceId string) error {
	//TODO: create map of IPs to ports and ensure that an IP does not have > 1 process listening on the same port
	visitor := func(svc *service.Service) error {
		// validate the service is ready to start
		err := f.validateServicesForStarting(ctx, svc)
		if err != nil {
			glog.Errorf("services failed validation for starting")
			return err
		}
		for _, ep := range svc.GetServiceVHosts() {
			for _, vh := range ep.VHosts {
				//check that vhosts aren't already started elsewhere
				if err := zkAPI(f).CheckRunningVHost(vh, svc.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// traverse all the services
	if err := f.walkServices(ctx, serviceId, visitor); err != nil {
		glog.Errorf("unable to walk services for service %s", serviceId)
		return err
	}

	return nil
}

func (f *Facade) fillOutService(ctx datastore.Context, svc *service.Service) error {
	if err := f.fillServiceAddr(ctx, svc); err != nil {
		return err
	}
	if err := f.fillServiceConfigs(ctx, svc); err != nil {
		return err
	}
	return nil
}

func (f *Facade) fillOutServices(ctx datastore.Context, svcs []service.Service) error {
	for i := range svcs {
		if err := f.fillOutService(ctx, &svcs[i]); err != nil {
			return err
		}
	}
	return nil
}

func (f *Facade) fillServiceConfigs(ctx datastore.Context, svc *service.Service) error {
	glog.V(3).Infof("fillServiceConfigs for %s", svc.ID)
	tenantID, servicePath, err := f.getTenantIDAndPath(ctx, *svc)
	if err != nil {
		return err
	}
	glog.V(3).Infof("service %v; tenantid=%s; path=%s", svc.ID, tenantID, servicePath)

	configStore := serviceconfigfile.NewStore()
	existingConfs, err := configStore.GetConfigFiles(ctx, tenantID, servicePath)
	if err != nil {
		return err
	}

	//found confs are the modified confs for f service
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

func (f *Facade) fillServiceAddr(ctx datastore.Context, svc *service.Service) error {
	addrs, err := f.getAddressAssignments(ctx, svc.ID)
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
func (f *Facade) updateService(ctx datastore.Context, svc *service.Service) error {
	id := strings.TrimSpace(svc.ID)
	if id == "" {
		return errors.New("empty Service.ID not allowed")
	}
	svc.ID = id
	//add assignment info to service so it is availble in zk
	f.fillServiceAddr(ctx, svc)

	svcStore := f.serviceStore

	oldSvc, err := svcStore.Get(ctx, svc.ID)
	if err != nil {
		return err
	}

	//Deal with Service Config Files
	//For now always make sure originalConfigs stay the same, essentially they are immutable
	svc.OriginalConfigs = oldSvc.OriginalConfigs

	//check if config files haven't changed
	if !reflect.DeepEqual(oldSvc.OriginalConfigs, svc.ConfigFiles) {
		//lets validate Service before doing more work....
		if err := svc.ValidEntity(); err != nil {
			return err
		}

		tenantID, servicePath, err := f.getTenantIDAndPath(ctx, *svc)
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

	svc.UpdatedAt = time.Now()
	if err := svcStore.Put(ctx, svc); err != nil {
		return err
	}

	// Remove the service from zookeeper if the pool ID has changed
	err = nil
	if oldSvc.PoolID != svc.PoolID {
		err = zkAPI(f).RemoveService(oldSvc)
	}
	if err == nil {
		err = zkAPI(f).UpdateService(svc)
	}
	return err
}

func lookUpTenant(svcID string) (string, bool) {
	tenanIDMutex.RLock()
	defer tenanIDMutex.RUnlock()
	tID, found := tenantIDs[svcID]
	return tID, found
}

func updateTenants(tenantID string, svcIDs ...string) {
	tenanIDMutex.Lock()
	defer tenanIDMutex.Unlock()
	for _, id := range svcIDs {
		tenantIDs[id] = tenantID
	}
}

// getTenantID calls its GetService function to get the tenantID
func getTenantID(svcID string, gs service.GetService) (string, error) {
	if tID, found := lookUpTenant(svcID); found {
		return tID, nil
	}

	svc, err := gs(svcID)
	if err != nil {
		return "", err
	}
	visitedIDs := make([]string, 0)
	visitedIDs = append(visitedIDs, svc.ID)
	for svc.ParentServiceID != "" {
		if tID, found := lookUpTenant(svc.ParentServiceID); found {
			return tID, nil
		}
		svc, err = gs(svc.ParentServiceID)
		if err != nil {
			return "", err
		}
		visitedIDs = append(visitedIDs, svc.ID)
	}

	updateTenants(svc.ID, visitedIDs...)
	return svc.ID, nil
}

var (
	tenantIDs    = make(map[string]string)
	tenanIDMutex = sync.RWMutex{}
)
