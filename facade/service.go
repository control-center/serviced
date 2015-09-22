// Copyright 2014 The Serviced Authors.
// Use of f source code is governed by a

package facade

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
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
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/validation"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicestate"

	"github.com/control-center/serviced/commons/docker"

	"github.com/control-center/serviced/utils"

	dockerclient "github.com/fsouza/go-dockerclient"
)

const (
	// The mount point in the service migration docker image
	MIGRATION_MOUNT_POINT = "/migration"

	// The well-known path within the service's docker image of the directory which contains the service's migration script
	EMBEDDED_MIGRATION_DIRECTORY = "/opt/serviced/migration"
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

	// verify the service with parent ID does not exist with the given name
	if s, err := store.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name); err != nil {
		glog.Errorf("Could not verify service path for %s: %s", svc.Name, err)
		return err
	} else if s != nil {
		err := fmt.Errorf("service %s found at %s", svc.Name, svc.ParentServiceID)
		glog.Errorf("Cannot create service %s: %s", svc.Name, err)
		return err
	}

	// Always add services in a stopped states so in case auto ip assignment fails,
	// the scheduler won't spam the log complaining about missing address assignments.
	svc.DesiredState = int(service.SVCStop)

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
	err := f.stopServiceForUpdate(ctx, svc)
	if err != nil {
		return err
	}

	return f.updateService(ctx, &svc)
}

// TODO: Should we use a lock to serialize migration for a given service? ditto for Add and UpdateService?
func (f *Facade) RunMigrationScript(ctx datastore.Context, request dao.RunMigrationScriptRequest) error {
	svc, err := f.GetService(datastore.Get(), request.ServiceID)
	if err != nil {
		glog.Errorf("ControlPlaneDao.RunMigrationScript: could not find service id %+v: %s", request.ServiceID, err)
		return err
	}

	glog.V(2).Infof("Facade:RunMigrationScript: start for service id %+v (dry-run=%v, sdkVersion=%s)",
		svc.ID, request.DryRun, request.SDKVersion)

	var migrationDir, inputFileName, scriptFileName, outputFileName string
	migrationDir, err = createTempMigrationDir(svc.ID)
	defer os.RemoveAll(migrationDir)
	if err != nil {
		return err
	}

	svcs, err2 := f.GetServiceList(ctx, svc.ID)
	if err2 != nil {
		return err2
	}

	glog.V(3).Infof("Facade:RunMigrationScript: temp directory for service migration: %s", migrationDir)
	inputFileName, err = f.createServiceMigrationInputFile(migrationDir, svcs)
	if err != nil {
		return err
	}

	if request.ScriptBody != "" {
		scriptFileName, err = createServiceMigrationScriptFile(migrationDir, request.ScriptBody)
		if err != nil {
			return err
		}

		_, scriptFile := path.Split(scriptFileName)
		containerScript := path.Join(MIGRATION_MOUNT_POINT, scriptFile)
		outputFileName, err = executeMigrationScript(svc.ID, nil, migrationDir, containerScript, inputFileName, request.SDKVersion)
		if err != nil {
			return err
		}
	} else {
		container, err := createServiceContainer(svc)
		if err != nil {
			return err
		} else {
			defer func() {
				if err := container.Delete(true); err != nil {
					glog.Errorf("Could not remove container %s (%s): %s", container.ID, svc.ImageID, err)
				}
			}()
		}

		containerScript := path.Join(EMBEDDED_MIGRATION_DIRECTORY, request.ScriptName)
		outputFileName, err = executeMigrationScript(svc.ID, container, migrationDir, containerScript, inputFileName, request.SDKVersion)
		if err != nil {
			return err
		}
	}

	migrationRequest, err := readServiceMigrationRequestFromFile(outputFileName)
	if err != nil {
		return err
	}

	migrationRequest.DryRun = request.DryRun

	err = f.MigrateServices(ctx, *migrationRequest)

	return err

}

func (f *Facade) MigrateServices(ctx datastore.Context, request dao.ServiceMigrationRequest) error {
	var err error

	// Validate the modified services.
	for _, svc := range request.Modified {
		if err = f.verifyServiceForUpdate(ctx, svc, nil); err != nil {
			return err
		}
	}

	// Make required mutations to the added services.
	for _, svc := range request.Added {
		svc.ID, err = utils.NewUUID36()
		if err != nil {
			return err
		}
		now := time.Now()
		svc.CreatedAt = now
		svc.UpdatedAt = now
		for _, ep := range svc.Endpoints {
			ep.AddressAssignment = addressassignment.AddressAssignment{}
		}
	}

	if err = f.validateAddedMigrationServices(ctx, request.Added); err != nil {
		return err
	}

	if err = f.validateServiceDeploymentRequests(ctx, request.Deploy); err != nil {
		return err
	}

	// If this isn't a dry run, make the changes.
	if !request.DryRun {

		// Add the added services.
		for _, svc := range request.Added {
			if err = f.AddService(ctx, *svc); err != nil {
				return err
			}
		}

		// Migrate the modified services.
		for _, svc := range request.Modified {
			if err = f.stopServiceForUpdate(ctx, *svc); err != nil {
				return err
			}
			if err = f.migrateService(ctx, svc); err != nil {
				return err
			}
		}

		// Deploy the service definitions.
		for _, request := range request.Deploy {
			parent, err := f.serviceStore.Get(ctx, request.ParentID)
			if err != nil {
				glog.Errorf("Could not get parent service %s", request.ParentID)
				return err
			}
			_, err = f.DeployService(ctx, parent.PoolID, request.ParentID, false, request.Service)
			if err != nil {
				glog.Errorf("Could not deploy service definition: %+v", request.Service)
				return err
			}

		}
	}

	return nil
}

func (f *Facade) validateServiceDeploymentRequests(ctx datastore.Context, requests []*dao.ServiceDeploymentRequest) error {
	for _, request := range requests {

		// Make sure the parent exists.
		parent, err := f.serviceStore.Get(ctx, request.ParentID)
		if err != nil {
			glog.Errorf("Could not get parent service %s", request.ParentID)
			return err
		}

		// Make sure we can build the service definition into a service.
		_, err = service.BuildService(request.Service, request.ParentID, parent.PoolID, int(service.SVCStop), parent.DeploymentID)
		if err != nil {
			glog.Errorf("Could not create service: %s", err)
			return err
		}

	}

	return nil
}

