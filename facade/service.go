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
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/zenoss/glog"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/validation"
	zkservice "github.com/control-center/serviced/zzk/service"

	"github.com/control-center/serviced/domain/service"

	"github.com/control-center/serviced/utils"
)

const (
	// The mount point in the service migration docker image
	MIGRATION_MOUNT_POINT = "/migration"

	// The well-known path within the service's docker image of the directory which contains the service's migration script
	EMBEDDED_MIGRATION_DIRECTORY = "/opt/serviced/migration"
)

var (
	ErrServiceExists            = errors.New("facade: service exists")
	ErrServiceDoesNotExist      = errors.New("facade: service does not exist")
	ErrServiceCollision         = errors.New("facade: service name already exists under parent")
	ErrTenantDoesNotMatch       = errors.New("facade: service tenants do not match")
	ErrServiceMissingAssignment = errors.New("facade: service is missing an address assignment")
	ErrServiceDuplicateEndpoint = errors.New("facade: duplicate endpoint found")
)

// AddService adds a service; return error if service already exists
func (f *Facade) AddService(ctx datastore.Context, svc service.Service) (err error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddService"))
	var tenantID string
	if svc.ParentServiceID == "" {
		tenantID = svc.ID
	} else if tenantID, err = f.GetTenantID(ctx, svc.ParentServiceID); err != nil {
		return err
	}
	mutex := getTenantLock(tenantID)
	mutex.RLock()
	defer mutex.RUnlock()
	return f.addService(ctx, tenantID, svc, false)
}

