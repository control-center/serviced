// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/zenoss/glog"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicestate"
)

var (
	ErrServiceExists     = errors.New("facade: service exists")
	ErrServiceNotExists  = errors.New("facade: service does not exist")
	ErrServicePathExists = errors.New("facade: service already exists at path")
	ErrServiceRunning    = errors.New("facade: service is running")
	ErrDStateUnknown     = errors.New("facade: service desired state is unknown")
	ErrBadServicePath    = errors.New("facade: bad service path")
	ErrNoParent          = errors.New("facade: child service cannot become parent")
	ErrDupEndpoints      = errors.New("facade: found duplicate endpoints on application")
)

// FilterService determines whether a service should be filtered
type FilterService func(*service.Service) bool

// FilterServiceSince filters out services updated before time since
func FilterServiceSince(since time.Time) FilterService {
	return func(svc *service.Service) bool {
		return svc.UpdatedAt.After(since)
	}
}

// FilterServiceByName filters services by name regex
func FilterServiceByName(nameRegex string) (FilterService, error) {
	r, err := regexp.Compile(nameRegex)
	if err != nil {
		glog.Errorf("Bad name regexp %s: %s", nameRegex, err)
		return nil, err
	}

	return func(svc *service.Service) bool {
		return r.MatchString(svc.Name)
	}, nil
}

// NoServiceFilter does not filter any services
var NoServiceFilter = func(*service.Service) bool { return true }

/* CRUD */