func (f *Facade) validateAddedMigrationServices(ctx datastore.Context, addedSvcs []*service.Service) error {

	// Create a list of all endpoint Application names.
	existing, err := f.serviceStore.GetServices(ctx)
	if err != nil {
		return err
	}
	apps := map[string]bool{}
	for _, svc := range existing {
		for _, ep := range svc.Endpoints {
			if ep.Purpose == "export" {
				apps[ep.Application] = true
			}
		}
	}

	// Perform the validation.
	for _, svc := range addedSvcs {

		// verify the service with name and parent does not collide with another existing service
		if s, err := f.serviceStore.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name); err != nil {
			return err
		} else if s != nil {
			if s.ID != svc.ID {
				return fmt.Errorf("ValidationError: Duplicate name detected for service %s found at %s", svc.Name, svc.ParentServiceID)
			}
		}

		// Make sure no endpoint apps are duplicated.
		for _, ep := range svc.Endpoints {
			if ep.Purpose == "export" {
				if _, ok := apps[ep.Application]; ok {
					return fmt.Errorf("ValidationError: Duplicate Application detected for endpoint %s found for service %s id %s", ep.Name, svc.Name, svc.ID)
				}
			}
		}
	}

	return nil
}

func (f *Facade) RemoveService(ctx datastore.Context, id string) error {
	store := f.serviceStore

	return f.walkServices(ctx, id, true, func(svc *service.Service) error {
		// remove all address assignments
		for _, endpoint := range svc.Endpoints {
			if assignment, err := f.FindAssignmentByServiceEndpoint(ctx, svc.ID, endpoint.Name); err != nil {
				glog.Errorf("Could not find address assignment %s for service %s (%s): %s", endpoint.Name, svc.Name, svc.ID, err)
				return err
			} else if assignment != nil {
				if err := f.RemoveAddressAssignment(ctx, assignment.ID); err != nil {
					glog.Errorf("Could not remove address assignment %s from service %s (%s): %s", endpoint.Name, svc.Name, svc.ID, err)
					return err
				}
			}
			endpoint.RemoveAssignment()
		}

		if err := zkAPI(f).RemoveService(svc); err != nil {
			glog.Errorf("Could not remove service %s (%s) from zookeeper: %s", svc.Name, svc.ID, err)
			return err
		}

		if err := store.Delete(ctx, svc.ID); err != nil {
			glog.Errorf("Error while removing service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}

		return nil
	})
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

// GetImageIDs returns a list of unique IDs of all the images of all the deployed services.
func (f *Facade) GetImageIDs(ctx datastore.Context) ([]string, error) {
	store := f.serviceStore
	svcs, err := store.GetServices(ctx)
	if err != nil {
		return nil, err
	}
	var imageIDs []string
	imagemap := make(map[string]struct{})
	for _, svc := range svcs {
		if len(svc.ImageID) == 0 {
			continue
		}
		if _, ok := imagemap[svc.ImageID]; !ok {
			imageIDs = append(imageIDs, svc.ImageID)
			imagemap[svc.ImageID] = struct{}{}
		}
	}
	return imageIDs, nil
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

// GetServices looks up all services. Allows filtering by tenant ID, name (regular expression), and/or update time.
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
				glog.Error("Facade.GetServices: err=", err)
				return nil, err
			}
		}

		// filter by the tenantID provided
		if request.(dao.ServiceRequest).TenantID != "" {
			services, err = f.filterByTenantID(ctx, request.(dao.ServiceRequest).TenantID, services)
			if err != nil {
				glog.Error("Facade.GetServices: err=", err)
				return nil, err
			}
		}

		return services, nil
	default:
		err := fmt.Errorf("Bad request type %v: %+v", v, request)
		glog.V(2).Info("Facade.GetServices: err=", err)
		return nil, err
	}
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

// GetTaggedServices looks up all services with the specified tags. Allows filtering by tenant ID and/or name (regular expression).
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
	glog.V(3).Infof("Facade.GetTenantId: %s", serviceID)
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

// FindChildService walks services below the service specified by serviceId, checking to see
// if childName matches the service's name. If so, it returns it.
func (f *Facade) FindChildService(ctx datastore.Context, parentServiceID string, childName string) (*service.Service, error) {
	glog.V(3).Infof("Facade.FindChildService")
	store := f.serviceStore
	parentService, err := store.Get(ctx, parentServiceID)
	if err != nil {
		glog.Errorf("Could not look up service %s: %s", parentServiceID, err)
		return nil, err
	} else if parentService == nil {
		err := fmt.Errorf("parent does not exist")
		return nil, err
	}

	return store.FindChildService(ctx, parentService.DeploymentID, parentService.ID, childName)
}

// ScheduleService changes a service's desired state and returns the number of affected services
func (f *Facade) ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, desiredState service.DesiredState) (int, error) {
	glog.V(4).Infof("Facade.ScheduleService %s (%s)", serviceID, desiredState)

	if desiredState.String() == "unknown" {
		return 0, fmt.Errorf("desired state unknown")
	} else if desiredState != service.SVCStop {
		if err := f.validateService(ctx, serviceID, autoLaunch); err != nil {
			glog.Errorf("Facade.ScheduleService validate service result: %s", err)
			return 0, err
		}
	}

	affected := 0

	visitor := func(svc *service.Service) error {
		if svc.ID != serviceID && svc.Launch == commons.MANUAL {
			return nil
		} else if svc.DesiredState == int(desiredState) {
			return nil
		}

		switch desiredState {
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
			svc.DesiredState = int(desiredState)
		}

		if err := f.fillServiceConfigs(ctx, svc); err != nil {
			return err
		}
		if err := f.updateService(ctx, svc); err != nil {
			glog.Errorf("Facade.ScheduleService update service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
		affected++
		return nil
	}

	err := f.walkServices(ctx, serviceID, autoLaunch, visitor)
	return affected, err
}

// GetServiceStates returns all the service states given a service ID
func (f *Facade) GetServiceStates(ctx datastore.Context, serviceID string) ([]servicestate.ServiceState, error) {
	glog.V(4).Infof("Facade.GetServiceStates %s", serviceID)

	svc, err := f.GetService(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not find service %s: %s", serviceID, err)
		return nil, err
	}

	var states []servicestate.ServiceState
	if err := zkAPI(f).GetServiceStates(svc.PoolID, &states, svc.ID); err != nil {
		glog.Errorf("Could not get service states for service %s (%s): %s", svc.Name, svc.ID, err)
		return nil, err
	}

	return states, nil
}

// WaitService waits for service/s to reach a particular desired state within the designated timeout
func (f *Facade) WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, serviceIDs ...string) error {
	glog.V(4).Infof("Facade.WaitService (%s)", dstate)

	// error out if the desired state is invalid
	if dstate.String() == "unknown" {
		return fmt.Errorf("desired state unknown")
	}

	// waitstatus is the return result for the awaiting service
	type waitstatus struct {
		ServiceID string
		Err       error
	}

	cancel := make(chan interface{})
	processing := make(map[string]struct{})
	done := make(chan waitstatus)

	defer close(cancel)
	for _, serviceID := range serviceIDs {
		// spawn a goroutine to wait for each particular service
		svc, err := f.GetService(ctx, serviceID)
		if err != nil {
			glog.Errorf("Error while getting service %s: %s", serviceID, err)
			return err
		}
		processing[svc.ID] = struct{}{}
		go func(s *service.Service) {
			err := zkAPI(f).WaitService(s, dstate, cancel)
			// this blocks until we pass a waitstatus object into the channel or we get a signal to cancel
			select {
			case done <- waitstatus{s.ID, err}:
			case <-cancel:
			}
			glog.V(1).Infof("Finished waiting for %s (%s) to %s: %s", s.Name, s.ID, dstate, err)
		}(svc)
	}

	timeoutC := time.After(timeout)
	for len(processing) > 0 {
		// wait for all the services to return within the desired timeout
		select {
		case result := <-done:
			delete(processing, result.ServiceID)
			if result.Err != nil {
				glog.Errorf("Error while waiting for service %s to %s: %s", result.ServiceID, dstate, result.Err)
				return result.Err
			}
		case <-timeoutC:
			return fmt.Errorf("timeout")
		}
	}

	return nil
}

func (f *Facade) StartService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, service.SVCRun)
}