func (f *Facade) addService(ctx datastore.Context, tenantID string, svc service.Service, setLockOnCreate bool) error {
	store := f.serviceStore
	// service add validation
	if err := f.validateServiceAdd(ctx, &svc); err != nil {
		glog.Errorf("Could not validate service %s (%s) for adding: %s", svc.Name, svc.ID, err)
		return err
	}
	var configFiles []servicedefinition.ConfigFile
	if svc.ConfigFiles == nil || len(svc.ConfigFiles) == 0 {
		for _, configFile := range svc.OriginalConfigs {
			configFiles = append(configFiles, configFile)
		}
	} else {
		for _, configFile := range svc.ConfigFiles {
			configFiles = append(configFiles, configFile)
		}
	}
	svc.ConfigFiles = nil
	// write the service into the database
	svc.UpdatedAt = time.Now()
	svc.CreatedAt = svc.UpdatedAt
	if err := store.Put(ctx, &svc); err != nil {
		glog.Errorf("Could not create service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	glog.Infof("Created service %s (%s)", svc.Name, svc.ID)
	// add the service configurations to the database
	if err := f.updateServiceConfigs(ctx, svc.ID, configFiles, true); err != nil {
		glog.Warningf("Could not set configurations to service %s (%s): %s", svc.Name, svc.ID, err)
	}
	glog.Infof("Set configuration information for service %s (%s)", svc.Name, svc.ID)
	// sync the service with the coordinator
	if err := f.syncService(ctx, tenantID, svc.ID, setLockOnCreate, setLockOnCreate); err != nil {
		glog.Errorf("Could not sync service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	glog.Infof("Synced service %s (%s) to the coordinator", svc.Name, svc.ID)
	return nil
}

func (f *Facade) validateServiceAdd(ctx datastore.Context, svc *service.Service) error {
	store := f.serviceStore
	// verify that the service does not already exist
	if _, err := store.Get(ctx, svc.ID); !datastore.IsErrNoSuchEntity(err) {
		if err != nil {
			glog.Errorf("Could not check the existance of service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		} else {
			glog.Errorf("Could not add service %s (%s): %s", svc.Name, svc.ID, ErrServiceExists)
			return ErrServiceExists
		}
	}
	// verify no collision with the service name
	if err := f.validateServiceName(ctx, svc); err != nil {
		glog.Errorf("Could not add service %s to parent %s: %s", svc.Name, svc.ParentServiceID, err)
		return err
	}

	// disable ports and vhosts that are already in use by another application
	for i, ep := range svc.Endpoints {
		for j, vhost := range ep.VHostList {
			if vhost.Enabled {
				serviceID, application, err := f.zzk.GetVHost(vhost.Name)
				if err != nil {
					glog.Errorf("Could not check public endpoint for virtual host %s: %s", vhost.Name, err)
					return err
				}
				if serviceID != "" || application != "" {
					glog.Warningf("VHost %s already in use by another application %s (%s)", vhost.Name, serviceID, application)
					svc.Endpoints[i].VHostList[j].Enabled = false
				}
			}
		}

		for j, port := range ep.PortList {
			if port.Enabled {
				serviceID, application, err := f.zzk.GetPublicPort(port.PortAddr)
				if err != nil {
					glog.Errorf("Could not check public endpoint for port %s: %s", port.PortAddr, err)
					return err
				}
				if serviceID != "" || application != "" {
					glog.Warningf("Public port %s already in use by another application %s (%s)", port.PortAddr, serviceID, application)
					svc.Endpoints[i].PortList[j].Enabled = false
				}
			}
		}
	}

	// set service defaults
	svc.DesiredState = int(service.SVCStop) // new services must always be stopped
	svc.DatabaseVersion = 0                 // create service set database version to 0
	// manage service configurations
	if svc.OriginalConfigs == nil || len(svc.OriginalConfigs) == 0 {
		if svc.ConfigFiles != nil {
			svc.OriginalConfigs = svc.ConfigFiles
		} else {
			svc.OriginalConfigs = make(map[string]servicedefinition.ConfigFile)
		}
	}
	return nil
}

// UpdateService updates an existing service; return error if the service does
// not exist.
func (f *Facade) UpdateService(ctx datastore.Context, svc service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("UpdateService"))
	tenantID, err := f.GetTenantID(ctx, svc.ID)
	if err != nil {
		return err
	}
	mutex := getTenantLock(tenantID)
	mutex.RLock()
	defer mutex.RUnlock()
	return f.updateService(ctx, tenantID, svc, false, false)
}

// MigrateService migrates an existing service; return error if the service does
// not exist
func (f *Facade) MigrateService(ctx datastore.Context, svc service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("MigrateService"))
	tenantID, err := f.GetTenantID(ctx, svc.ID)
	if err != nil {
		return err
	}
	mutex := getTenantLock(tenantID)
	mutex.RLock()
	defer mutex.RUnlock()
	return f.updateService(ctx, tenantID, svc, true, false)
}

func (f *Facade) updateService(ctx datastore.Context, tenantID string, svc service.Service, migrate, setLockOnUpdate bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("updateService"))
	store := f.serviceStore
	cursvc, err := f.validateServiceUpdate(ctx, &svc)
	if err != nil {
		glog.Errorf("Could not validate service %s (%s) for update: %s", svc.Name, svc.ID, err)
		return err
	}
	// set service configurations
	if migrate {
		if svc.OriginalConfigs == nil || len(svc.OriginalConfigs) == 0 {
			if svc.ConfigFiles != nil {
				svc.OriginalConfigs = svc.ConfigFiles
			} else {
				svc.OriginalConfigs = make(map[string]servicedefinition.ConfigFile)
			}
		}
	} else {
		svc.OriginalConfigs = cursvc.OriginalConfigs
	}
	var configFiles []servicedefinition.ConfigFile
	if svc.ConfigFiles != nil {
		for _, configFile := range svc.ConfigFiles {
			configFiles = append(configFiles, configFile)
		}
		svc.ConfigFiles = nil
	}
	// write the service into the database
	svc.UpdatedAt = time.Now()
	if err := store.Put(ctx, &svc); err != nil {
		glog.Errorf("Could not create service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	glog.Infof("Updated service %s (%s)", svc.Name, svc.ID)

	if err := f.updateServiceConfigs(ctx, svc.ID, configFiles, true); err != nil {
		glog.Warningf("Could not set configurations to service %s (%s): %s", svc.Name, svc.ID, err)
	}
	glog.Infof("Set configuration information for service %s (%s)", svc.Name, svc.ID)

	// remove the service from coordinator if the pool has changed
	if cursvc.PoolID != svc.PoolID {
		if err := f.zzk.RemoveService(cursvc.PoolID, cursvc.ID); err != nil {
			// synchronizer will eventually clean this service up
			glog.Warningf("COORD: Could not delete service %s from pool %s: %s", cursvc.ID, cursvc.PoolID, err)
			cursvc.DesiredState = int(service.SVCStop)
			f.zzk.UpdateService(tenantID, cursvc, false, false)
		}
	}

	// sync the service with the coordinator
	if err := f.syncService(ctx, tenantID, svc.ID, setLockOnUpdate, setLockOnUpdate); err != nil {
		glog.Errorf("Could not sync service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	glog.Infof("Synced service %s (%s) to the coordinator", svc.Name, svc.ID)
	return nil
}

func (f *Facade) validateServiceUpdate(ctx datastore.Context, svc *service.Service) (*service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("validateServiceUpdate"))
	store := f.serviceStore
	// verify that the service exists
	cursvc, err := store.Get(ctx, svc.ID)
	if err != nil {
		glog.Errorf("Could not load service %s (%s) from database: %s", svc.Name, svc.ID, err)
		return nil, err
	}
	// verify no collision with the service name
	if svc.ParentServiceID != cursvc.ParentServiceID || svc.Name != cursvc.Name {
		// if the parent changed, make sure it shares the same tenant
		if svc.ParentServiceID != cursvc.ParentServiceID {
			if err := f.validateServiceTenant(ctx, svc.ParentServiceID, svc.ID); err != nil {
				glog.Errorf("Could not validate tenant for updated service %s: %s", svc.ID, err)
				return nil, err
			}
		}
		if err := f.validateServiceName(ctx, svc); err != nil {
			glog.Errorf("Could not validate service name for updated service %s: %s", svc.ID, err)
			return nil, err
		}

		// If the service has been reparented, we need to clear it from the cache
		f.serviceCache.RemoveIfParentChanged(svc.ID, svc.ParentServiceID)
	}

	// disallow enabling ports and vhosts that are already enabled by a different
	// service and application.
	// TODO: what if they are on the same service?
	for _, ep := range svc.Endpoints {
		for _, vhost := range ep.VHostList {
			if vhost.Enabled {
				serviceID, application, err := f.zzk.GetVHost(vhost.Name)
				if err != nil {
					glog.Errorf("Could not check public endpoint for virtual host %s: %s", vhost.Name, err)
					return nil, err
				}
				if (serviceID != "" && serviceID != svc.ID) || (application != "" && application != ep.Application) {
					glog.Errorf("VHost %s already in use by another application %s (%s)", vhost.Name, serviceID, application)
					return nil, fmt.Errorf("vhost %s is already in use", vhost.Name)
				}
			}
		}

		for _, port := range ep.PortList {
			if port.Enabled {
				serviceID, application, err := f.zzk.GetPublicPort(port.PortAddr)
				if err != nil {
					glog.Errorf("Could not check public endpoint for port %s: %s", port.PortAddr, err)
					return nil, err
				}
				if (serviceID != "" && serviceID != svc.ID) || (application != "" && application != ep.Application) {
					glog.Errorf("Public port %s already in use by another application %s (%s)", port.PortAddr, serviceID, application)
					return nil, fmt.Errorf("port %s is already in use", port.PortAddr)
				}
			}
		}
	}

	// set read-only fields
	svc.CreatedAt = cursvc.CreatedAt
	svc.DeploymentID = cursvc.DeploymentID
	// verify the desired state of the service
	if svc.DesiredState != int(service.SVCStop) {
		if err := f.validateServiceStart(ctx, svc); err != nil {
			glog.Warningf("Could not validate %s for starting: %s", svc.ID, err)
			svc.DesiredState = int(service.SVCStop)
		}
	}
	return cursvc, nil
}

// validateServiceName ensures that the service does not collide with a
// service at the same path
func (f *Facade) validateServiceName(ctx datastore.Context, svc *service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("validateServiceName"))
	store := f.serviceStore
	if svc.ParentServiceID != "" {
		psvc, err := store.Get(ctx, svc.ParentServiceID)
		if err != nil {
			glog.Errorf("Could not verify the existance of parent %s for service %s: %s", svc.ParentServiceID, svc.Name, err)
			return err
		}
		svc.DeploymentID = psvc.DeploymentID
	}
	cursvc, err := store.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name)
	if err != nil {
		glog.Errorf("Could not check the service name %s for parent %s: %s", svc.Name, svc.ParentServiceID, err)
		return err
	}
	if cursvc != nil {
		return ErrServiceCollision
	}
	return nil
}

// validateServiceTenant ensures the services are on the same tenant
func (f *Facade) validateServiceTenant(ctx datastore.Context, serviceA, serviceB string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("validateServiceTenant"))
	if serviceA == "" || serviceB == "" {
		return ErrTenantDoesNotMatch
	}
	tenantA, err := f.GetTenantID(ctx, serviceA)
	if err != nil {
		glog.Errorf("Could not look up tenant for service %s: %s", serviceA, err)
		return err
	}
	tenantB, err := f.GetTenantID(ctx, serviceB)
	if err != nil {
		glog.Errorf("Could not look up tenant for service %s: %s", serviceB, err)
		return err
	}
	if tenantA != tenantB {
		return ErrTenantDoesNotMatch
	}
	return nil
}

// validateServiceStart determines whether the service can actually be set to
// start.
func (f *Facade) validateServiceStart(ctx datastore.Context, svc *service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("validateServiceStart"))
	// ensure that all endpoints are available
	for _, ep := range svc.Endpoints {
		if ep.IsConfigurable() {
			as, err := f.FindAssignmentByServiceEndpoint(ctx, svc.ID, ep.Name)
			if err != nil {
				glog.Errorf("Could not look up assignment %s for service %s: %s", ep.Name, svc.ID, err)
				return err
			}
			if as == nil {
				return ErrServiceMissingAssignment
			}
		}
	}
	return nil
}

// syncService syncs service data from the database into the coordinator.
func (f *Facade) syncService(ctx datastore.Context, tenantID, serviceID string, setLockOnCreate, setLockOnUpdate bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("syncService"))
	svc, err := f.GetService(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not get service %s to sync: %s", serviceID, err)
		return err
	}
	if err := f.zzk.UpdateService(tenantID, svc, setLockOnCreate, setLockOnUpdate); err != nil {
		glog.Errorf("Could not sync service %s to the coordinator: %s", serviceID, err)
		return err
	}
	return nil
}

