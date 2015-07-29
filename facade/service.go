// Copyright 2014 The Serviced Authors.
// Use of f source code is governed by a

package facade

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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
	"github.com/control-center/serviced/domain/servicedefinition"
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

var (
	ErrServiceExists     = errors.New("facade: service exists")
	ErrServiceNotExists  = errors.New("facade: service does not exist")
	ErrServicePathExists = errors.New("facade: service already exists at path")
	ErrServiceRunning    = errors.New("facade: service is running")
	ErrDStateUnknown     = errors.New("facade: service desired state is unknown")
)

// ServiceHasInstances is an error that describes the number of running
// instances for a particular service
type ServiceHasInstances struct {
	ServiceID string
	Instances int
}

// Error implements error
func (err ServiceHasInstances) Error() string {
	return fmt.Sprintf("service %s has %d running instances", err.ServiceID, err.Instances)
}

// AddService creates a new service; returns an error if the service already
// exists.
func (f *Facade) AddService(ctx datastore.Context, svc service.Service, manualAssignIPs bool) error {
	glog.V(2).Infof("Facade.AddService: %+v", svc)
	store := f.serviceStore

	// check if the service exists
	if _, err := store.Get(ctx, svc.ID); !datastore.IsErrNoSuchEntity(err) {
		if err != nil {
			glog.Errorf("Could not look up service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		} else {
			glog.Errorf("Could not add service %s (%s): %s", svc.Name, svc.ID, ErrServiceExists)
			return ErrServiceExists
		}
	}

	// verify the service can be added
	if err := f.canEditService(ctx, svc.PoolID); err != nil {
		glog.Errorf("Cannot add service %s (%s) to resource pool %s: %s", svc.Name, svc.ID, svc.PoolID, err)
		return err
	}

	// verify the service can be added at the specified path
	if err := f.validateServicePath(ctx, &svc); err != nil {
		glog.Errorf("Could not validate path for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	// compare the incoming config files to see if there are modifications from
	// the original.  If there are, we need to perform an update to add those
	// modifications to the service.
	if svc.OriginalConfigs == nil || len(svc.OriginalConfigs) == 0 {
		if svc.ConfigFiles == nil || len(svc.ConfigFiles) == 0 {
			svc.OriginalConfigs = make(map[string]servicedefinition.ConfigFile)
		} else {
			svc.OriginalConfigs = svc.ConfigFiles
		}
	} else {
		if svc.ConfigFiles == nil || len(svc.ConfigFiles) == 0 {
			svc.ConfigFiles = svc.OriginalConfigs
		}
	}

	// Always add services in a stopped state so in case auto ip assignment fails,
	// the scheduler won't spam the log complaining about missing address assignments.
	svc.DesiredState = int(service.SVCStop)

	// Strip the database version; we already know this is a create
	svc.DatabaseVersion = 0

	// set the create/update timestamps
	now := time.Now()
	svc.CreatedAt = now
	svc.UpdatedAt = now

	// TODO: this should be transactional
	if err := store.Put(ctx, &svc); err != nil {
		glog.Errorf("Could not add service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	// set the address assignment.
	if !manualAssignIPs {
		// AssignIPs automatically updates the service
		if err := f.AssignIPs(ctx, svc.ID, ""); err == nil {
			return nil
		}
		glog.Warningf("Could not create an address assignment for service %s (%s): %s", svc.Name, svc.ID, err)
	}
	// update the service
	if err := f.updateService(ctx, &svc, false); err != nil {
		glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	return nil
}

// UpdateService updates an existing service; returns an error if the service
// does not exist.
func (f *Facade) UpdateService(ctx datastore.Context, svc service.Service) error {
	glog.V(2).Infof("Facade.UpdateService: %+v", svc)
	store := f.serviceStore
	// check if the service exists
	if _, err := store.Get(ctx, svc.ID); datastore.IsErrNoSuchEntity(err) {
		glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, ErrServiceNotExists)
		return ErrServiceNotExists
	} else if err != nil {
		glog.Errorf("Could not look up service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	// verify that the service can be updated
	if err := f.canEditService(ctx, svc.PoolID); err != nil {
		glog.Errorf("Could not update service %s (%s) in pool %s: %s", svc.Name, svc.ID, svc.PoolID, err)
		return err
	}
	// update the service
	if err := f.updateService(ctx, &svc, false); err != nil {
		glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	return nil
}

// RemoveService removes an existing service.
func (f *Facade) RemoveService(ctx datastore.Context, serviceID string) error {
	glog.V(2).Infof("Facade.RemoveService: %s", serviceID)
	store := f.serviceStore
	removeService := func(svc *service.Service) error {
		if svc.DesiredState != int(service.SVCStop) {
			glog.Errorf("Could not remove service %s (%s): %s", svc.Name, svc.ID, ErrServiceRunning)
			return ErrServiceRunning
		}
		// check if there are any running instances
		var states []servicestate.ServiceState
		if err := zkAPI(f).GetServiceStates(svc.PoolID, &states, svc.ID); err != nil {
			glog.Errorf("Could not check service states for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		} else if numstates := len(states); numstates > 0 {
			err := ServiceHasInstances{svc.ID, numstates}
			glog.Errorf("Could not remove service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
		// remove address assignments
		if err := f.RemoveAddrAssignmentsByService(ctx, svc.ID); err != nil {
			glog.Errorf("Could not remove address assignments from service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
		// delete the service
		if err := store.Delete(ctx, svc.ID); err != nil {
			glog.Errorf("Could not remove service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		} else if err := zkAPI(f).RemoveService(svc); err != nil {
			glog.Errorf("Could not remove service %s (%s) from coordinator: %s", svc.Name, svc.ID, err)
			return err
		}
		return nil
	}
	return f.walkServices(ctx, serviceID, true, removeService)
}

// ScheduleService schedules a service (and optionally its children) by
// changing the desired state; returns the number of affected services.
func (f *Facade) ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, state service.DesiredState) (int, error) {
	glog.V(2).Infof("Facade.ScheduleService: serviceID=%s, autoLaunch=%s, state=%s", serviceID, autoLaunch, state)
	if state.String() == "unknown" {
		return 0, ErrDStateUnknown
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
		glog.Errorf("Could not scheduled service(s) to start from %s: %s", serviceID, err)
		return affected, err
	}
	return affected, nil
}

// StartService schedules a service (and optionally its children) to start;
// returns the number of affected services.
func (f *Facade) StartService(ctx datastore.Context, serviceID string, autoLaunch bool) (int, error) {
	glog.V(2).Infof("Facade.StartService: serviceID=%s, autoLaunch=%s", serviceID, autoLaunch)
	return f.ScheduleService(ctx, serviceID, autoLaunch, service.SVCRun)
}

// RestartService schedules a service (and optionally its children) to restart;
// returns the number of affected services.
func (f *Facade) RestartService(ctx datastore.Context, serviceID string, autoLaunch bool) (int, error) {
	glog.V(2).Infof("Facade.RestartService: serviceID=%s, autoLaunch=%s", serviceID, autoLaunch)
	return f.ScheduleService(ctx, serviceID, autoLaunch, service.SVCRestart)
}

// PauseService schedules a service (and optionally its children) to pause;
// returns the number of affected services.
func (f *Facade) PauseService(ctx datastore.Context, serviceID string, autoLaunch bool) (int, error) {
	glog.V(2).Infof("Facade.PauseService: serviceID=%s, autoLaunch=%s", serviceID, autoLaunch)
	return f.ScheduleService(ctx, serviceID, autoLaunch, service.SVCPause)
}

// StopService schedules a service (and optionally its children) to stop;
// returns the number of affected services.
func (f *Facade) StopService(ctx datastore.Context, serviceID string, autoLaunch bool) (int, error) {
	glog.V(2).Infof("Facade.StopService: serviceID=%s, autoLaunch=%s", serviceID, autoLaunch)
	return f.ScheduleService(ctx, serviceID, autoLaunch, service.SVCStop)
}

// MigrateServices migrates a collection of services.
func (f *Facade) MigrateServices(ctx datastore.Context, request dao.ServiceMigrationRequest) error {
	glog.V(2).Infof("Facade.MigrateServices: %+v", request)
	// Do validations
	for _, svcMod := range request.Modified {
		if err := f.validateServiceMigrateUpdate(ctx, svcMod); err != nil {
			return err
		}
	}
	for _, svcAdd := range request.Added {
		var err error
		if svcAdd.ID, err = utils.NewUUID36(); err != nil {
			return err
		}
		if err := f.validateServiceMigrateAdd(ctx, svcAdd); err != nil {
			return err
		}
	}
	for _, deploy := range request.Deploy {
		if err := f.validateServiceMigrateDeploy(ctx, deploy.PoolID, deploy.ParentID, deploy.Service); err != nil {
			return err
		}
	}
	// Make changes if this is not a dry-run
	if !request.DryRun {
		for _, svcMod := range request.Modified {
			if err := f.updateService(ctx, svcMod, true); err != nil {
				return err
			}
		}
		for _, svcAdd := range request.Added {
			if err := f.AddService(ctx, *svcAdd, true); err != nil {
				return err
			}
		}
		for _, deploy := range request.Deploy {
			if _, err := f.DeployService(ctx, deploy.PoolID, deploy.ParentID, false, deploy.Service, true); err != nil {
				return err
			}
		}
	}
	return nil
}

// canEditService checks the pool id of a service and verifies whether it can
// be added or modified.
func (f *Facade) canEditService(ctx datastore.Context, poolID string) error {
	if p, err := f.GetResourcePool(ctx, poolID); err != nil {
		return err
	} else if p == nil {
		return ErrPoolNotExists
	}
	return nil
}

// updateService validates fields for service update and then updates the
// service.
func (f *Facade) updateService(ctx datastore.Context, svc *service.Service, migrate bool) error {
	store := f.serviceStore
	currentService, err := store.Get(ctx, svc.ID)
	if err != nil {
		glog.Errorf("Could not look up service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	// set immutables
	svc.DeploymentID = currentService.DeploymentID
	svc.CreatedAt = currentService.CreatedAt
	// set the config files
	var config map[string]servicedefinition.ConfigFile
	if migrate {
		svc.ConfigFiles, config = nil, currentService.OriginalConfigs
	} else {
		svc.OriginalConfigs, svc.ConfigFiles, config = currentService.OriginalConfigs, nil, svc.ConfigFiles
	}
	// clear address assignments from the service
	for i := range svc.Endpoints {
		svc.Endpoints[i].RemoveAssignment()
	}
	// validate service path
	if currentService.ParentServiceID != svc.ParentServiceID || currentService.Name != svc.Name {
		if err := f.validateServicePath(ctx, svc); err != nil {
			glog.Errorf("Could not validate path for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
	}
	// update the pool
	if currentService.PoolID != svc.PoolID {
		// check and remove address assignments
		if err := f.RemoveAddrAssignmentsByService(ctx, svc.ID); err != nil {
			glog.Errorf("Could not remove address assignments for service %s (%s): %s", currentService.Name, currentService.ID, err)
			return err
		}
		// remove the service from the coordinator pool
		if err := zkAPI(f).RemoveService(currentService); err != nil {
			// synchronizer will eventually clean this up
			glog.Warningf("Coordinator: Could not delete service %s (%s) from pool %s: %s", currentService.Name, currentService.ID, currentService.PoolID, err)
			currentService.DesiredState = int(service.SVCStop)
			zkAPI(f).UpdateService(currentService)
		}
	}
	// validate services for starting
	if currentService.DesiredState == int(service.SVCStop) && currentService.DesiredState != svc.DesiredState {
		if err := f.validateServiceForStarting(ctx, svc); err != nil {
			glog.Errorf("Could not validate service start for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
	}

	svc.UpdatedAt = time.Now()
	// TODO: this should be transactional
	if err := store.Put(ctx, svc); err != nil {
		glog.Errorf("Could not update service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	// set the service configs
	if config != nil {
		if err := f.updateServiceConfigs(ctx, *svc, config, migrate); err != nil {
			glog.Errorf("Could not update service configs for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
	}
	if err := f.setServiceConfigs(ctx, svc); err != nil {
		glog.Warningf("Could not set the service configs for service %s (%s): %s", svc.Name, svc.ID, err)
	}
	// set the address assignments
	if err := f.setAddrAssignment(ctx, svc); err != nil {
		glog.Warningf("Could not set the address assignments for service %s (%s): %s", svc.Name, svc.ID, err)
	}
	if err := zkAPI(f).UpdateService(svc); err != nil {
		glog.Errorf("Could not update service %s (%s) via coordinator: %s", svc.Name, svc.ID, err)
		return err
	}
	return nil
}

// validateServicePath ensures the service is added to a unique path
func (f *Facade) validateServicePath(ctx datastore.Context, svc *service.Service) error {
	store := f.serviceStore
	// verify the service can be added at the specified path
	if parentID := strings.TrimSpace(svc.ParentServiceID); parentID != "" {
		if _, err := store.Get(ctx, parentID); err != nil {
			glog.Errorf("Could not verify the existance of parent %s: %s", svc.ParentServiceID, err)
			return err
		}
	} else if s, err := store.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name); err != nil {
		glog.Errorf("Could not verify service path for %s (%s): %s", svc.Name, svc.ID, err)
		return err
	} else if s != nil && s.ID != svc.ID {
		glog.Errorf("Found service %s (%s) at %s: %s", svc.Name, svc.ID, svc.ParentServiceID, ErrServicePathExists)
		return ErrServicePathExists
	}
	return nil
}

// validateServiceForStarting ensures the service can be allowed to start
func (f *Facade) validateServiceForStarting(ctx datastore.Context, svc *service.Service) error {
	// get the address assignments for the service
	assigns, err := f.GetAddrAssignmentsByService(ctx, svc.ID)
	if err != nil {
		glog.Errorf("Could not get address assignments for service %s: %s", svc.ID, err)
		return err
	}
	assignmap := make(map[string]addressassignment.AddressAssignment)
	for _, a := range assigns {
		assignmap[a.EndpointName] = a
	}
	// validate service for starting
	for _, endpoint := range svc.Endpoints {
		// make sure the service has all of its address assignments
		if endpoint.IsConfigurable() {
			if _, ok := assignmap[endpoint.Name]; !ok {
				glog.Errorf("Service %s (%s) missing an address assignment for endpoint %s", svc.Name, svc.ID, endpoint.Name)
				return MissingAddressAssignment{svc.ID, endpoint.Name}
			}
		}
		// check that vhosts aren't already started elsewhere
		for _, vh := range endpoint.VHosts {
			if err := zkAPI(f).CheckRunningVHost(vh, svc.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateServiceMigrateDeploy verifies whether a service definition can be
// migrated.
func (f *Facade) validateServiceMigrateDeploy(ctx datastore.Context, poolID, parentID string, svcdef servicedefinition.ServiceDefinition) error {
	_, parentPath, err := f.GetServicePath(ctx, parentID)
	if err != nil {
		glog.Errorf("Could not look up parent service %s: %s", parentID, err)
		return err
	}
	deploymentID := strings.SplitN(parentPath, "/", 2)[0]

	svc, err := service.BuildService(svcdef, parentID, poolID, int(service.SVCStop), deploymentID)
	if err != nil {
		glog.Errorf("Could not build service %s: %s", svcdef.Name, err)
		return err
	}
	if err := svc.ValidEntity(); err != nil {
		glog.Errorf("Service %s of %s is not a valid entity: %s", svcdef.Name, parentPath, err)
		return err
	}
	if err := f.validateServiceMigrateAdd(ctx, svc); err != nil {
		glog.Errorf("Could not validate deployment of service %s: %s", svcdef.Name, err)
		return err
	}

	var validateChildren func(string, *service.Service, servicedefinition.ServiceDefinition) error
	validateChildren = func(parentPath string, svc *service.Service, sd servicedefinition.ServiceDefinition) error {
		children := make(map[string]struct{})
		deploymentPath := path.Join(parentPath, svc.Name)

		for _, sdef := range sd.Services {
			// build the service and verify that it is valid
			childSvc, err := service.BuildService(sdef, svc.ID, svc.PoolID, int(service.SVCStop), svc.DeploymentID)
			if err != nil {
				glog.Errorf("Could not build service %s of %s: %s", sdef.Name, deploymentPath, err)
				return err
			}
			if err := childSvc.ValidEntity(); err != nil {
				glog.Errorf("Service %s (at %s) is not a valid entity: %s", sdef.Name, deploymentPath, err)
				return err
			}
			// verify that there are no path collisions
			if _, ok := children[sd.Name]; ok {
				glog.Errorf("Could not validate child service %s of %s", sdef.Name, deploymentPath)
				return ErrServicePathExists
			} else {
				children[sd.Name] = struct{}{}
			}
			// recursively check this service's child services
			if err := validateChildren(deploymentPath, childSvc, sdef); err != nil {
				return err
			}
		}
		return nil
	}
	return validateChildren(parentPath, svc, svcdef)
}

// validateServiceMigrateAdd verifies whether a new service can be migrated.
func (f *Facade) validateServiceMigrateAdd(ctx datastore.Context, svc *service.Service) error {
	store := f.serviceStore
	if err := svc.ValidEntity(); err != nil {
		glog.Errorf("Service %s (%s) is not a valid entity: %s", svc.Name, svc.ID, err)
		return err
	} else if _, err := store.Get(ctx, svc.ID); !datastore.IsErrNoSuchEntity(err) {
		if err != nil {
			glog.Errorf("Could not look up service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		} else {
			glog.Errorf("Could not add service %s (%s): %s", svc.Name, svc.ID, ErrServiceExists)
			return ErrServiceExists
		}
	} else if err := f.canEditService(ctx, svc.PoolID); err != nil {
		glog.Errorf("Cannot add service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	} else if err := f.validateServicePath(ctx, svc); err != nil {
		glog.Errorf("Could not validate service path for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	return nil
}

// validateServiceMigrateUpdate verifies whether an existing service can be
// migrated
func (f *Facade) validateServiceMigrateUpdate(ctx datastore.Context, svc *service.Service) error {
	store := f.serviceStore
	if err := svc.ValidEntity(); err != nil {
		glog.Errorf("Service %s (%s) is not a valid entity: %s", svc.Name, svc.ID, err)
		return err
	} else if _, err := store.Get(ctx, svc.ID); err != nil {
		glog.Errorf("Could not look up service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	} else if err := f.canEditService(ctx, svc.PoolID); err != nil {
		glog.Errorf("Cannot edit service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	} else if err := f.validateServicePath(ctx, svc); err != nil {
		glog.Errorf("Could not validate service path for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	return nil
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

func (f *Facade) GetPoolForService(ctx datastore.Context, id string) (string, error) {
	glog.V(3).Infof("Facade.GetPoolForService: id=%s", id)
	store := f.serviceStore
	svc, err := store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return svc.PoolID, nil
}

// GetImages returns all the images of all the deployed services.
func (f *Facade) GetImages(ctx datastore.Context) ([]string, error) {
	store := f.serviceStore
	svcs, err := store.GetServices(ctx)
	if err != nil {
		return nil, err
	}
	var imageIDs []string
	imagemap := make(map[string]struct{})
	for _, svc := range svcs {
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

func (f *Facade) ServiceUse(ctx datastore.Context, serviceID string, imageName string, registry string, noOp bool) (string, error) {
	result, err := docker.ServiceUse(serviceID, imageName, registry, noOp)
	if err != nil {
		return "", err
	}
	return result, nil
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

// GetServicePath returns the tenantID and path to a service, starting from the
// deployment id.
func (f *Facade) GetServicePath(ctx datastore.Context, serviceID string) (string, string, error) {
	glog.V(2).Infof("Facade.GetServicePath: %s", serviceID)
	store := f.serviceStore

	var getParentPath func(string) (string, string, error)
	getParentPath = func(string) (string, string, error) {
		svc, err := store.Get(ctx, serviceID)
		if err != nil {
			glog.Errorf("Could not look up service %s: %s", serviceID, err)
			return "", "", err
		}
		if svc.ParentServiceID == "" {
			return svc.ID, path.Join(svc.DeploymentID, svc.ID), nil
		}

		t, p, err := getParentPath(svc.ParentServiceID)
		if err != nil {
			return "", "", err
		}
		return t, path.Join(p, svc.ID), nil
	}
	return getParentPath(serviceID)
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

func (f *Facade) fillOutService(ctx datastore.Context, svc *service.Service) error {
	if err := f.setAddrAssignment(ctx, svc); err != nil {
		return err
	} else if err := f.setServiceConfigs(ctx, svc); err != nil {
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