func (f *Facade) RestartService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, service.SVCRestart)
}

func (f *Facade) PauseService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, service.SVCPause)
}

func (f *Facade) StopService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, service.SVCStop)
}

type ipinfo struct {
	IP     string
	Type   string
	HostID string
}

type Ports map[uint16]struct{}

func GetPorts(endpoints []service.ServiceEndpoint) (Ports, error) {
	ports := make(map[uint16]struct{})
	for _, endpoint := range endpoints {
		if endpoint.IsConfigurable() {
			port := endpoint.AddressConfig.Port
			if _, ok := ports[port]; ok {
				return nil, fmt.Errorf("multiple endpoints found at port %d", port)
			}
			ports[port] = struct{}{}
		}
	}
	return Ports(ports), nil
}

func (m Ports) List() (ports []uint16) {
	for p := range m {
		ports = append(ports, p)
	}
	return
}

func (f *Facade) RestoreIPs(ctx datastore.Context, svc service.Service) error {
	for _, ep := range svc.Endpoints {
		if ep.AddressAssignment.IPAddr != "" {
			if assign, err := f.FindAssignmentByServiceEndpoint(ctx, svc.ID, ep.Name); err != nil {
				glog.Errorf("Could not look up address assignment %s for service %s (%s): %s", ep.Name, svc.Name, svc.ID, err)
				return err
			} else if assign == nil || !assign.EqualIP(ep.AddressAssignment) {
				ip, err := f.getManualAssignment(ctx, svc.PoolID, ep.AddressAssignment.IPAddr, ep.AddressConfig.Port)
				if err != nil {
					glog.Warningf("Could not assign ip (%s) to endpoint %s for service %s (%s): %s", ep.AddressAssignment.IPAddr, ep.Name, svc.Name, svc.ID, err)
					continue
				}

				assign = &addressassignment.AddressAssignment{
					AssignmentType: ip.Type,
					HostID:         ip.HostID,
					PoolID:         svc.PoolID,
					IPAddr:         ip.IP,
					Port:           ep.AddressConfig.Port,
					ServiceID:      svc.ID,
					EndpointName:   ep.Name,
				}
				if _, err := f.assign(ctx, *assign); err != nil {
					glog.Errorf("Could not restore address assignment for %s of service %s at %s:%d: %s", assign.EndpointName, assign.ServiceID, assign.IPAddr, assign.Port, err)
					return err
				}
				glog.Infof("Restored address assignment for endpoint %s of service %s at %s:%d", assign.EndpointName, assign.ServiceID, assign.IPAddr, assign.Port)
			} else {
				glog.Infof("Endpoint %s for service %s (%s) already assigned; skipping", assign.EndpointName, assign.ServiceID)
			}
		}
	}
	return nil
}