// RestoreServices reverts service data
func (f *Facade) RestoreServices(ctx datastore.Context, tenantID string, svcs []service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("RestoreServices"))
	// get pools
	pools, err := f.GetResourcePools(ctx)
	if err != nil {
		glog.Errorf("Could not look up resource pools: %s", err)
		return err
	}
	poolsmap := make(map[string]struct{})
	for _, pool := range pools {
		poolsmap[pool.ID] = struct{}{}
	}
	// remove services for tenant
	if err := f.removeService(ctx, tenantID); err != nil {
		if !datastore.IsErrNoSuchEntity(err) {
			return err
		}
	}
	// get service tree
	svcsmap := make(map[string][]service.Service)
	for _, svc := range svcs {
		svcsmap[svc.ParentServiceID] = append(svcsmap[svc.ParentServiceID], svc)
	}
	// add the services
	var traverse func(parentID string) error
	traverse = func(parentID string) error {
		for _, svc := range svcsmap[parentID] {
			svc.DatabaseVersion = 0
			svc.DesiredState = int(service.SVCStop)
			if _, ok := poolsmap[svc.PoolID]; !ok {
				glog.Warningf("Could not find pool %s for service %s (%s).  Setting pool to default.", svc.PoolID, svc.Name, svc.ID)
				svc.PoolID = "default"
			}
			if err := f.addService(ctx, tenantID, svc, true); err != nil {
				glog.Errorf("Could not restore service %s (%s): %s", svc.Name, svc.ID, err)
				return err
			}
			if err := f.restoreIPs(ctx, &svc); err != nil {
				glog.Warningf("Could not restore address assignments for service %s (%s): %s", svc.Name, svc.ID, err)
			}
			if err := traverse(svc.ID); err != nil {
				return err
			}
			glog.Infof("Restored service %s (%s)", svc.Name, svc.ID)
		}
		return nil
	}
	if err := traverse(""); err != nil {
		glog.Errorf("Error while rolling back services: %s", err)
		return err
	}
	return nil
}

// MigrateServices performs a batch migration on a group of services.
func (f *Facade) MigrateServices(ctx datastore.Context, req dao.ServiceMigrationRequest) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("MigrateServices"))
	var svcAll []service.Service
	// validate service updates
	for _, svc := range req.Modified {
		if _, err := f.validateServiceUpdate(ctx, svc); err != nil {
			glog.Errorf("Could not validate service %s (%s) for update: %s", svc.Name, svc.ID, err)
			return err
		}
		svcAll = append(svcAll, *svc)
	}
	// validate service adds
	for _, svc := range req.Added {
		if err := f.validateServiceAdd(ctx, svc); err != nil {
			glog.Errorf("Could not validate service %s (%s) for add: %s", svc.Name, svc.ID, err)
			return err
		} else if svc.ID, err = utils.NewUUID36(); err != nil {
			glog.Errorf("Could not generate id for service %s: %s", svc.ID, err)
			return err
		}
		svcAll = append(svcAll, *svc)
	}
	// validate service deployments
	for _, sdreq := range req.Deploy {
		svcs, err := f.validateServiceDeployment(ctx, sdreq.ParentID, &sdreq.Service)
		if err != nil {
			glog.Errorf("Could not validate service %s for deployment: %s", sdreq.Service.Name, err)
			return err
		}
		svcAll = append(svcAll, svcs...)
	}
	// validate service migration
	if err := f.validateServiceMigration(ctx, svcAll); err != nil {
		glog.Errorf("Could not validate migration of services: %s", err)
		return err
	}
	glog.Infof("Validation checks passed for service migration")

	// Do migration
	for _, svc := range req.Modified {
		if err := f.MigrateService(ctx, *svc); err != nil {
			return err
		}
	}
	for _, svc := range req.Added {
		if err := f.AddService(ctx, *svc); err != nil {
			return err
		}
	}
	for _, sdreq := range req.Deploy {
		if _, err := f.DeployService(ctx, "", sdreq.ParentID, false, sdreq.Service); err != nil {
			glog.Errorf("Could not deploy service definition {%+v}: %s", sdreq.Service, err)
			return err
		}
	}
	glog.Infof("Service migration completed successfully")
	return nil
}