// AddService creates a new local service.  Retuns an error if the service
// already exists
func (f *Facade) AddService(ctx datastore.Context, svc service.Service, autoAssignIPs bool) error {
	glog.V(2).Infof("Facade.AddService: %+v", svc)
	store := f.serviceStore

	// check if the service exists
	if svc.ID = strings.TrimSpace(svc.ID); svc.ID != "" {
		if _, err := store.Get(ctx, svc.ID); !datastore.IsErrNoSuchEntity(err) {
			if err != nil {
				glog.Errorf("Could not look up service %s (%s): %s", svc.Name, svc.ID, err)
				return err
			} else {
				glog.Errorf("Could not add service %s (%s): %s", svc.Name, svc.ID, ErrServiceExists)
				return ErrServiceExists
			}
		}
	}

	// verify that the service can be added
	if err := f.canEditService(ctx, svc.PoolID); err != nil {
		glog.Errorf("Can not add service %s (%s) to pool %s: %s", svc.Name, svc.ID, svc.PoolID, err)
		return err
	}

	// verify the service can be added to the specified path
	if s, err := store.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name); err != nil {
		glog.Errorf("Could not verify service path for %s (%s): %s", svc.Name, svc.ID, err)
		return err
	} else if s != nil {
		glog.Errorf("Found service %s (%s) at %s", svc.Name, svc.ID, svc.ParentServiceID, ErrServicePathExists)
		return ErrServicePathExists
	}

	// Compare the incoming config files to see if there are modifications from
	// the original.  If there are, we need to perform an update to add those
	// modifications to the service. (WHY?)
	if svc.OriginalConfigs == nil {
		if svc.ConfigFiles != nil {
			svc.OriginalConfigs = svc.ConfigFiles
		} else {
			svc.OriginalConfigs = make(map[string]servicedefinition.ConfigFile)
		}
	}
	configs := svc.ConfigFiles
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)

	// strip the database version
	svc.DatabaseVersion = 0

	// set the Create/Update timestamp
	now := time.Now()
	svc.CreatedAt = now
	svc.UpdatedAt = now

	// clear the address assignments
	ipaddress := ""
	for i, ep := range svc.Endpoints {
		if svc.Endpoints[i].IsConfigurable() && ipaddress == "" && ep.AddressAssignment.IPAddr != "" {
			ipaddress = ep.AddressAssignment.IPAddr
		}
		svc.Endpoints[i].RemoveAssignment()
	}

	// TODO: this should be transactional
	if err := store.Put(ctx, &svc); err != nil {
		glog.Errorf("Could not add service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	// set the address assignment
	if autoAssignIPs {
		if err := f.AssignIPs(ctx, svc.ID, ipaddress); err != nil {
			glog.Warningf("Could not assign ip to service %s (%s): %s", svc.Name, svc.ID, err)
		}
	}

	svc.ConfigFiles = configs
	if err := f.updateService(ctx, &svc); err != nil {
		defer store.Delete(ctx, svc.ID)
		glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	return nil
}

// UpdateService updates a local service.  Returns an error if the service
// does not exist.
func (f *Facade) UpdateService(ctx datastore.Context, svc service.Service) error {
	glog.V(2).Infof("Facade.AddService: %+v", svc)
	store := f.serviceStore

	// verify that the service can be added
	if err := f.canEditService(ctx, svc.PoolID); err != nil {
		glog.Errorf("Can not add service %s (%s) to pool %s: %s", svc.Name, svc.ID, svc.PoolID, err)
		return err
	}

	// update the service
	if err := f.updateService(ctx, &svc, false); err != nil {
		glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	return nil
}

// RemoveService removes a local service.
func (f *Facade) RemoveService(ctx datastore.Context, serviceID string) error {
	glog.V(2).Infof("Facade.RemoveService: %s", serviceID)
	store := f.serviceStore

	removeService := func(svc *service.Service) error {
		// TODO: this should be transactional
		// verify the service is stopped
		if svc.DesiredState != int(service.SVCStop) {
			glog.Errorf("Could not remove service %s (%s): %s", svc.Name, svc.ID, ErrServiceRunning)
			return ErrServiceRunning
		} else {
			// check to see if there are any running instances
			var states []servicestate.ServiceState
			if err := zkAPI(f).GetServiceStates(svc.PoolID, &states, svc.ID); err != nil {
				glog.Errorf("Could not check service states for service %s (%s): %s", svc.Name, svc.ID, err)
				return err
			} else if numstates := len(states); numstates > 0 {
				glog.Errorf("Could not remove service %s (%s); found %d running instances", svc.Name, svc.ID, numstates)
				return fmt.Errorf("service has %d running instances", numstates)
			}
		}

		// remove address assignments
		if err := f.RemoveAddrAssignmentsByService(ctx, svc.ID); err != nil {
			glog.Errorf("Could not remove address assignments from service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}

		// TODO: remove service configs

		// delete the service
		if err := store.Delete(ctx, svc.ID); err != nil {
			glog.Errorf("Could not remove service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}

		// alert the coordinator
		if err := zkAPI(f).RemoveService(svc); err != nil {
			glog.Errorf("Could not remove service %s (%s) from coordinator client: %s", svc.Name, svc.ID, err)
			return err
		}
	}

	return f.walkServices(ctx, serviceID, true, removeService)
}

/* Mutation */

// SyncRemoteService synchronizes an upstream service, either by adding or
// updating it.
func (f *Facade) SyncRemoteService(ctx datastore.Context, svc service.Service) error {
	glog.V(2).Infof("Facade.SyncRemoteService: %+v", svc)

	// verify the service is linked to a governor
	gp, err := f.GetGovernedPool(ctx, svc.PoolID)
	if err != nil {
		glog.Errorf("Could not look up governed pool %s: %s", svc.PoolID, err)
		return err
	} else if gp == nil {
		return ErrGovPoolNotExists
	}

	// Set the deployment id and pool ID
	// TODO: we may want to disallow ':' in deployment ids of local applications
	svc.DeploymentID = fmt.Sprintf("%s:%s", gp.RemotePoolID, svc.DeploymentID)
	svc.PoolID = gp.PoolID

	// update the service
	if err := f.updateService(ctx, &svc, false); err != nil {
		glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	return nil
}

// ScheduleService schedules a service (and optionally its children) to change
// state.  Returns the number of affected services.
func (f *Facade) ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, state service.DesiredState) (int, error) {
	glog.V(2).Infof("Facade.ScheduleService: serviceID=%s, state=%s, autoLaunch=%s", serviceID, state, autoLaunch)

	if state.String() == "unknown" {
		return 0, ErrDStateUnknown
	} else if state != service.SVCStop {
		if err := f.canStartService(ctx, serviceID, autoLaunch); err != nil {
			glog.Errorf("Could not schedule service %s to %s: %s", serviceID, state, err)
			return 0, err
		}
	}

	affected := 0
	scheduleService := func(svc *service.Service) error {
		if svc.ID != serviceID && svc.Launch == commons.MANUAL {
			return nil
		} else if svc.DesiredState == int(state) {
			return nil
		}

		switch state {
		case service.SVCRestart:
			// shutdown all service instances
			var states []servicestate.ServiceState
			if err := zkAPI(f).GetServiceStates(svc.PoolID, &states, svc.ID); err != nil {
				return err
			}

			for _, state := range states {
				if err := zkAPI(f).StopServiceInstance(svc.PoolID, state.HostID, state.ID); err != nil {
					return err
				}
			}
			svc.DesiredState = int(service.SVCRun)
		default:
			svc.DesiredState = int(state)
		}

		if err := f.updateService(ctx, svc, false); err != nil {
			glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
		affected++
		return nil
	}

	if err := f.walkServices(ctx, serviceID, autoLaunch, scheduleService); err != nil {
		glog.Errorf("Could not schedule service(s) to start from %s: %s", serviceID, err)
		return affected, err
	}
	return affected, nil
}

// StartService schedules a service (and optionally its children) to start.
// Returns the number of affected services.
func (f *Facade) StartService(ctx datastore.Context, serviceID string, autoLaunch bool) (int, error) {
	glog.V(2).Infof("Facade.StartService: serviceID=%s, autoLaunch=%s", serviceID, autoLaunch)
	return f.ScheduleService(ctx, serviceID, autoLaunch, service.SVCRun)
}

// RestartSevice schedules a service (and optionally its children) to restart.
// Returns the number of affected services.
func (f *Facade) RestartService(ctx datastore.Context, serviceID string, autoLaunch bool) (int, error) {
	glog.V(2).Infof("Facade.RestartService: serviceID=%s, autoLaunch=%s", serviceID, autoLaunch)
	return f.ScheduleService(ctx, serviceID, autoLaunch, service.SVCRestart)
}

// PauseService schedules a service (and optionally its children) to pause.
// Returns the number of affected services.
func (f *Facade) PauseService(ctx datastore.Context, serviceID string, autoLaunch bool) (int, error) {
	glog.V(2).Infof("Facade.PauseService: serviceID=%s, autoLaunch=%s", serviceID, autoLaunch)
	return f.ScheduleService(ctx, serviceID, autoLaunch, service.SVCPause)
}

// StopService schedules a service (optionally its children) to stop.  Returns
// the number of affected services.
func (f *Facade) StopService(ctx datastore.Context, serviceID string, autoLaunch bool) (int, error) {
	glog.V(2).Infof("Facade.StopService: serviceID=%s, autoLaunch=%s", serviceID, autoLaunch)
	return f.ScheduleService(ctx, serviceID, autoLaunch, service.SVCStop)
}

/* Search */

// GetService looks up a service by its serviceID.
func (f *Facade) GetService(ctx datastore.Context, serviceID string) (*service.Service, error) {
	glog.V(2).Infof("Facade.GetService: %s", serviceID)
	store := f.serviceStore

	svc, err := store.Get(ctx, serviceID)
	if datastore.IsErrNoSuchEntity(err) {
		return nil, nil
	} else if err != nil {
		glog.Errorf("Error while looking up service %s: %s", serviceID, err)
		return nil, err
	}
	f.setServiceData(ctx, svc)
	return svc, nil
}

// GetChildService looks up a service by is parent service ID and name.
func (f *Facade) GetChildService(ctx datastore.Context, parentServiceID, childName string) (*service.Service, error) {
	glog.V(2).Infof("Facade.GetChildService: parentServiceID=%s, name=%s", parentServiceID, childName)
	store := f.serviceStore

	// TODO: need a way to handle upstream services
	parentService, err := store.Get(ctx, parentServiceID)
	if err != nil {
		glog.Errorf("Could not look up parent service %s: %s", parentServiceID, err)
		return nil, err
	}

	svc, err := store.FindChildService(ctx, parentService.DeploymentID, parentService.ID, childName)
	if datastore.IsErrNoSuchEntity(err) {
		return nil, nil
	} else if err != nil {
		glog.Errorf("Error while looking up child service %s at %s (%s): %s", childName, parentService.Name, parentService.ID, err)
		return nil, err
	}
	f.setServiceData(ctx, svc)
	return nil, nil
}

// GetAllServices returns all the services.
func (f *Facade) GetAllServices(ctx datastore.Context, filter FilterService) ([]string, []service.Service, error) {
	glog.V(2).Infof("Facade.GetAllServices")
	store := f.serviceStore

	svcs, err := store.GetServices(ctx)
	if err != nil {
		glog.Errorf("Error trying to look up all services: %s", err)
		return nil, nil, err
	}
	serviceIDs, svcs := f.filterServices(ctx, svcs, filter)
	return serviceIDs, svcs, nil
}

// GetServicesByPool returns all the services that reside in a particular
// resource pool.
func (f *Facade) GetServicesByPool(ctx datastore.Context, poolID string, filter FilterService) ([]string, []service.Service, error) {
	glog.V(2).Infof("Facade.GetServicesByPool: %s", poolID)
	store := f.serviceStore

	svcs, err := store.GetServicesByPool(ctx, poolID)
	if err != nil {
		glog.Errorf("Error trying to look up services for pool %s: %s", poolID, err)
		return nil, nil, err
	}
	serviceIDs, svcs := f.filterServices(ctx, svcs, filter)
	return serviceIDs, svcs, nil
}

// GetServicesByTenant returns all services under a particular tenant.
func (f *Facade) GetServicesByTenant(ctx datastore.Context, tenantID string, filter FilterService) ([]string, []service.Service, error) {
	glog.V(2).Infof("Facade.GetServicesByTenant: %s", tenantID)
	store := f.serviceStore

	var svcs []service.Service

	svc, err := store.Get(ctx, tenantID)
	if err != nil {
		glog.Errorf("Could not find service %s: %s", tenantID, err)
		return nil, nil, err
	}
	svcs = append(svcs, *svc)

	var getChildren func(string) ([]service.Service, error)

	getChildren = func(parentServiceID string) ([]service.Service, error) {
		childsvcs, err := store.GetChildServices(ctx, parentServiceID)
		if err != nil {
			glog.Errorf("Could not get child services of %s: %s", parentServiceID, err)
			return nil, err
		}
		var svcs []service.Service
		for _, svc := range childsvcs {
			svcs = append(svcs, svc)
			children, err := getChildren(svc.ID)
			if err != nil {
				return nil, err
			}
			svcs = append(svcs, children...)
		}
		return svcs, nil
	}

	children, err := getChildren(svc.ID)
	if err != nil {
		return nil, nil, err
	}
	svcs = append(svcs, children...)
	serviceIDs, svcs := f.filterServices(ctx, svcs, filter)
	return serviceIDs, svcs, nil
}

// GetServicesByDeployment returns all services for a particular deployment.
func (f *Facade) GetServicesByDeployment(ctx datastore.Context, deploymentID string, filter FilterService) ([]string, []service.Service, error) {
	glog.V(2).Infof("Facade.GetServicesByDeployment: %s", deploymentID)
	store := f.serviceStore

	svcs, err := store.GetServicesByDeployment(ctx, deploymentID)
	if err != nil {
		glog.Errorf("Could not look up services by deployment %s: %s", deploymentID, err)
		return nil, nil, err
	}
	serviceIDs, svcs := f.filterServices(ctx, svcs, filter)
	return serviceIDs, svcs, nil
}

// GetTaggedServices looks up a group of services by tags identified by
// name=value pairs.
func (f *Facade) GetTaggedServices(ctx datastore.Context, tags []string, filter FilterService) ([]string, []service.Service, error) {
	glog.V(2).Infof("Facade.GetTaggedServices: %s", tags)
	store := f.serviceStore

	svcs, err := store.GetTaggedServices(ctx, tags...)
	if err != nil {
		glog.Errorf("Could not look up services by tags %s: %s", tags, err)
		return nil, nil, err
	}
	serviceIDs, svcs := f.filterServices(svcs, filter)
	return serviceIDs, svcs, nil
}

// GetChildServices returns the immediate children of a parent service.
func (f *Facade) GetChildServices(ctx datastore.Context, parentServiceID string) ([]service.Service, error) {
	glog.V(2).Infof("Facade.GetChildServices: %s", parentServiceID)
	store := f.serviceStore

	svcs, err := store.GetChildServices(ctx, parentServiceID)
	if err != nil {
		glog.Errorf("Could not get child services of %s: %s", parentServiceID, err)
		return nil, err
	}
	_, svcs = f.filterServices(svcs, NoFilter)
	return svcs, nil
}

// GetTenantID returns the tenant ID of the provided service.
func (f *Facade) GetTenantID(ctx datastore.Context, serviceID string) (string, error) {
	glog.V(2).Infof("Facade.GetTenantID: %s", serviceID)
	store := f.serviceStore

	var getParent func(string) (string, error)
	getParent = func(serviceID string) (string, error) {
		svc, err := store.Get(ctx, serviceID)
		if err != nil {
			glog.Errorf("Could not look up service %s: %s", serviceID, err)
			return "", err
		}
		if svc.ParentServiceID == "" {
			return svc.ID, nil
		}
		return getParent(svc.ParentServiceID)
	}
	return getParent(serviceID)
}

// GetServicePath returns the tenantID and path to a service, starting from the
// deployment id.
func (f *Facade) GetServicePath(ctx datastore.Context, serviceID string) (string, string, error) {
	glog.V(2).Infof("Facade.GetServicePath: %s", serviceID)
	store := f.serviceStore

	var getParentPath func(string) (string, string, error)
	getParentPath = func(serviceID string) (string, string, error) {
		svc, err := store.Get(ctx, serviceID)
		if err != nil {
			glog.Errorf("Could not look up service %s: %s", serviceID, err)
			return "", "", err
		}
		if svc.ParentServiceID == "" {
			return svc.ID, path.Join(svc.DeploymentID, svc.ID), nil
		}

		t, p, err := getParent(svc.ParentServiceID)
		if err != nil {
			return "", "", nil
		} else {
			return t, path.Join(p, svc.ID), nil
		}
	}
	return getParent(serviceID)
}

/* Business logic */

// canEditService checks the pool id of a service and verifies whether it can
// added or modified.
func (f *Facade) canEditService(ctx datastore.Context, poolID string) error {
	x, err := f.getRemoteSecret(ctx, poolID)
	if err != nil {
		glog.Errorf("Could not verify resource pool %s: %s", poolID, err)
		return err
	}

	if x == "" {
		if p, err := f.GetGovernedPoolByPoolID(ctx, poolID); err != nil {
			return err
		} else if p != nil {
			return err
		}

		if p, err := f.GetResourcePool(ctx, poolID); err != nil {
			return err
		} else if p == nil {
			return ErrPoolNotExists
		}
	} else {
		if p, err := f.GetGovernedPool(ctx, svc.PoolID); err != nil {
			return err
		} else if p != nil {
			return ErrGovPoolExists
		}

		// TODO: make sure the remote pool exists; this MUST NOT return NIL
		// if it returns nil, delete the secret and return nil
	}

	return nil
}

// canStartService verifies if a service can be started
func (f *Facade) canStartService(ctx datastore.Context, serviceID string, autoLaunch bool) error {
	canStartService := func(svc *service.Service) error {
		f.setServiceData(ctx, svc)

		// make sure the applicable endpoints have address assingnments
		for _, ep := range svc.Endpoints {
			if ep.IsConfigurable() && ep.AddressAssignment.IPAddress == "" {
				glog.Errorf("Endpoint %s on service %s (%s) is missing an address assignment", ep.Name, svc.Name, svc.ID)
				return ErrMissingAddrAssign
			}
		}

		// check that vhosts aren't already started elsewhere
		for _, ep := range svc.GetServiceVHosts() {
			for _, vh := range ep.VHosts {
				if err := zkAPI(f).CheckRunningVHost(vh, svc.ID); err != nil {
					return err
				}
			}
		}

		return nil
	}
	return f.walkServices(ctx, serviceID, autoLaunch, canStartService)
}

// updateService updates a service in the datastore and sets attributes on the
// service.
func (f *Facade) updateService(ctx datastore.Context, svc *service.Service, migrate bool) error {
	store := f.serviceStore

	currentService, err := store.Get(ctx, svc)
	if err != nil {
		glog.Errorf("Could not look up service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	/* validation */

	// cannot make a child service a tenant service
	if currentService.ParentServiceID != svc.ParentServiceID && svc.ParentServiceID == "" {
		glog.Errorf("Cannot change a child service %s (%s) into a tenant", svc.Name, svc.ID)
		return ErrNoParent
	}

	// make sure the path is still valid
	// TODO: some delegate pools may have invalid parent service ids (how to fix?)
	if currentService.ParentServiceID != svc.ParentServiceID || currentService.Name != svc.Name {
		if _, err := store.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name); err != nil {
			glog.Errorf("Could not verify service path for %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
	} else if s != nil {
		glog.Errorf("Found service %s (%s) at %s", svc.Name, svc.ID, svc.ParentServiceID, ErrServicePathExists)
		return ErrServicePathExists
	}

	// make sure there are no duplicate endpoints
	var tenantID string
	if svc.ParentServiceID != "" {
		var err error
		if tenantID, err = f.GetTenantID(ctx, svc.ParentServiceID); err != nil {
			glog.Errorf("Could not find the tenant of service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
	} else {
		tenantID = svc.ID
	}
	endpointApps := make(map[string]struct{})
	if _, tenantsvcs, err := f.GetServicesByTenant(ctx, tenantID, func(svc *service.Service) bool {
		for _, ep := range svc.Endpoints {
			if ep.Purpose == "export" {
				if _, ok := endpointApps[ep.Application]; ok {
					glog.Errorf("Duplicate application detected for endpoint %s on service %s (%s)", ep.Name, svc.Name, svc.ID)
					return true
				} else {
					endpointApps[ep.Application] = struct{}{}
				}
			}
		}
		return false
	}); err != nil {
		glog.Errorf("Could not validate application endpoints for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	} else if len(tenantsvcs) > 0 {
		return ErrDupEndpoints
	}

	// set immutable values
	svc.DeploymentID = currentService.DeploymentID
	svc.OriginalConfigs = currentService.OriginalConfigs
	config, svc.ConfigFiles = svc.ConfigFiles, make(map[string]*serviceconfigfile.SvcConfigFile)
	svc.CreatedAt = currentService.CreatedAt

	// clear address assignments
	for i := range svc.Endpoints {
		ep := &svc.Endpoints[i]
		ep.RemoveAssignment()
	}
	svc.UpdatedAt = time.Now()

	// TODO: this should be transactional
	if err := store.Put(ctx, svc); err != nil {
		glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	// check the pool and update address assignments
	if currentService.PoolID != svc.PoolID {
		// check and remove address assignments
		if err := f.RemoveAddrAssignmentsByService(ctx, svc.ID); err != nil {
			glog.Warningf("Could not remove address assignments for service %s (%s): %s", currentService.Name, currentService.ID, err)
		}

		if err := zkAPI(f).RemoveService(currentService); err != nil {
			// synchronizer will eventually clean this up
			glog.Warningf("Coordinator: Could not delete service %s (%s) from pool %s: %s", currentService.Name, currentService.ID, currentService.PoolID, err)
			currentService.DesiredState = int(service.SVCStop)
			zkAPI(f).UpdateService(currentService)
		}
	}

	// update the service configs
	if config != nil {
		if migrate {
			if err := f.migrateServiceConfigs(ctx, svc, config); err != nil {
				glog.Errorf("Could not migrate configs for service %s (%s): %s", svc.Name, svc.ID, err)
				return err
			}
		} else {
			if err := f.updateServiceConfigs(ctx, svc, config); err != nil {
				glog.Errorf("Could not update configs for service %s (%s): %s", svc.Name, svc.ID, err)
				return err
			}
		}
	}

	// alert the coordinator
	f.setServiceData(ctx, svc)
	if err := zkAPI(f).UpdateService(svc); err != nil {
		glog.Errorf("Could not set service %s (%s) in coordinator client: %s", svc.Name, svc.ID, err)
		return err
	}

	return nil
}

// setServiceData updates the service object with its linked data.
func (f *Facade) setServiceData(ctx datastore.Context, svc *service.Service) {
	// set the address assignment
	if err := f.setAddressAssignment(ctx, svc); err != nil {
		glog.Warningf("Could not set service %s (%s) with its address assignment: %s", svc.Name, svc.ID, err)
	}
	// set the service configs
	if err := f.setServiceConfigs(ctx, svc); err != nil {
		glog.Warningf("Could not set service %s (%s) with its service configs: %s", svc.Name, svc.ID, err)
	}
}

// walkServices follows service (and descendents) and perform visitor function.
func (f *Facade) walkServices(ctx datastore.Context, serviceID string, traverse bool, visitFn service.Visit) error {
	store := f.serviceStore
	getChildren := func(parentServiceID string) ([]service.Service, error) {
		if !traverse {
			return []service.Service{}, nil
		}
		return store.GetChildServices(ctx, parentServiceID)
	}
	getService := func(serviceID string) (service.Service, error) {
		svc, err := store.Get(ctx, serviceID)
		if err != nil {
			return service.Service{}, err
		}
		return *svc, nil
	}
	return service.Walk(serviceID, visitFn, getService, getChildren)
}

// filterServices filters out non-matching service data
func (f *Facade) filterServices(ctx datastore.Context, allSvcs []service.Service, filter FilterService) ([]string, []service.Service) {
	serviceIDs := make([]string, len(svcs))
	var svcs []service.Service
	for i := range svcs {
		svc := svcs[i]
		serviceIDs[i] = svc.ID
		if filter(&svc) {
			f.setServiceData(&svc)
			svcs = append(svcs, svc)
		}
	}
	return serviceIDs, svcs
}