func (f *Facade) AssignIPs(ctx datastore.Context, request dao.AssignmentRequest) error {
	visitor := func(svc *service.Service) error {
		// get all of the ports for the service
		portmap, err := GetPorts(svc.Endpoints)
		if err != nil {
			glog.Errorf("Could not get ports for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		} else if len(portmap) == 0 {
			return nil
		}

		glog.V(1).Infof("Found %+v ports for service %s (%s)", portmap.List(), svc.Name, svc.ID)

		// get all of the address assignments for the service
		var assignments []addressassignment.AddressAssignment
		if err := f.GetServiceAddressAssignments(ctx, svc.ID, &assignments); err != nil {
			glog.Errorf("Could not get address assignments for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}

		var ip ipinfo
		if request.AutoAssignment {
			allports := portmap.List()

			// look at the current address assignments and figure out the current ip
			var ipaddr string
			for _, a := range assignments {
				if ipaddr == "" {
					ipaddr = a.IPAddr
				} else if ipaddr != a.IPAddr {
					ipaddr = ""
					break
				}
				delete(portmap, a.Port)
			}

			// try to manually assign the remaining endpoints
			if ipaddr != "" {
				ip, _ = f.getManualAssignment(ctx, svc.PoolID, ipaddr, portmap.List()...)
			}

			// if the remaining endpoints cannot be reassigned, find an ip for all endpoints
			if ip.IP == "" {
				var err error
				if ip, err = f.getAutoAssignment(ctx, svc.PoolID, allports...); err != nil {
					glog.Errorf("Could not get an ip to assign to service %s (%s): %s", svc.Name, svc.ID, err)
					return err
				}
			}
		} else {
			// look at the current address assignments and figure out which endpoints need to be reassigned
			for _, a := range assignments {
				if a.IPAddr == request.IPAddress {
					delete(portmap, a.Port)
				}
			}

			// try to find an assignment for the remaining endpoints
			var err error
			if ip, err = f.getManualAssignment(ctx, svc.PoolID, request.IPAddress, portmap.List()...); err != nil {
				glog.Errorf("Could not get an ip to assign to service %s (%s): %s", svc.Name, svc.ID, err)
				return err
			}
		}

		// remove the address assignments for all non-matching ips
		exclude := make(map[string]struct{})
		for _, assignment := range assignments {
			if assignment.IPAddr == ip.IP {
				exclude[assignment.EndpointName] = struct{}{}
			} else if err := f.RemoveAddressAssignment(ctx, assignment.ID); err != nil {
				glog.Errorf("Error removing address assignment %s for %s (%s): %s", assignment.EndpointName, svc.Name, svc.ID, err)
				return err
			}
		}

		restart := false
		for _, endpoint := range svc.Endpoints {
			if _, ok := exclude[endpoint.Name]; !ok && endpoint.IsConfigurable() {
				newassign := addressassignment.AddressAssignment{
					AssignmentType: ip.Type,
					HostID:         ip.HostID,
					PoolID:         svc.PoolID,
					IPAddr:         ip.IP,
					Port:           endpoint.AddressConfig.Port,
					ServiceID:      svc.ID,
					EndpointName:   endpoint.Name,
				}

				if _, err := f.assign(ctx, newassign); err != nil {
					glog.Errorf("Error creating address assignment for %s of service %s at %s:%d: %s", newassign.EndpointName, newassign.ServiceID, newassign.IPAddr, newassign.Port, err)
					return err
				}
				glog.Infof("Created address assignment for endpoint %s of service %s at %s:%d", newassign.EndpointName, newassign.ServiceID, newassign.IPAddr, newassign.Port)
				restart = true
			}
		}

		// Restart the service if it is running and new address assignments are made
		if restart && svc.DesiredState == int(service.SVCRun) {
			f.RestartService(ctx, dao.ScheduleServiceRequest{svc.ID, false})
		}

		return nil
	}

	// traverse all the services
	return f.walkServices(ctx, request.ServiceID, true, visitor)
}

func (f *Facade) ServiceUse(ctx datastore.Context, serviceID string, imageName string, registryName string, noOp bool) (string, error) {
	result, err := docker.ServiceUse(serviceID, imageName, registryName, noOp)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (f *Facade) getAutoAssignment(ctx datastore.Context, poolID string, ports ...uint16) (ipinfo, error) {
	pool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Error while looking up pool %s: %s", poolID, err)
		return ipinfo{}, err
	}

	ignoreips := make(map[string]struct{})
	for _, port := range ports {
		// Get all of the address assignments for port
		assignments, err := f.GetServiceAddressAssignmentsByPort(ctx, port)
		if err != nil {
			glog.Errorf("Error while looking up address assignments for port %d: %s", port, err)
			return ipinfo{}, err
		}

		// Find out all of the host ips that cannot be used
		for _, assignment := range assignments {
			ignoreips[assignment.IPAddr] = struct{}{}
		}
	}

	// Filter virtual ips
	var ips []ipinfo
	for _, vip := range pool.VirtualIPs {
		if _, ok := ignoreips[vip.IP]; !ok {
			ips = append(ips, ipinfo{vip.IP, commons.VIRTUAL, ""})
		}
	}

	hosts, err := f.FindHostsInPool(ctx, poolID)
	if err != nil {
		glog.Errorf("Error while looking up hosts in pool %s: %s", poolID, err)
		return ipinfo{}, err
	}
	var resources []host.HostIPResource
	for _, host := range hosts {
		if host.IPs != nil {
			resources = append(resources, host.IPs...)
		}
	}
	// Filter static ips
	for _, hostIP := range resources {
		if _, ok := ignoreips[hostIP.IPAddress]; !ok {
			ips = append(ips, ipinfo{hostIP.IPAddress, commons.STATIC, hostIP.HostID})
		}
	}

	// Pick an ip
	total := len(ips)
	if total == 0 {
		err := fmt.Errorf("No IPs are available to be assigned")
		glog.Errorf("Error acquiring IP assignment: %s", err)
		return ipinfo{}, err
	}

	rand.Seed(time.Now().UTC().UnixNano())
	return ips[rand.Intn(total)], nil
}

func (f *Facade) getManualAssignment(ctx datastore.Context, poolID, ipAddr string, ports ...uint16) (ipinfo, error) {
	// Check if the assignment is already there
	for _, port := range ports {
		if exists, err := f.FindAssignmentByHostPort(ctx, ipAddr, port); err != nil {
			glog.Errorf("Error while looking for assignment for (%s:%d): %s", ipAddr, port, err)
			return ipinfo{}, err
		} else if exists != nil {
			err := fmt.Errorf("assignment exists for %s:%d", ipAddr, port)
			glog.Errorf("Assignment found for endpoint %s on service %s: %s", exists.EndpointName, exists.ServiceID, err)
			return ipinfo{}, err
		}
	}

	pool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Error while looking up pool %s: %s", poolID, err)
		return ipinfo{}, err
	}

	for _, vip := range pool.VirtualIPs {
		if vip.IP == ipAddr {
			return ipinfo{vip.IP, commons.VIRTUAL, ""}, nil
		}
	}

	host, err := f.GetHostByIP(ctx, ipAddr)
	if err != nil {
		glog.Errorf("Error while looking for host with IP %s: %s", ipAddr, err)
		return ipinfo{}, err
	} else if host == nil {
		err := fmt.Errorf("host not found")
		glog.Errorf("Could not find IP %s", ipAddr)
		return ipinfo{}, err
	} else if host.PoolID != poolID {
		err := fmt.Errorf("host not found in pool")
		glog.Errorf("Host %s (%s) not found in pool %s", host.ID, ipAddr, err)
		return ipinfo{}, err
	}

	for _, hostIP := range host.IPs {
		if hostIP.IPAddress == ipAddr {
			return ipinfo{hostIP.IPAddress, commons.STATIC, hostIP.HostID}, nil
		}
	}

	// this should never happen
	return ipinfo{}, fmt.Errorf("host IP not found")
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
func (f *Facade) walkServices(ctx datastore.Context, serviceID string, traverse bool, visitFn service.Visit) error {
	store := f.serviceStore
	getChildren := func(parentID string) ([]service.Service, error) {
		if !traverse {
			return []service.Service{}, nil
		}
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
		if endpoint.IsConfigurable() {
			if assignment, err := f.FindAssignmentByServiceEndpoint(ctx, svc.ID, endpoint.Name); err != nil {
				glog.Errorf("Error looking up address assignment for endpoint %s of service %s (%s): %s", endpoint.Name, svc.Name, svc.ID, err)
				return err
			} else if assignment == nil {
				return fmt.Errorf("service %s is missing an address assignment", svc.ID)
			}
		}

		if len(endpoint.VHostList) > 0 {
			// TODO: check to see if this vhost is in use by another app
		}
	}

	// add additional validation checks to the services
	return nil
}

// validate the provided service
func (f *Facade) validateService(ctx datastore.Context, serviceId string, autoLaunch bool) error {
	//TODO: create map of IPs to ports and ensure that an IP does not have > 1 process listening on the same port
	visitor := func(svc *service.Service) error {
		// validate the service is ready to start
		err := f.validateServicesForStarting(ctx, svc)
		if err != nil {
			glog.Errorf("services failed validation for starting")
			return err
		}
		for _, ep := range svc.GetServiceVHosts() {
			for _, vh := range ep.VHostList {
				//check that vhosts aren't already started elsewhere
				if err := zkAPI(f).CheckRunningVHost(vh.Name, svc.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// traverse all the services
	if err := f.walkServices(ctx, serviceId, autoLaunch, visitor); err != nil {
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

	foundConfs, err := getExistingConfigs(ctx, tenantID, servicePath)
	if err != nil {
		return err
	}

	//replace with stored service config only if it is an existing config
	for name, conf := range foundConfs {
		if _, found := svc.ConfigFiles[name]; found {
			svc.ConfigFiles[name] = conf.ConfFile
		}
	}
	return nil
}

func (f *Facade) fillServiceAddr(ctx datastore.Context, svc *service.Service) error {
	store := addressassignment.NewStore()

	for idx := range svc.Endpoints {
		endpointName := svc.Endpoints[idx].Name
		//make sure there is no assignment; shouldn't be but just in case
		svc.Endpoints[idx].RemoveAssignment()
		//only lookup if there is a possibility for an address assignment. i.e. AddressConfig has port and protocol
		if svc.Endpoints[idx].IsConfigurable() {
			if assignment, err := f.FindAssignmentByServiceEndpoint(ctx, svc.ID, endpointName); err != nil {
				glog.Errorf("Error searching for address assignments for endpoint %s of service %s (%s): %s", endpointName, svc.Name, svc.ID, err)
				return err
			} else if assignment != nil {
				// verify the ports match
				if port := svc.Endpoints[idx].AddressConfig.Port; assignment.Port != port {
					glog.Infof("Removing address assignment for endpoint %s of service %s (%s)", endpointName, svc.Name, svc.ID)
					if err := store.Delete(ctx, addressassignment.Key(assignment.ID)); err != nil {
						glog.Errorf("Error removing address assignment for endpoint %s of service %s (%s): %s", endpointName, svc.Name, svc.ID, err)
						return err
					}
					continue
				}

				// verify the ip exists
				if exists, err := f.HasIP(ctx, svc.PoolID, assignment.IPAddr); err != nil {
					glog.Errorf("Error validating address assignment for endpoint %s of service %s (%s): %s", endpointName, svc.Name, svc.ID, err)
					return err
				} else if !exists {
					glog.Infof("Removing address assignment for endpoint %s of service %s (%s)", endpointName, svc.Name, svc.ID)
					if err := store.Delete(ctx, addressassignment.Key(assignment.ID)); err != nil {
						glog.Errorf("Error removing address assignment for endpoint %s of service %s (%s): %s", endpointName, svc.Name, svc.ID, err)
						return err
					}
					continue
				}
				svc.Endpoints[idx].SetAssignment(*assignment)
			}
		}
	}
	return nil
}

func (f *Facade) updateService(ctx datastore.Context, svc *service.Service) error {
	migrateConfiguration := false
	return f.updateServiceDefinition(ctx, migrateConfiguration, svc)
}

func (f *Facade) migrateService(ctx datastore.Context, svc *service.Service) error {
	migrateConfiguration := true
	return f.updateServiceDefinition(ctx, migrateConfiguration, svc)
}

// updateService internal method to use when service has been validated
func (f *Facade) updateServiceDefinition(ctx datastore.Context, migrateConfiguration bool, svc *service.Service) error {
	var oldSvc *service.Service
	err := f.verifyServiceForUpdate(ctx, svc, &oldSvc)
	if err != nil {
		glog.Errorf("Could not verify service %s: %s", svc.ID, err)
		return err
	}

	//add assignment info to service so it is availble in zk
	f.fillServiceAddr(ctx, svc)

	if migrateConfiguration {
		err = f.migrateServiceConfigs(ctx, oldSvc, svc)
	} else {
		err = f.updateServiceConfigs(ctx, oldSvc, svc)
	}
	if err != nil {
		return err
	}

	svc.UpdatedAt = time.Now()
	svcStore := f.serviceStore
	if err := svcStore.Put(ctx, svc); err != nil {
		return err
	}

	// Remove the service from zookeeper if the pool ID has changed
	if oldSvc.PoolID != svc.PoolID {
		if err := zkAPI(f).RemoveService(oldSvc); err != nil {
			// Synchronizer will eventually clean this service up
			glog.Warningf("ZK: Could not delete service %s (%s) from pool %s: %s", svc.Name, svc.ID, oldSvc.PoolID, err)
			oldSvc.DesiredState = int(service.SVCStop)
			zkAPI(f).UpdateService(oldSvc)
		}
	}

	return zkAPI(f).UpdateService(svc)
}

// validateServiceEndpoints traverses the service tree and checks for duplicate
// endpoints
// WARNING: This code is unused in CC 1.1, but should be added back in CC 1.2
// (see CC-811 for more information)
func (f *Facade) validateServiceEndpoints(ctx datastore.Context, svc *service.Service) error {
	epValidator := service.NewServiceEndpointValidator()
	vErr := validation.NewValidationError()

	epValidator.IsValid(vErr, svc)
	if vErr.HasError() {
		glog.Errorf("Service %s (%s) has duplicate endpoints: %s", svc.Name, svc.ID, vErr)
		return vErr
	}

	var tenantID string
	if svc.ParentServiceID == "" {
		// this service is a tenant so we don't have to traverse its tree if
		// it is a new service
		if _, err := f.serviceStore.Get(ctx, svc.ID); datastore.IsErrNoSuchEntity(err) {
			return nil
		} else if err != nil {
			glog.Errorf("Could not look up service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
		tenantID = svc.ID
	} else {
		var err error
		if tenantID, err = f.GetTenantID(ctx, svc.ParentServiceID); err != nil {
			glog.Errorf("Could not look up tenantID for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
	}

	if err := f.walkServices(ctx, tenantID, true, func(s *service.Service) error {
		// we can skip this service because we already checked it above
		if s.ID != svc.ID {
			epValidator.IsValid(vErr, s)
		}
		return nil
	}); err != nil {
		glog.Errorf("Could not walk service tree of %s (%s) with tenant %s: %s", svc.Name, svc.ID, tenantID, err)
		return err
	}
	if vErr.HasError() {
		return vErr
	}
	return nil
}

// Verify that the svc is valid for update.
// Should be called for all updated (edited), and migrated services.
// This method is only responsible for validation.
func (f *Facade) verifyServiceForUpdate(ctx datastore.Context, svc *service.Service, oldSvc **service.Service) error {
	glog.V(2).Infof("Facade:verifyServiceForUpdate: service ID %+v", svc.ID)

	id := strings.TrimSpace(svc.ID)
	if id == "" {
		return errors.New("empty Service.ID not allowed")
	}
	svc.ID = id

	// Primary service validation
	if err := svc.ValidEntity(); err != nil {
		return err
	}

	svcStore := f.serviceStore
	currentSvc, err := svcStore.Get(ctx, svc.ID)
	if err != nil {
		return err
	}

	// verify the service with name and parent does not collide with another existing service
	if s, err := svcStore.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name); err != nil {
		glog.Errorf("Could not verify service path for %s: %s", svc.Name, err)
		return err
	} else if s != nil {
		if s.ID != svc.ID {
			err := fmt.Errorf("service %s found at %s", svc.Name, svc.ParentServiceID)
			glog.Errorf("Cannot update service %s: %s", svc.Name, err)
			return err
		}
	}

	// make sure that the tenant ID and path are valid
	_, _, err = f.getTenantIDAndPath(ctx, *svc)
	if err != nil {
		return err
	}

	if oldSvc != nil {
		*oldSvc = currentSvc
	}
	return nil
}

func (f *Facade) migrateServiceConfigs(ctx datastore.Context, oldSvc, newSvc *service.Service) error {
	if reflect.DeepEqual(oldSvc.OriginalConfigs, newSvc.OriginalConfigs) {
		return nil
	}

	tenantID, servicePath, err := f.getTenantIDAndPath(ctx, *newSvc)
	if err != nil {
		return err
	}

	// addedConfs = anything in new, but not old (new config needs to be added)
	// deletedConfs = anything in old, but not new (old config needs to be removed)
	// sharedConfs = anything in both old and new versions of the service(retain existing config as is)
	addedConfs := make(map[string]*serviceconfigfile.SvcConfigFile)
	for key, newConf := range newSvc.OriginalConfigs {
		if _, found := oldSvc.OriginalConfigs[key]; !found {
			configFile, err := serviceconfigfile.New(tenantID, servicePath, newConf)
			if err != nil {
				return err
			}
			addedConfs[key] = configFile
		}
	}

	deletedConfs := make(map[string]*serviceconfigfile.SvcConfigFile)
	sharedConfs := make(map[string]*serviceconfigfile.SvcConfigFile)
	for key, oldConf := range oldSvc.OriginalConfigs {
		configFile, err := serviceconfigfile.New(tenantID, servicePath, oldConf)
		if err != nil {
			return err
		}
		if _, found := newSvc.OriginalConfigs[key]; !found {
			deletedConfs[key] = configFile
		} else {
			sharedConfs[key] = configFile
		}
	}

	existingConfs, err := getExistingConfigs(ctx, tenantID, servicePath)
	if err != nil {
		return err
	}

	glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: new configurations: %d %v\n", len(addedConfs), addedConfs)
	glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: deleted configurations: %d %v\n", len(deletedConfs), deletedConfs)
	glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: sharedConfs configurations: %d %v\n", len(sharedConfs), sharedConfs)
	glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: existing, customized configurations: %d %v\n", len(existingConfs), existingConfs)

	configStore := serviceconfigfile.NewStore()

	// sanity check - nothing in the added list should be part of the current customizations
	for _, conf := range addedConfs {
		if existing, found := existingConfs[conf.ConfFile.Filename]; found {
			glog.Warningf("Facade:migrateServiceConfigs: service ID %+v: new configuration %s found in config store", newSvc.ID, existing.ConfFile.Filename)
		}
	}

	// remove shared configurations from the list of existing configs
	for _, conf := range sharedConfs {
		if _, found := existingConfs[conf.ConfFile.Filename]; found {
			glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: keep unchanged config %s", newSvc.ID, conf.ConfFile.Filename)
			delete(existingConfs, conf.ConfFile.Filename)
		} else {
			glog.Warningf("Facade:migrateServiceConfigs: service ID %+v: unchanged configuration %s not found in config store", newSvc.ID, conf.ConfFile.Filename)
		}
	}

	// sanity check - At this point, deletedConfs and existingConfs should have the same set of filenames
	for _, conf := range deletedConfs {
		if existing, found := existingConfs[conf.ConfFile.Filename]; found {
			glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: deleting config %s", newSvc.ID, conf.ConfFile.Filename)
			delete(existingConfs, conf.ConfFile.Filename)
			err = configStore.Delete(ctx, serviceconfigfile.Key(existing.ID))
			if err != nil {
				glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: failed to delete config %s: %s", newSvc.ID, conf.ConfFile.Filename, err)
				return err
			}
		} else {
			glog.Warningf("Facade:migrateServiceConfigs: service ID %+v: obsolete configuration %s not found in config store", newSvc.ID, conf.ConfFile.Filename)
		}
	}

	// If things are working normally, existingConfs should be empty at this point, but
	//	just in case it's not, delete any remaining configurations
	for _, conf := range existingConfs {
		glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: deleting config %s", newSvc.ID, conf.ConfFile.Filename)
		err = configStore.Delete(ctx, serviceconfigfile.Key(conf.ID))
		if err != nil {
			glog.V(2).Infof("Facade:migrateServiceConfigs: service ID %+v: failed to delete config %s: %s", newSvc.ID, conf.ConfFile.Filename, err)
			return err
		}
	}

	return nil
}

func (f *Facade) updateServiceConfigs(ctx datastore.Context, oldSvc, newSvc *service.Service) error {
	//Deal with Service Config Files
	//For now always make sure originalConfigs stay the same, essentially they are immutable
	newSvc.OriginalConfigs = oldSvc.OriginalConfigs

	if err := f.fillServiceConfigs(ctx, oldSvc); err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.ConfigFiles, newSvc.ConfigFiles) {
		tenantID, servicePath, err := f.getTenantIDAndPath(ctx, *newSvc)
		if err != nil {
			return err
		}

		newConfs := make(map[string]*serviceconfigfile.SvcConfigFile)
		//config files are different, for each one that is different validate and add to newConfs
		for key, oldConf := range oldSvc.OriginalConfigs {
			if conf, found := newSvc.ConfigFiles[key]; found {
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
		foundConfs, err := getExistingConfigs(ctx, tenantID, servicePath)
		if err != nil {
			return err
		}

		//add or replace stored service config
		configStore := serviceconfigfile.NewStore()
		for _, newConf := range newConfs {
			if existing, found := foundConfs[newConf.ConfFile.Filename]; found {
				newConf.ID = existing.ID
				//delete it from stored confs, left overs will be deleted from DB
				delete(foundConfs, newConf.ConfFile.Filename)
			}
			glog.V(2).Infof("Facade:updateServiceConfigs: service ID %+v: updating config %s", newSvc.ID, newConf.ConfFile.Filename)
			configStore.Put(ctx, serviceconfigfile.Key(newConf.ID), newConf)
		}

		//remove leftover non-updated stored confs, conf was probably reverted to original or no longer exists
		for _, confToDelete := range foundConfs {
			glog.V(2).Infof("Facade:updateServiceConfigs: service ID %+v: deleting config %s", newSvc.ID, confToDelete.ConfFile.Filename)
			configStore.Delete(ctx, serviceconfigfile.Key(confToDelete.ID))
		}
	}
	return nil
}

func (f *Facade) GetServiceList(ctx datastore.Context, serviceID string) ([]*service.Service, error) {
	svcs := make([]*service.Service, 0, 1)

	err := f.walkServices(ctx, serviceID, true, func(childService *service.Service) error {
		svcs = append(svcs, childService)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error assembling list of services: %s", err)
	}

	return svcs, nil
}

func (f *Facade) stopServiceForUpdate(ctx datastore.Context, svc service.Service) error {
	//cannot update service without validating it.
	if svc.DesiredState != int(service.SVCStop) {
		if err := f.validateServicesForStarting(ctx, &svc); err != nil {
			glog.Warningf("Could not validate service %s (%s) for starting: %s", svc.Name, svc.ID, err)
			svc.DesiredState = int(service.SVCStop)
		}

		for _, ep := range svc.GetServiceVHosts() {
			for _, vh := range ep.VHostList {
				//check that vhosts aren't already started elsewhere
				if err := zkAPI(f).CheckRunningVHost(vh.Name, svc.ID); err != nil {
					return err
				}
			}
		}
	}
	return nil
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

// Returns a map based on the configuration file name of all existing configurations (modified by user or not)
func getExistingConfigs(ctx datastore.Context, tenantID, servicePath string) (map[string]*serviceconfigfile.SvcConfigFile, error) {
	configStore := serviceconfigfile.NewStore()
	confs, err := configStore.GetConfigFiles(ctx, tenantID, servicePath)
	if err != nil {
		return nil, err
	}

	// foundConfs are the existing configurations for the service
	existingConfs := make(map[string]*serviceconfigfile.SvcConfigFile)
	for _, svcConfig := range confs {
		existingConfs[svcConfig.ConfFile.Filename] = svcConfig
	}
	return existingConfs, nil
}

// Creates a temporary directory to hold files related to service migration
func createTempMigrationDir(serviceID string) (string, error) {
	tmpParentDir := utils.TempDir("service-migration")
	err := os.MkdirAll(tmpParentDir, 0750)
	if err != nil {
		return "", fmt.Errorf("Unable to create temporary directory: %s", err)
	}

	var migrationDir string
	dirPrefix := fmt.Sprintf("%s-", serviceID)
	migrationDir, err = ioutil.TempDir(tmpParentDir, dirPrefix)
	if err != nil {
		return "", fmt.Errorf("Unable to create temporary directory: %s", err)
	}

	return migrationDir, nil
}

// Write out the service definition as a JSON file for use as input to the service migration
func (f *Facade) createServiceMigrationInputFile(tmpDir string, svcs []*service.Service) (string, error) {
	inputFileName := path.Join(tmpDir, "input.json")
	jsonServices, err := json.MarshalIndent(svcs, " ", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshalling service: %s", err)
	}

	err = ioutil.WriteFile(inputFileName, jsonServices, 0440)
	if err != nil {
		return "", fmt.Errorf("error writing service to temp file: %s", err)
	}

	return inputFileName, nil
}

// Write out the body of the script to a file
func createServiceMigrationScriptFile(tmpDir, scriptBody string) (string, error) {
	scriptFileName := path.Join(tmpDir, "migrate.py")
	err := ioutil.WriteFile(scriptFileName, []byte(scriptBody), 0440)
	if err != nil {
		return "", fmt.Errorf("error writing to script file: %s", err)
	}

	return scriptFileName, nil
}

func createServiceContainer(service *service.Service) (*docker.Container, error) {
	var emptyStruct struct{}
	containerName := fmt.Sprintf("%s-%s", service.Name, "migration")
	containerDefinition := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Name: containerName,
			Config: &dockerclient.Config{
				User:       "root",
				WorkingDir: "/",
				Image:      service.ImageID,
				Volumes:    map[string]struct{}{EMBEDDED_MIGRATION_DIRECTORY: emptyStruct},
			},
		},
		dockerclient.HostConfig{},
	}

	container, err := docker.NewContainer(containerDefinition, false, 0, nil, nil)
	if err != nil {
		glog.Errorf("Error trying to create container %v: %v", containerDefinition, err)
		return nil, err
	}

	glog.V(1).Infof("Created container %s named %s based on image %s", container.ID, containerName, service.ImageID)
	return container, nil
}

// executeMigrationScript executes containerScript in a docker container based
// the service migration SDK image.
//
// tmpDir is the temporary directory that is mounted into the service migration container under
// the directory identified by MIGRATON_MOUNT_POINT. Both the input and output files are written to
// tmpDir/MIGRATION_MOUNT_POINT
//
// The value of containerScript should be always be a fully qualified, container-local path
// to the service migration script, though the path may vary depending on the value of serviceContainer.
// If serviceContainer is not specified, then containerScript should start with MIGRATON_MOUNT_POINT
// If serviceContainer is specified, then the service-migration container will be run
// with volume(s) mounted from serviceContainer. This allows for cases where containerScript physically
// resides in the serviceContainer; i.e. under the directory specified by EMBEDDED_MIGRATION_DIRECTORY
//
// Returns the name of the file under tmpDir containing the output from the migration script
func executeMigrationScript(serviceID string, serviceContainer *docker.Container, tmpDir, containerScript, inputFilePath, sdkVersion string) (string, error) {
	const SERVICE_MIGRATION_IMAGE_NAME = "zenoss/service-migration_v1"
	const SERVICE_MIGRATION_TAG_NAME = "1.0.0"
	const OUTPUT_FILE = "output.json"

	// get the container-local path names for the input and output files.
	_, inputFile := path.Split(inputFilePath)
	containerInputFile := path.Join(MIGRATION_MOUNT_POINT, inputFile)
	containerOutputFile := path.Join(MIGRATION_MOUNT_POINT, OUTPUT_FILE)

	tagName := SERVICE_MIGRATION_TAG_NAME
	if sdkVersion != "" {
		tagName = sdkVersion
	} else if tagOverride := os.Getenv("SERVICED_SERVICE_MIGRATION_TAG"); tagOverride != "" {
		tagName = tagOverride
	}

	glog.V(2).Infof("Facade:executeMigrationScript: using docker tag=%q", tagName)
	dockerImage := fmt.Sprintf("%s:%s", SERVICE_MIGRATION_IMAGE_NAME, tagName)

	mountPath := fmt.Sprintf("%s:%s:rw", tmpDir, MIGRATION_MOUNT_POINT)
	runArgs := []string{
		"run", "--rm", "-t",
		"--name", "service-migration",
		"-e", fmt.Sprintf("MIGRATE_INPUTFILE=%s", containerInputFile),
		"-e", fmt.Sprintf("MIGRATE_OUTPUTFILE=%s", containerOutputFile),
		"-v", mountPath,
	}
	if serviceContainer != nil {
		runArgs = append(runArgs, "--volumes-from", serviceContainer.ID)
	}
	runArgs = append(runArgs, dockerImage)
	runArgs = append(runArgs, "python", containerScript)

	cmd := exec.Command("docker", runArgs...)

	glog.V(2).Infof("Facade:executeMigrationScript: service ID %+v: cmd: %v", serviceID, cmd)

	cmdMessages, err := cmd.CombinedOutput()
	if exitStatus, _ := utils.GetExitStatus(err); exitStatus != 0 {
		err := fmt.Errorf("migration script failed: %s", err)
		if cmdMessages != nil {
			glog.Errorf("Service migration script for %s reported: %s", serviceID, string(cmdMessages))
		}
		return "", err
	}
	if cmdMessages != nil {
		glog.V(1).Infof("Service migration script for %s reported: %s", serviceID, string(cmdMessages))
	}

	return path.Join(tmpDir, OUTPUT_FILE), nil
}

func readServiceMigrationRequestFromFile(outputFileName string) (*dao.ServiceMigrationRequest, error) {
	data, err := ioutil.ReadFile(outputFileName)
	if err != nil {
		return nil, fmt.Errorf("could not read new service definition: %s", err)
	}

	var request dao.ServiceMigrationRequest
	if err = json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("could not unmarshall new service definition: %s", err)
	}
	return &request, nil
}