func (f *Facade) SyncServiceRegistry(ctx datastore.Context, svc *service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("SyncServiceRegistry"))
	tenantID, err := f.GetTenantID(datastore.Get(), svc.ID)
	if err != nil {
		glog.Errorf("Could not check tenant of service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	err = f.zzk.SyncServiceRegistry(tenantID, svc)
	if err != nil {
		glog.Errorf("Could not sync public endpoints for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	return nil
}

// validateServiceDeployment returns the services that will be deployed
func (f *Facade) validateServiceDeployment(ctx datastore.Context, parentID string, sd *servicedefinition.ServiceDefinition) ([]service.Service, error) {
	store := f.serviceStore
	parent, err := store.Get(ctx, parentID)
	if err != nil {
		glog.Errorf("Could not get parent service %s: %s", parentID, err)
		return nil, err
	}
	// recursively create services
	var deployServices func(*service.Service, *servicedefinition.ServiceDefinition) ([]service.Service, error)
	deployServices = func(parent *service.Service, sd *servicedefinition.ServiceDefinition) ([]service.Service, error) {
		var svcs []service.Service
		svc, err := service.BuildService(*sd, parent.ID, parent.PoolID, int(service.SVCStop), parent.DeploymentID)
		if err != nil {
			glog.Errorf("Could not create service %s: %s", sd.Name, err)
			return nil, err
		}
		svcs = append(svcs, *svc)
		for _, sdSvc := range sd.Services {
			childsvcs, err := deployServices(svc, &sdSvc)
			if err != nil {
				return nil, err
			}
			svcs = append(svcs, childsvcs...)
		}
		return svcs, nil
	}
	return deployServices(parent, sd)
}

// validateServiceMigration makes sure there are no collisions with the added/modified
// services.
func (f *Facade) validateServiceMigration(ctx datastore.Context, svcs []service.Service) error {
	svcParentMapNameMap := make(map[string]map[string]struct{})
	endpointMap := make(map[string]struct{})
	for _, svc := range svcs {
		// check for name uniqueness within the set of new/modified/deployed services
		if svcNameMap, ok := svcParentMapNameMap[svc.ParentServiceID]; ok {
			if _, ok := svcNameMap[svc.Name]; ok {
				glog.Errorf("Found a collision for service name %s and parent %s", svc.Name, svc.ParentServiceID)
				return ErrServiceCollision
			}
			svcParentMapNameMap[svc.ParentServiceID][svc.Name] = struct{}{}
		} else {
			svcParentMapNameMap[svc.ParentServiceID] = make(map[string]struct{})
		}

		// check for endpoint name uniqueness within the set of new/modified/deployed services
		for _, ep := range svc.Endpoints {
			if ep.Purpose == "export" {
				if _, ok := endpointMap[ep.Application]; ok {
					glog.Errorf("Endpoint %s in migrated service %s is a duplicate of an endpoint in one of the other migrated services", svc.Name, ep.Application)
					return ErrServiceDuplicateEndpoint
				}
				endpointMap[ep.Application] = struct{}{}
			}
		}

		// check for endpoint name uniqueness btwn this migrated service and the services already defined in
		// the parent application.
		//
		// Note - this is not the most performant way to do this, but migration is not a
		// performance-critical operation, so no-harm/no-foul.
		if err := f.validateServiceEndpoints(ctx, &svc); err != nil {
			glog.Errorf("Migrated service %s has a duplicate endpoint: %s", svc.Name, err)
			return ErrServiceDuplicateEndpoint
		}
	}
	return nil
}

func (f *Facade) RemoveService(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("RemoveService"))
	tenantID, err := f.GetTenantID(ctx, id)
	if err != nil {
		glog.Errorf("Could not get tenant of service %s: %s", id, err)
		return err
	}
	if err := f.lockTenant(ctx, tenantID); err != nil {
		return err
	}
	defer f.retryUnlockTenant(ctx, tenantID, nil, time.Second)
	if err := f.removeService(ctx, id); err != nil {
		glog.Errorf("Could not remove service %s: %s", id, err)
		return err
	}
	if tenantID == id {
		if err := f.dfs.Destroy(tenantID); err != nil {
			glog.Errorf("Could not destroy volume for tenant %s: %s", tenantID, err)
			return err
		}
		f.zzk.RemoveTenantExports(tenantID)
		f.zzk.DeleteRegistryLibrary(tenantID)
	}
	return nil
}

func (f *Facade) removeService(ctx datastore.Context, id string) error {
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
		if err := f.zzk.RemoveServiceEndpoints(svc.ID); err != nil {
			glog.Errorf("Could not remove public endpoints for service %s (%s) from zookeeper: %s", svc.Name, svc.ID, err)
			return err
		}
		if err := f.zzk.RemoveService(svc.PoolID, svc.ID); err != nil {
			glog.Errorf("Could not remove service %s (%s) from zookeeper: %s", svc.Name, svc.ID, err)
			return err
		}

		if err := store.Delete(ctx, svc.ID); err != nil {
			glog.Errorf("Error while removing service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}

		f.serviceCache.RemoveIfParentChanged(svc.ID, svc.ParentServiceID)
		return nil
	}, "removeService")
}

func (f *Facade) GetPoolForService(ctx datastore.Context, id string) (string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetPoolForService"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetImageIDs"))
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

func (f *Facade) GetHealthChecksForService(ctx datastore.Context, serviceID string) (map[string]health.HealthCheck, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetHealthChecksForService"))
	glog.V(3).Infof("Facade.GetHealthChecksForService: id=%s", serviceID)
	store := f.serviceStore
	svc, err := store.Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	return svc.HealthChecks, nil
}

// GetServicesByImage fetches, from Elastic, all services using the supplied image ID.
// Empty parts of the supplied image ID will not be considered.  For example,
// "alskdjalskdjas/myImage:latest", "myImage:latest", "myImage"
func (f *Facade) GetServicesByImage(ctx datastore.Context, imageID string) ([]service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServicesByImage"))
	img, err := commons.ParseImageID(imageID)
	if err != nil {
		return nil, err
	}
	svcs, err := f.getServices(ctx)
	if err != nil {
		return nil, err
	}
	matchingSvcs := make([]service.Service, len(svcs))
	for _, svc := range svcs {
		svcImg, err := commons.ParseImageID(svc.ImageID)
		if err != nil {
			return nil, fmt.Errorf("cannot parse image id for service %s: %s", svc.Name, err)
		}
		if img.User != "" && img.User != svcImg.User {
			continue
		} else if img.Repo != "" && img.Repo != svcImg.Repo {
			continue
		} else if img.Tag != "" && img.Tag != svcImg.Tag {
			continue
		} else {
			matchingSvcs = append(matchingSvcs, svc)
		}
	}
	return matchingSvcs, nil
}

func (f *Facade) GetService(ctx datastore.Context, id string) (*service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetService"))
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

// GetEvaluatedService returns a service where an evaluation has been executed against all templated properties.
func (f *Facade) GetEvaluatedService(ctx datastore.Context, serviceID string, instanceID int) (*service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetEvaluatedService"))
	logger := plog.WithFields(log.Fields{
		"serviceID":  serviceID,
		"instanceID": instanceID,
	})
	logger.Debug("Started Facade.GetEvaluatedService")
	defer logger.Debug("Finished Facade.GetEvaluatedService")

	svc, err := f.GetService(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	if err := f.evaluateService(ctx, svc, instanceID); err != nil {
		return nil, err
	}
	return svc, nil
}

// evaluateService translates the service template fields
func (f *Facade) evaluateService(ctx datastore.Context, svc *service.Service, instanceID int) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("evaluatedService"))

	// service lookup
	getService := func(serviceID string) (service.Service, error) {
		svc := service.Service{}
		result, err := f.GetService(ctx, serviceID)
		if result != nil {
			svc = *result
		}
		return svc, err
	}

	// service child lookup
	getServiceChild := func(parentID, childName string) (service.Service, error) {
		svc := service.Service{}
		result, err := f.FindChildService(ctx, parentID, childName)
		if result != nil {
			svc = *result
		}
		return svc, err
	}
	return svc.Evaluate(getService, getServiceChild, instanceID)
}

// GetServices looks up all services. Allows filtering by tenant ID, name (regular expression), and/or update time.
func (f *Facade) GetServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServices"))
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

// GetAllServices will get all the services
func (f *Facade) GetAllServices(ctx datastore.Context) ([]service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetAllServices"))
	svcs, err := f.getServices(ctx)
	if err != nil {
		return nil, err
	}
	return svcs, nil
}

// GetServicesByPool looks up all services in a particular pool
func (f *Facade) GetServicesByPool(ctx datastore.Context, poolID string) ([]service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServicesByPool"))
	glog.V(3).Infof("Facade.GetServicesByPool")
	store := f.serviceStore
	results, err := store.GetServicesByPool(ctx, poolID)
	if err != nil {
		glog.Error("Facade.GetServicesByPool: err=", err)
		return results, err
	}

	// For performance optimizations, do not retrieve config files, but we do need to fill out
	//    the address assignments.
	for i, _ := range results {
		if err = f.fillServiceAddr(ctx, &results[i]); err != nil {
			return results, err
		}
	}
	return results, nil
}

// GetTaggedServices looks up all services with the specified tags. Allows filtering by tenant ID and/or name (regular expression).
func (f *Facade) GetTaggedServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetTaggedServices"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetTenantID"))
	glog.V(3).Infof("Facade.GetTenantId: %s", serviceID)
	gs := func(id string) (service.Service, error) {
		return f.getService(ctx, id)
	}
	return f.serviceCache.GetTenantID(serviceID, gs)
}

// Get the exported endpoints for a service
func (f *Facade) GetServiceEndpoints(ctx datastore.Context, serviceID string, reportImports, reportExports, validate bool) ([]applicationendpoint.EndpointReport, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceEndpoints"))
	svc, err := f.GetService(ctx, serviceID)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceID, err)
		return nil, err
	}

	states, err := f.zzk.GetServiceStates(svc.PoolID, svc.ID)
	if err != nil {
		err = fmt.Errorf("Could not get service states for service %s (%s): %s", svc.Name, svc.ID, err)
		return nil, err
	}

	appEndpoints := make([]applicationendpoint.ApplicationEndpoint, 0)
	if len(states) == 0 {
		appEndpoints = append(appEndpoints, getEndpointsFromServiceDefinition(svc, reportImports, reportExports)...)
	} else {
		for _, state := range states {
			instanceEndpoints := getEndpointsFromState(state, reportImports, reportExports)
			appEndpoints = append(appEndpoints, instanceEndpoints...)
		}
	}

	sort.Sort(applicationendpoint.ApplicationEndpointSlice(appEndpoints))
	return applicationendpoint.BuildEndpointReports(appEndpoints), nil
}

// Get a list of exported endpoints defined for the service
func getEndpointsFromServiceDefinition(service *service.Service, reportImports, reportExports bool) []applicationendpoint.ApplicationEndpoint {
	var endpoints []applicationendpoint.ApplicationEndpoint
	for _, serviceEndpoint := range service.Endpoints {
		if !reportImports && strings.HasPrefix(serviceEndpoint.Purpose, "import") {
			continue
		} else if !reportExports && strings.HasPrefix(serviceEndpoint.Purpose, "export") {
			continue
		}

		endpoint := applicationendpoint.ApplicationEndpoint{}
		endpoint.ServiceID = service.ID
		endpoint.Application = serviceEndpoint.Application
		endpoint.Purpose = serviceEndpoint.Purpose
		endpoint.Protocol = serviceEndpoint.Protocol
		endpoint.ContainerPort = serviceEndpoint.PortNumber
		endpoint.VirtualAddress = serviceEndpoint.VirtualAddress
		endpoints = append(endpoints, endpoint)
	}
	return endpoints
}

// Get a list of exported endpoints for all service instances based just on the current ServiceState
func getEndpointsFromState(state zkservice.State, reportImports, reportExports bool) []applicationendpoint.ApplicationEndpoint {
	var endpoints []applicationendpoint.ApplicationEndpoint
	if reportImports {
		for _, ep := range state.Imports {
			endpoint := applicationendpoint.ApplicationEndpoint{
				ServiceID:      state.ServiceID,
				Application:    ep.Application,
				Purpose:        ep.Purpose,
				ContainerID:    state.ContainerID,
				ContainerIP:    state.PrivateIP,
				ContainerPort:  ep.PortNumber,
				HostID:         state.HostID,
				VirtualAddress: ep.VirtualAddress,
				InstanceID:     state.InstanceID,
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	if reportExports {
		for _, ep := range state.Exports {
			endpoint := applicationendpoint.ApplicationEndpoint{
				ServiceID:     state.ServiceID,
				Application:   ep.Application,
				Protocol:      ep.Protocol,
				Purpose:       "export",
				ContainerID:   state.ContainerID,
				ContainerIP:   state.PrivateIP,
				ContainerPort: ep.PortNumber,
				HostID:        state.HostID,
				HostIP:        state.HostIP,
				InstanceID:    state.InstanceID,
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

// FindChildService walks services below the service specified by serviceId, checking to see
// if childName matches the service's name. If so, it returns it.
func (f *Facade) FindChildService(ctx datastore.Context, parentServiceID string, childName string) (*service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("FindChildService"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ScheduleService"))
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		return 0, err
	}
	mutex := getTenantLock(tenantID)
	mutex.RLock()
	defer mutex.RUnlock()
	return f.scheduleService(ctx, tenantID, serviceID, autoLaunch, desiredState, false)
}

func (f *Facade) scheduleService(ctx datastore.Context, tenantID, serviceID string, autoLaunch bool, desiredState service.DesiredState, locked bool) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade_scheduleService"))
	glog.V(4).Infof("Facade.ScheduleService %s (%s)", serviceID, desiredState)

	// Build a list of services to be scheduled
	svcs := []service.Service{}
	visitor := func(svc *service.Service) error {
		svcs = append(svcs, *svc)
		return nil
	}
	err := f.walkServices(ctx, serviceID, autoLaunch, visitor, "scheduleService")
	if err != nil {
		glog.Errorf("Could not retrieve service(s) for scheduling %s: %s", serviceID, err)
		return 0, err
	}

	if desiredState != service.SVCStop {
		// Verify that all of the services are ready to be started
		if desiredState.String() == "unknown" {
			return 0, fmt.Errorf("desired state unknown")
		}
		for _, svc := range svcs {
			if err := f.validateServiceStart(ctx, &svc); err != nil {
				glog.Errorf("Service %s (%s) failed validation for start: %s", svc.Name, svc.ID, err)
				return 0, err
			}
		}
	}

	// Schedule the services, calculating the number of affected services as we go
	affected := 0
	for _, svc := range svcs {
		if svc.ID != serviceID && svc.Launch == commons.MANUAL {
			continue
		} else if svc.DesiredState == int(desiredState) {
			continue
		}

		err := f.scheduleOneService(ctx, tenantID, &svc, desiredState)
		if err != nil {
			return affected, err
		}

		affected++
	}
	return affected, nil
}

func (f *Facade) scheduleOneService(ctx datastore.Context, tenantID string, svc *service.Service, desiredState service.DesiredState) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("validateServiceStart"))
	switch desiredState {
	case service.SVCRestart:
		// shutdown all service instances
		if err := f.zzk.StopServiceInstances(svc.PoolID, svc.ID); err != nil {
			return err
		}
		svc.DesiredState = int(service.SVCRun)
	default:
		svc.DesiredState = int(desiredState)
	}

	// write the service into the database
	svc.UpdatedAt = time.Now()
	if err := f.serviceStore.Put(ctx, svc); err != nil {
		glog.Errorf("Facade.scheduleService: Could not create service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	glog.Infof("Scheduled service %s (%s) to %s", svc.Name, svc.ID, service.DesiredState(svc.DesiredState).String())

	if err := f.fillServiceAddr(ctx, svc); err != nil {
		return err
	}
	if err := f.zzk.UpdateService(tenantID, svc, false, false); err != nil {
		glog.Errorf("Facade.scheduleService: Could not sync service %s to the coordinator: %s", svc.ID, err)
		return err
	}
	return nil
}

// WaitService waits for service/s to reach a particular desired state within the designated timeout
func (f *Facade) WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, recursive bool, serviceIDs ...string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("WaitService"))
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

	waitSvcIds := make([]string, len(serviceIDs))
	copy(waitSvcIds, serviceIDs)
	if recursive {
		// Get all child services
		for _, svcID := range serviceIDs {
			childSvcs, err := f.GetServiceList(ctx, svcID)
			if err != nil {
				return err
			}
			for _, childSvc := range childSvcs {
				waitSvcIds = append(waitSvcIds, childSvc.ID)
			}
		}
	}
	for _, serviceID := range waitSvcIds {
		// spawn a goroutine to wait for each particular service
		svc, err := f.GetService(ctx, serviceID)
		if err != nil {
			glog.Errorf("Error while getting service %s: %s", serviceID, err)
			return err
		}
		processing[svc.ID] = struct{}{}
		go func(s *service.Service) {
			err := f.zzk.WaitService(s, dstate, cancel)
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("StopService"))
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, service.SVCRun)
}

func (f *Facade) RestartService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("RestartService"))
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, service.SVCRestart)
}

func (f *Facade) PauseService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("PauseService"))
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, service.SVCPause)
}

func (f *Facade) StopService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("StopService"))
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

func (f *Facade) restoreIPs(ctx datastore.Context, svc *service.Service) error {
	for _, ep := range svc.Endpoints {
		if addrAssign := ep.AddressAssignment; addrAssign.IPAddr != "" {
			glog.Infof("Found an address assignment at %s:%d to endpoint %s for service %s (%s)", addrAssign.IPAddr, ep.AddressConfig.Port, ep.Name, svc.Name, svc.ID)
			ip, err := f.getManualAssignment(ctx, svc.PoolID, addrAssign.IPAddr, ep.AddressConfig.Port)
			if err != nil {
				glog.Warningf("Could not assign ip %s:%d to endpoint %s for service %s (%s): %s", addrAssign.IPAddr, ep.AddressConfig.Port, ep.Name, svc.Name, svc.ID, err)
				continue
			}
			newAddrAssign := addressassignment.AddressAssignment{
				AssignmentType: ip.Type,
				HostID:         ip.HostID,
				PoolID:         svc.PoolID,
				IPAddr:         ip.IP,
				Port:           ep.AddressConfig.Port,
				ServiceID:      svc.ID,
				EndpointName:   ep.Name,
			}
			if _, err := f.assign(ctx, newAddrAssign); err != nil {
				glog.Errorf("Could not restore address assignment for service %s (%s) at %s:%d for endpoint %s: %s", svc.Name, svc.ID, addrAssign.IPAddr, ep.AddressConfig.Port, ep.Name, err)
				return err
			}
			glog.Infof("Restored address assignment for service %s (%s) at %s:%d for endpoint %s", svc.Name, svc.ID, addrAssign.IPAddr, ep.AddressConfig.Port, ep.Name)
		}
	}
	return nil
}

func (f *Facade) AssignIPs(ctx datastore.Context, request addressassignment.AssignmentRequest) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AssignIPs"))
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
		assignments, err := f.GetServiceAddressAssignments(ctx, svc.ID)
		if err != nil {
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
	return f.walkServices(ctx, request.ServiceID, true, visitor, "AssignIPs")
}

// ServiceUse will tag a new image (imageName) in a given registry for a given tenant
// to latest, making sure to push changes to the registry
func (f *Facade) ServiceUse(ctx datastore.Context, serviceID, imageName, registryName string, replaceImgs []string, noOp bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceUse"))
	glog.Infof("Pushing image %s for tenant %s into elastic", imageName, serviceID)
	// Push into elastic
	if err := f.Download(imageName, serviceID); err != nil {
		return err
	}

	if len(replaceImgs) > 0 {
		glog.Infof("Replacing references to images %s with %s for tenant %s", strings.Join(replaceImgs, ", "), imageName, serviceID)
		// Determine what services need to be updated
		svcs, err := f.GetServiceList(ctx, serviceID)
		if err != nil {
			return err
		}
		srchImgs := make(map[string]struct{})
		for _, replaceImg := range replaceImgs {
			img, err := commons.ParseImageID(replaceImg)
			if err != nil {
				return fmt.Errorf("error parsing image ID %s: %s", replaceImg, err)
			}
			srchImgs[img.Repo] = struct{}{}
		}
		newImg, err := commons.ParseImageID(imageName)
		if err != nil {
			return fmt.Errorf("error parsing image ID %s: %s", imageName, err)
		}
		var svcsToUpdate []*service.Service
		for _, svc := range svcs {
			if svc.ImageID == "" {
				continue
			}
			origImg, err := commons.ParseImageID(svc.ImageID)
			if err != nil {
				return fmt.Errorf("error parsing image ID %s: %s", svc.ImageID, err)
			}
			// Only match on repo names
			if _, ok := srchImgs[origImg.Repo]; !ok {
				glog.V(1).Infof("Skipping image replace for service %s due to mismatch: targetImages => %s existingImg => %s", svc.Name, srchImgs, origImg.String())
				continue
			}
			// Change the image in the affected svc to point to our new image
			origImg.Merge(&commons.ImageID{Repo: newImg.Repo})
			glog.Infof("Updating image in service %s to %s", svc.Name, origImg.String())
			svc.ImageID = origImg.String()
			svcsToUpdate = append(svcsToUpdate, svc)
		}

		// Update all the services
		for _, svc := range svcsToUpdate {
			if err = f.UpdateService(ctx, *svc); err != nil {
				return fmt.Errorf("error updating service %s: %s", svc.Name, err)
			}
		}
	}
	return nil
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
	svc, err := store.Get(ctx, id)
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

// getTenantIDs filters the list of all tenant ids
func (f *Facade) getTenantIDs(ctx datastore.Context) ([]string, error) {
	store := f.serviceStore
	results, err := store.GetServices(ctx)
	if err != nil {
		glog.Errorf("Facade.GetServices: %s", err)
		return nil, err
	}
	var svcids []string
	for _, svc := range results {
		if svc.ParentServiceID == "" {
			svcids = append(svcids, svc.ID)
		}
	}
	return svcids, nil
}

// traverse all the services (including the children of the provided service)
func (f *Facade) walkServices(ctx datastore.Context, serviceID string, traverse bool, visitFn service.Visit, callerLabel string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start(fmt.Sprintf("walkServices_%s", callerLabel)))
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

func (f *Facade) fillOutService(ctx datastore.Context, svc *service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("fillOutService"))
	if err := f.fillServiceAddr(ctx, svc); err != nil {
		return err
	}
	if err := f.fillServiceConfigs(ctx, svc); err != nil {
		return err
	}
	return nil
}

func (f *Facade) fillOutServices(ctx datastore.Context, svcs []service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("fillOutServices"))
	for i := range svcs {
		if err := f.fillOutService(ctx, &svcs[i]); err != nil {
			return err
		}
	}
	return nil
}

func (f *Facade) fillServiceAddr(ctx datastore.Context, svc *service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("fillServiceAddr"))
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

// validateServiceEndpoints traverses the service tree for given application and checks for duplicate
// endpoints.
// WARNING: This code is only used in CC 1.1 in the context of service migrations, but it should be
//          added back in CC 1.2 in a more general way (see CC-811 for more information)
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
	}, "validateServiceEndpoints"); err != nil {
		glog.Errorf("Could not walk service tree of %s (%s) with tenant %s: %s", svc.Name, svc.ID, tenantID, err)
		return err
	}
	if vErr.HasError() {
		return vErr
	}
	return nil
}

// GetServiceList gets all child services of the service specified by the
// given service ID, and returns them in a slice
func (f *Facade) GetServiceList(ctx datastore.Context, serviceID string) ([]*service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceList"))
	svcs := make([]*service.Service, 0, 1)

	err := f.walkServices(ctx, serviceID, true, func(childService *service.Service) error {
		// Populate service config + addr info
		if err := f.fillOutService(ctx, childService); err != nil {
			return err
		}

		svcs = append(svcs, childService)
		return nil
	}, "GetServiceList")

	if err != nil {
		return nil, fmt.Errorf("error assembling list of services: %s", err)
	}

	return svcs, nil
}

func (f *Facade) getExcludedVolumes(ctx datastore.Context, serviceID string) []string {
	var (
		volmap  = map[string]struct{}{}
		volumes []string
	)
	f.walkServices(ctx, serviceID, true, func(childService *service.Service) error {
		for _, vol := range childService.Volumes {
			if vol.ExcludeFromBackups {
				volmap[vol.ResourcePath] = struct{}{}
			}
		}
		return nil
	}, "getExcludedVolumes")
	for vol := range volmap {
		volumes = append(volumes, vol)
	}
	return volumes

}

func (f *Facade) GetInstanceMemoryStats(startTime time.Time, instances ...metrics.ServiceInstance) ([]metrics.MemoryUsageStats, error) {
	return f.metricsClient.GetInstanceMemoryStats(startTime, instances...)
}

// Get all the service details
func (f *Facade) GetAllServiceDetails(ctx datastore.Context) ([]service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetAllServiceDetails"))
	return f.serviceStore.GetAllServiceDetails(ctx)
}

// GetServiceDetails returns the details of a particular service
func (f *Facade) GetServiceDetails(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceDetails"))
	return f.serviceStore.GetServiceDetails(ctx, serviceID)
}

// GetServiceDetailsAncestry returns a service and its ancestors
func (f *Facade) GetServiceDetailsAncestry(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceDetailsAncestry"))
	s, err := f.serviceStore.GetServiceDetails(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	if s.ParentServiceID != "" {
		ps, err := f.GetServiceDetailsAncestry(ctx, s.ParentServiceID)
		if err != nil {
			return nil, err
		}
		s.Parent = ps
	}

	return s, nil
}

// Get the details of the child services for the given parent
func (f *Facade) GetServiceDetailsByParentID(ctx datastore.Context, parentID string) ([]service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceDetailsByParentID"))
	return f.serviceStore.GetServiceDetailsByParentID(ctx, parentID)
}

// Get the monitoring profile of a given service
func (f *Facade) GetServiceMonitoringProfile(ctx datastore.Context, serviceID string) (*domain.MonitorProfile, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceMonitoringProfile"))
	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	return &svc.MonitoringProfile, nil
}

// GetServiceExportedEndpoints returns all the exported endpoints for a service
// and its children if enabled.
func (f *Facade) GetServiceExportedEndpoints(ctx datastore.Context, serviceID string, children bool) ([]service.ExportedEndpoint, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServiceExportedEndpoints"))
	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	eps := f.getServiceExportedEndpoints(*svc)

	if children {
		var set func(serviceID string) error

		set = func(serviceID string) error {
			svcs, err := f.serviceStore.GetChildServices(ctx, serviceID)
			if err != nil {
				return err
			}

			for _, svc := range svcs {
				eps = append(eps, f.getServiceExportedEndpoints(svc)...)
				if err := set(svc.ID); err != nil {
					return err
				}
			}
			return nil
		}

		if err := set(serviceID); err != nil {
			return nil, err
		}
	}

	return eps, nil
}

func (f *Facade) getServiceExportedEndpoints(svc service.Service) []service.ExportedEndpoint {
	eps := []service.ExportedEndpoint{}

	for _, ep := range svc.Endpoints {
		if ep.Purpose == "export" {
			eps = append(eps, service.ExportedEndpoint{
				ServiceID:   svc.ID,
				ServiceName: svc.Name,
				Application: ep.Application,
				Protocol:    ep.Protocol,
			})
		}
	}

	return eps
}

// GetServicePublicEndpoints returns all the endpoints for a service and its
// children if enabled.
func (f *Facade) GetServicePublicEndpoints(ctx datastore.Context, serviceID string, children bool) ([]service.PublicEndpoint, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetServicePublicEndpoints"))
	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	pubeps := f.getServicePublicEndpoints(*svc)

	if children {
		var setChildrenPublicEndpoints func(serviceID string) error

		setChildrenPublicEndpoints = func(serviceID string) error {
			svcs, err := f.serviceStore.GetChildServices(ctx, serviceID)
			if err != nil {
				return err
			}

			for _, svc := range svcs {
				pubeps = append(pubeps, f.getServicePublicEndpoints(svc)...)
				if err := setChildrenPublicEndpoints(svc.ID); err != nil {
					return err
				}
			}

			return nil
		}

		if err := setChildrenPublicEndpoints(serviceID); err != nil {
			return nil, err
		}
	}

	return pubeps, nil
}

func (f *Facade) getServicePublicEndpoints(svc service.Service) []service.PublicEndpoint {
	pubs := []service.PublicEndpoint{}

	for _, ep := range svc.Endpoints {
		for _, vhost := range ep.VHostList {
			pubs = append(pubs, service.PublicEndpoint{
				ServiceID:   svc.ID,
				ServiceName: svc.Name,
				Application: ep.Application,
				Protocol:    "https",
				VHostName:   vhost.Name,
				Enabled:     vhost.Enabled,
			})
		}

		for _, port := range ep.PortList {
			pub := service.PublicEndpoint{
				ServiceID:   svc.ID,
				ServiceName: svc.Name,
				Application: ep.Application,
				PortAddress: port.PortAddr,
				Enabled:     port.Enabled,
			}

			if strings.HasPrefix(port.Protocol, "http") {
				pub.Protocol = port.Protocol
			} else if port.UseTLS {
				pub.Protocol = "Other, secure (TLS)"
			} else {
				pub.Protocol = "Other, non-secure"
			}

			pubs = append(pubs, pub)
		}
	}

	return pubs
}
