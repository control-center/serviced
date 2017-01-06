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
	"path"
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
	ErrEmergencyShutdownNoOp    = errors.New("facade: cannot perform operation, emergency shutdown flag is set")
)

// AddService adds a service; return error if service already exists
func (f *Facade) AddService(ctx datastore.Context, svc service.Service) (err error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AddService"))
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

	// remove any BuiltIn enabled monitoring configs
	metricConfigs := []domain.MetricConfig{}
	for _, mc := range svc.MonitoringProfile.MetricConfigs {
		if mc.ID == "metrics" {
			continue
		}

		metrics := []domain.Metric{}
		for _, m := range mc.Metrics {
			if !m.BuiltIn {
				metrics = append(metrics, m)
			}
		}
		mc.Metrics = metrics
		metricConfigs = append(metricConfigs, mc)
	}
	svc.MonitoringProfile.MetricConfigs = metricConfigs

	graphs := []domain.GraphConfig{}
	for _, g := range svc.MonitoringProfile.GraphConfigs {
		if !g.BuiltIn {
			graphs = append(graphs, g)
		}
	}
	svc.MonitoringProfile.GraphConfigs = graphs

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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.UpdateService"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.MigrateService"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.updateService"))
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
			f.zzk.UpdateService(ctx, tenantID, cursvc, false, false)
		}
	}

	f.poolCache.SetDirty()

	// sync the service with the coordinator
	if err := f.syncService(ctx, tenantID, svc.ID, setLockOnUpdate, setLockOnUpdate); err != nil {
		glog.Errorf("Could not sync service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	glog.Infof("Synced service %s (%s) to the coordinator", svc.Name, svc.ID)
	return nil
}

func (f *Facade) validateServiceUpdate(ctx datastore.Context, svc *service.Service) (*service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.validateServiceUpdate"))
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

	// remove any BuiltIn enabled monitoring configs
	metricConfigs := []domain.MetricConfig{}
	for _, mc := range svc.MonitoringProfile.MetricConfigs {
		if mc.ID == "metrics" {
			continue
		}

		metrics := []domain.Metric{}
		for _, m := range mc.Metrics {
			if !m.BuiltIn {
				metrics = append(metrics, m)
			}
		}
		mc.Metrics = metrics
		metricConfigs = append(metricConfigs, mc)
	}
	svc.MonitoringProfile.MetricConfigs = metricConfigs

	graphs := []domain.GraphConfig{}
	for _, g := range svc.MonitoringProfile.GraphConfigs {
		if !g.BuiltIn {
			graphs = append(graphs, g)
		}
	}
	svc.MonitoringProfile.GraphConfigs = graphs

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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.validateServiceName"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.validateServiceTenant"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.validateServiceStart"))
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

	if svc.EmergencyShutdown {
		return ErrEmergencyShutdownNoOp
	}
	return nil
}

// syncService syncs service data from the database into the coordinator.
func (f *Facade) syncService(ctx datastore.Context, tenantID, serviceID string, setLockOnCreate, setLockOnUpdate bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.syncService"))
	svc, err := f.GetService(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not get service %s to sync: %s", serviceID, err)
		return err
	}
	if err := f.zzk.UpdateService(ctx, tenantID, svc, setLockOnCreate, setLockOnUpdate); err != nil {
		glog.Errorf("Could not sync service %s to the coordinator: %s", serviceID, err)
		return err
	}
	return nil
}

// RestoreServices reverts service data
func (f *Facade) RestoreServices(ctx datastore.Context, tenantID string, svcs []service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RestoreServices"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.MigrateServices"))
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
	if err := f.validateServiceMigration(ctx, svcAll, req.ServiceID); err != nil {
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.SyncServiceRegistry"))
	tenantID, err := f.GetTenantID(datastore.Get(), svc.ID)
	if err != nil {
		glog.Errorf("Could not check tenant of service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	err = f.zzk.SyncServiceRegistry(ctx, tenantID, svc)
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
		// Evaluate templated endpoints
		if err = f.evaluateEndpointTemplates(ctx, svc); err != nil {
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
func (f *Facade) validateServiceMigration(ctx datastore.Context, svcs []service.Service, tenantID string) error {
	svcParentMapNameMap := make(map[string]map[string]struct{})
	endpointMap := make(map[string]string)
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
					glog.Errorf("Endpoint %s in migrated service %s is a duplicate of an endpoint in one of the other migrated services", ep.Application, svc.Name)
					return ErrServiceDuplicateEndpoint
				}
				endpointMap[ep.Application] = svc.ID
			}
		}
	}

	// check for endpoint name uniqueness btwn the migrated service and the services already defined in
	// the parent application.
	alleps, err := f.GetServiceExportedEndpoints(ctx, tenantID, true)
	if err != nil {
		glog.Errorf("Error looking up exported endpoints for tenant %s: %s", tenantID, err)
		return err
	}
	for _, ep := range alleps {
		if _, ok := endpointMap[ep.Application]; ok {
			if ep.ServiceID != endpointMap[ep.Application] {
				glog.Errorf("Endpoint %s in migrated service is a duplicate of an endpoint already in the application", ep.Application)
				return ErrServiceDuplicateEndpoint
			}
		}
	}

	return nil
}

func (f *Facade) RemoveService(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemoveService"))
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

		f.poolCache.SetDirty()

		f.serviceCache.RemoveIfParentChanged(svc.ID, svc.ParentServiceID)
		return nil
	}, "removeService")
}

func (f *Facade) GetPoolForService(ctx datastore.Context, id string) (string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetPoolForService"))
	glog.V(3).Infof("Facade.GetPoolForService: id=%s", id)
	store := f.serviceStore
	svc, err := store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return svc.PoolID, nil
}

func (f *Facade) GetHealthChecksForService(ctx datastore.Context, serviceID string) (map[string]health.HealthCheck, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetHealthChecksForService"))
	glog.V(3).Infof("Facade.GetHealthChecksForService: id=%s", serviceID)
	store := f.serviceStore
	svc, err := store.Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	return svc.HealthChecks, nil
}

func (f *Facade) GetService(ctx datastore.Context, id string) (*service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetService"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetEvaluatedService"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.evaluatedService"))

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
// NOTE: Do NOT use this method unless you absolutely, positively need to get a full copy of every service. At sites
//       with 1000s of services, this can be a really expensive call.
func (f *Facade) GetServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServices"))
	var (
		serviceRequest dao.ServiceRequest
		services       []service.Service
		err            error
	)
	switch v := request.(type) {
	case dao.ServiceRequest:
		serviceRequest = request.(dao.ServiceRequest)
	default:
		err := fmt.Errorf("Bad request type %v: %+v", v, request)
		plog.WithError(err).Debug("Unable to get services")
		return nil, err
	}
	logger := plog.WithFields(log.Fields{
		"tags":         serviceRequest.Tags,
		"tenantid":     serviceRequest.TenantID,
		"updatedsince": int(serviceRequest.UpdatedSince.Seconds()),
		"nameregex":    serviceRequest.NameRegex,
	})
	logger.Debug("Started Facade.GetServices")
	defer logger.Debug("Finished Facade.GetServices")

	store := f.serviceStore
	if serviceRequest.UpdatedSince != 0 {
		services, err = store.GetUpdatedServices(ctx, serviceRequest.UpdatedSince)
		if err != nil {
			logger.WithError(err).Error("Unable to get services changed since")
			return nil, err
		}
	} else {
		services, err = store.GetServices(ctx)
		if err != nil {
			logger.WithError(err).Error("Unable to get all services")
			return nil, err
		}
	}

	// filter by the name provided
	if serviceRequest.NameRegex != "" {
		services, err = filterByNameRegex(serviceRequest.NameRegex, services)
		if err != nil {
			logger.WithError(err).Error("Unable to filter services by name")
			return nil, err
		}
	}

	// filter by the tenantID provided
	if serviceRequest.TenantID != "" {
		services, err = f.filterByTenantID(ctx, serviceRequest.TenantID, services)
		if err != nil {
			logger.WithError(err).Error("Unable to filter services by tenant ID")
			return nil, err
		}
	}

	if err = f.fillOutServices(ctx, services); err != nil {
		return nil, err
	}

	return services, nil
}

// GetAllServices will get all the services
// NOTE: Do NOT use this method unless you absolutely, positively need to get a full copy of every service. At sites
//       with 1000s of services, this can be a really expensive call.
func (f *Facade) GetAllServices(ctx datastore.Context) ([]service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetAllServices"))
	var (
		err  error
		svcs []service.Service
	)
	plog.Debug("Started Facade.GetAllServices")
	defer plog.Debug("Finished Facade.GetAllServices")

	svcs, err = f.getServices(ctx)
	if err != nil {
		return nil, err
	}
	return svcs, nil
}

// GetServicesByPool looks up all services in a particular pool
func (f *Facade) GetServicesByPool(ctx datastore.Context, poolID string) ([]service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServicesByPool"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetTaggedServices"))
	var (
		serviceRequest dao.ServiceRequest
		tags           []string
		logger         *log.Entry
		err            error
		services       []service.Service
	)

	switch v := request.(type) {
	case dao.ServiceRequest:
		serviceRequest = request.(dao.ServiceRequest)
		tags = serviceRequest.Tags
		logger = plog.WithFields(log.Fields{
			"tags":         tags,
			"tenantid":     serviceRequest.TenantID,
			"updatedsince": int(serviceRequest.UpdatedSince.Seconds()),
			"nameregex":    serviceRequest.NameRegex,
		})
	case []string:
		tags = request.([]string)
		logger = plog.WithFields(log.Fields{
			"tags": tags,
		})
	default:
		err := fmt.Errorf("Bad request type %v: %+v", v, request)
		plog.WithError(err).Error("GetTaggedServices failed")
		return nil, err
	}

	logger.Debug("Started Facade.GetTaggedServices")
	defer logger.Debug("Finished Facade.GetTaggedServices")

	store := f.serviceStore
	services, err = store.GetTaggedServices(ctx, tags...)
	if err != nil {
		logger.WithError(err).Error("Unable to get tagged services")
		return nil, err
	}
	if err = f.fillOutServices(ctx, services); err != nil {
		return nil, err
	}

	// filter by the name provided
	if serviceRequest.NameRegex != "" {
		services, err = filterByNameRegex(serviceRequest.NameRegex, services)
		if err != nil {
			logger.WithError(err).Error("Unable to filter services by name")
			return nil, err
		}
	}

	// filter by the tenantID provided
	if serviceRequest.TenantID != "" {
		services, err = f.filterByTenantID(ctx, serviceRequest.TenantID, services)
		if err != nil {
			logger.WithError(err).Error("Unable to filter services by tenant ID")
			return nil, err
		}
	}
	return services, nil
}

// Get a list of tenant IDs
func (f *Facade) GetTenantIDs(ctx datastore.Context) ([]string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetTenantIDs"))
	plog.Debug("Started Facade.GetTenantIDs")
	defer plog.Debug("Finished Facade.GetTenantIDs")

	svcs, err := f.GetServiceDetailsByParentID(ctx, "", 0)
	if err != nil {
		plog.WithError(err).Error("Could not get tenant IDs")
		return nil, err
	}
	tenantIDs := []string{}
	for _, tenant := range svcs {
		tenantIDs = append(tenantIDs, tenant.ID)
	}
	return tenantIDs, nil
}

// The tenant id is the root service uuid. Walk the service tree to root to find the tenant id.
func (f *Facade) GetTenantID(ctx datastore.Context, serviceID string) (string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetTenantID"))
	logger := plog.WithFields(log.Fields{
		"serviceID": serviceID,
	})
	logger.Debug("Started Facade.GetTenantID")
	defer logger.Debug("Finished Facade.GetTenantID")

	gs := func(id string) (*service.ServiceDetails, error) {
		return f.GetServiceDetails(ctx, id)
	}
	return f.serviceCache.GetTenantID(serviceID, gs)
}

// Get the exported endpoints for a service
func (f *Facade) GetServiceEndpoints(ctx datastore.Context, serviceID string, reportImports, reportExports, validate bool) ([]applicationendpoint.EndpointReport, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceEndpoints"))
	logger := plog.WithFields(log.Fields{
		"serviceID":     serviceID,
		"reportImports": reportImports,
		"reportExports": reportExports,
	})
	logger.Debug("Started Facade.GetServiceEndpoints")
	defer logger.Debug("Finished Facade.GetServiceEndpoints")

	svc, err := f.GetService(ctx, serviceID)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceID, err)
		return nil, err
	}

	states, err := f.zzk.GetServiceStates(ctx, svc.PoolID, svc.ID)
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.FindChildService"))
	logger := plog.WithFields(log.Fields{
		"parentServiceID": parentServiceID,
		"childName":       childName,
	})
	logger.Debug("Started Facade.FindChildService")
	defer logger.Debug("Finished Facade.FindChildService")

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
func (f *Facade) ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, synchronous bool, desiredState service.DesiredState) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.ScheduleService"))
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		return 0, err
	}
	mutex := getTenantLock(tenantID)
	mutex.RLock()
	defer mutex.RUnlock()
	return f.scheduleService(ctx, tenantID, serviceID, autoLaunch, synchronous, desiredState, false, false)
}

func (f *Facade) clearEmergencyStopFlag(ctx datastore.Context, tenantID, serviceID string) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.clearEmergencyStopFlag"))
	svcs := []*service.Service{}
	visitor := func(svc *service.Service) error {
		if svc.EmergencyShutdown {
			svcs = append(svcs, svc)
		}
		return nil
	}
	err := f.walkServices(ctx, serviceID, true, visitor, "clearEmergencyStopFlag")
	if err != nil {
		plog.WithError(err).Errorf("Could not retrieve service(s) to clear emergency stop flag")
		return 0, err
	}

	cleared := 0
	for _, svc := range svcs {
		svc.EmergencyShutdown = false
		err = f.updateService(ctx, tenantID, *svc, false, false)
		if err != nil {
			plog.WithField("service", svc.ID).WithError(err).Error("Failed to update database with EmergencyShutdown")
		} else {
			cleared++
		}
	}
	return cleared, nil
}

func (f *Facade) scheduleService(ctx datastore.Context, tenantID, serviceID string, autoLaunch bool, synchronous bool, desiredState service.DesiredState, locked bool, emergency bool) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.scheduleService"))
	logger := plog.WithFields(log.Fields{
		"tenantid":     tenantID,
		"serviceid":    serviceID,
		"desiredstate": desiredState,
		"autolaunch":   autoLaunch,
		"synchronous":  synchronous,
	})
	logger.Info("Started Facade.ScheduleService")

	if desiredState.String() == "unknown" {
		return 0, fmt.Errorf("desired state unknown")
	}

	// Build a list of services to be scheduled
	svcs := []*service.Service{}
	visitor := func(svc *service.Service) error {
		if desiredState != service.SVCStop {
			// Verify that all of the services are ready to be started
			if err := f.validateServiceStart(ctx, svc); err != nil {
				logger.WithError(err).WithField("servicename", svc.Name).WithField("serviceid", svc.ID).Error("Service failed validation for start")
				return err
			}
		}
		svcs = append(svcs, svc)
		return nil
	}
	err := f.walkServices(ctx, serviceID, autoLaunch, visitor, "scheduleService")
	if err != nil {
		logger.WithError(err).Errorf("Could not retrieve service(s) for scheduling")
		return 0, err
	}

	serviceScheduler := func() (int, error) {
		affected := 0
		var errToReturn error = nil
		if emergency {

			// Sort the services by emergency shutdown order
			sort.Sort(service.ByEmergencyShutdown{svcs})

			// Start one group at a time
			if len(svcs) > 0 {
				previousLevel := svcs[0].EmergencyShutdownLevel
				previousStartLevel := svcs[0].StartLevel
				nextBatch := []*service.Service{}
				nextBatchIDs := []string{}
				for _, svc := range svcs {
					currentLevel := svc.EmergencyShutdownLevel
					currentStartLevel := svc.StartLevel
					sameBatch := currentLevel == previousLevel
					if sameBatch && currentLevel == 0 {
						// For emergency shutdown level 0, we group by reverse start level
						sameBatch = currentStartLevel == previousStartLevel
					}
					if sameBatch {
						nextBatch = append(nextBatch, svc)
						nextBatchIDs = append(nextBatchIDs, svc.ID)

						// Set EmergencyShutdown to true for this service and update the database
						svc.EmergencyShutdown = true
						uerr := f.updateService(ctx, tenantID, *svc, false, false)
						if uerr != nil {
							errToReturn = uerr
							logger.WithField("service", svc.ID).WithError(uerr).Error("Failed to update database with EmergencyShutdown")
						}
					} else {
						// Schedule this batch
						levelLogger := logger.WithField("level", previousLevel)
						levelLogger.Info("Shutting down all services at current emergency shutdown level")
						a, serr := scheduleServices(f, nextBatch, ctx, tenantID, serviceID, desiredState)
						if serr != nil {
							errToReturn = serr
							levelLogger.WithError(serr).Error("Error scheduling services to stop")
						} else {
							// Wait for services to change state before continuing
							f.WaitService(ctx, desiredState, f.serviceRunLevelTimeout, false, nextBatchIDs...)
						}
						affected += a
						nextBatch = []*service.Service{svc}
						nextBatchIDs = []string{svc.ID}

						// Set EmergencyShutdown to true for this service and update the database
						svc.EmergencyShutdown = true
						uerr := f.updateService(ctx, tenantID, *svc, false, false)
						if uerr != nil {
							errToReturn = uerr
							logger.WithField("service", svc.ID).WithError(uerr).Error("Failed to update database with EmergencyShutdown")
						}
					}
					previousLevel = currentLevel
					previousStartLevel = currentStartLevel
				}

				// Schedule the last batch
				levelLogger := logger.WithField("level", previousLevel)
				levelLogger.Info("Shutting down all services at current emergency shutdown level")
				a, serr := scheduleServices(f, nextBatch, ctx, tenantID, serviceID, desiredState)
				if serr != nil {
					errToReturn = serr
					levelLogger.WithError(serr).Error("Error scheduling services to stop")
				} else {
					// Wait for services to change state before continuing
					f.WaitService(ctx, desiredState, f.serviceRunLevelTimeout, false, nextBatchIDs...)
				}
				affected += a

			}

		} else {
			affected, errToReturn = scheduleServices(f, svcs, ctx, tenantID, serviceID, desiredState)
		}

		return affected, errToReturn
	}

	affected := 0
	if synchronous {
		logger.Debug("Scheduling services synchronously")
		// Schedule the services synchronously, calculating the number of affected services as we go
		affected, err = serviceScheduler()

	} else {
		logger.Debug("Scheduling services asynchronously")
		// Schedule the services asynchronously, returning the number of services we are attempting to schedule
		affected = len(svcs)
		err = nil
		go serviceScheduler()
	}

	return affected, err
}

func scheduleServices(f *Facade, svcs []*service.Service, ctx datastore.Context, tenantID string, serviceID string,
	desiredState service.DesiredState) (int, error) {
	logger := plog.WithFields(log.Fields{
		"parentserviceid": serviceID,
		"tenantid":        tenantID,
		"desiredstate":    desiredState,
	})
	logger.Debug("Begin scheduleServices")
	servicesToSchedule := make([]*service.Service, 0)
	for _, svc := range svcs {
		if svc.ID != serviceID && svc.Launch == commons.MANUAL {
			continue
		} else if svc.DesiredState == int(desiredState) {
			continue
		}

		err := f.updateDesiredState(ctx, tenantID, svc, desiredState)
		if err != nil {
			logger.WithError(err).WithField("serviceid", svc.ID).WithField("tenantid", tenantID).Errorf("Error scheduling service")
			return 0, err
		}
		if err := f.fillServiceAddr(ctx, svc); err != nil {
			return 0, err
		}
		logger.WithFields(log.Fields{
			"servicename": svc.Name,
			"serviceid":   svc.ID,
		}).Info("Scheduled service")
		servicesToSchedule = append(servicesToSchedule, svc)
	}

	if err := f.zzk.UpdateServices(ctx, tenantID, servicesToSchedule, false, false); err != nil {
		logger.WithError(err).Error("Could not sync service(s)")
		return 0, err
	}

	logger.WithField("count", len(servicesToSchedule)).Debug("Finished scheduleServices")
	return len(servicesToSchedule), nil
}

func (f *Facade) updateDesiredState(ctx datastore.Context, tenantID string, svc *service.Service, desiredState service.DesiredState) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.updateDesiredState"))
	switch desiredState {
	case service.SVCRestart:
		// shutdown all service instances
		if err := f.zzk.StopServiceInstances(ctx, svc.PoolID, svc.ID); err != nil {
			return err
		}
		svc.DesiredState = int(service.SVCRun)
	default:
		svc.DesiredState = int(desiredState)
	}

	// write the service into the database
	if err := f.serviceStore.UpdateDesiredState(ctx, svc.ID, svc.DesiredState); err != nil {
		glog.Errorf("Facade.updateDesiredState: Could not create service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	return nil
}

// Update the serviceCache with values from ZK.
func (f *Facade) UpdateServiceCache(ctx datastore.Context) error {
	svcNodes, err := f.zzk.GetServiceNodes()
	if err != nil {
		return err
	}
	for _, svcNode := range svcNodes {
		f.serviceStore.UpdateDesiredState(ctx, svcNode.ID, svcNode.DesiredState)
	}
	return nil
}

// WaitService waits for service/s to reach a particular desired state within the designated timeout
func (f *Facade) WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, recursive bool, serviceIDs ...string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.WaitService"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.StartService"))
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, request.Synchronous, service.SVCRun)
}

func (f *Facade) RestartService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RestartService"))
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, request.Synchronous, service.SVCRestart)
}

func (f *Facade) PauseService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.PauseService"))
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, request.Synchronous, service.SVCPause)
}

func (f *Facade) StopService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.StopService"))
	return f.ScheduleService(ctx, request.ServiceID, request.AutoLaunch, request.Synchronous, service.SVCStop)
}

func (f *Facade) EmergencyStopService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.EmergencyStopService"))
	tenantID, err := f.GetTenantID(ctx, request.ServiceID)
	if err != nil {
		return 0, err
	}
	mutex := getTenantLock(tenantID)
	mutex.RLock()
	defer mutex.RUnlock()
	return f.scheduleService(ctx, tenantID, request.ServiceID, request.AutoLaunch, request.Synchronous, service.SVCStop, false, true)
}

// ClearEmergencyStopFlag sets EmergencyStop to false for all services on the tenant that have it set to true
func (f *Facade) ClearEmergencyStopFlag(ctx datastore.Context, serviceID string) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.ClearEmergencyStopFlag"))
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		return 0, err
	}
	mutex := getTenantLock(tenantID)
	mutex.RLock()
	defer mutex.RUnlock()
	return f.clearEmergencyStopFlag(ctx, tenantID, serviceID)
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AssignIPs"))
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
			f.RestartService(ctx, dao.ScheduleServiceRequest{svc.ID, false, true})
		}

		return nil
	}

	// traverse all the services
	return f.walkServices(ctx, request.ServiceID, true, visitor, "AssignIPs")
}

// ServiceUse will tag a new image (imageName) in a given registry for a given tenant
// to latest, making sure to push changes to the registry
func (f *Facade) ServiceUse(ctx datastore.Context, serviceID, imageName, registryName string, replaceImgs []string, noOp bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.ServiceUse"))
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

func (f *Facade) filterDetailsByTenantID(ctx datastore.Context, matchTenantID string, services []service.ServiceDetails) ([]service.ServiceDetails, error) {
	matches := []service.ServiceDetails{}
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.getServices"))
	plog.Debug("Started Facade.getServices")
	defer plog.Debug("Finished Facade.getServices")

	store := f.serviceStore
	results, err := store.GetServices(ctx)
	if err != nil {
		plog.WithError(err).Error("Unable to get a list of all services")
		return results, err
	}
	return results, nil
}

// traverse all the services (including the children of the provided service)
func (f *Facade) walkServices(ctx datastore.Context, serviceID string, traverse bool, visitFn service.Visit, callerLabel string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start(fmt.Sprintf("Facade.walkServices_%s", callerLabel)))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.fillOutService"))
	if err := f.fillServiceAddr(ctx, svc); err != nil {
		return err
	}
	if err := f.fillServiceConfigs(ctx, svc); err != nil {
		return err
	}
	return nil
}

func (f *Facade) fillOutServices(ctx datastore.Context, svcs []service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.fillOutServices"))
	for i := range svcs {
		if err := f.fillOutService(ctx, &svcs[i]); err != nil {
			return err
		}
	}
	return nil
}

func (f *Facade) fillServiceAddr(ctx datastore.Context, svc *service.Service) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.fillServiceAddr"))
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

// GetServiceList gets all child services of the service specified by the
// given service ID, and returns them in a slice
func (f *Facade) GetServiceList(ctx datastore.Context, serviceID string) ([]*service.Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceList"))
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
func (f *Facade) GetAllServiceDetails(ctx datastore.Context, since time.Duration) ([]service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetAllServiceDetails"))
	return f.serviceStore.GetAllServiceDetails(ctx, since)
}

// GetServiceDetails returns the details of a particular service
func (f *Facade) GetServiceDetails(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceDetails"))
	return f.serviceStore.GetServiceDetails(ctx, serviceID)
}

// GetServiceDetailsAncestry returns a service and its ancestors
func (f *Facade) GetServiceDetailsAncestry(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceDetailsAncestry"))
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
func (f *Facade) GetServiceDetailsByParentID(ctx datastore.Context, parentID string, since time.Duration) ([]service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceDetailsByParentID"))
	return f.serviceStore.GetServiceDetailsByParentID(ctx, parentID, since)
}

// Get the details of all services for the specified tenant
func (f *Facade) GetServiceDetailsByTenantID(ctx datastore.Context, tenantID string) ([]service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceDetailsByTenantID"))
	svcs, err := f.serviceStore.GetAllServiceDetails(ctx, 0)
	if err != nil {
		return nil, err
	}
	return f.filterDetailsByTenantID(ctx, tenantID, svcs)
}

// Get the monitoring profile of a given service
func (f *Facade) GetServiceMonitoringProfile(ctx datastore.Context, serviceID string) (*domain.MonitorProfile, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceMonitoringProfile"))
	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	return &svc.MonitoringProfile, nil
}

// GetServiceExportedEndpoints returns all the exported endpoints for a service
// and its children if enabled.
func (f *Facade) GetServiceExportedEndpoints(ctx datastore.Context, serviceID string, children bool) ([]service.ExportedEndpoint, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceExportedEndpoints"))

	// get all the exported endpoints and map them to their service
	alleps, err := f.serviceStore.GetAllExportedEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	epmap := make(map[string][]service.ExportedEndpoint)
	for i, ep := range alleps {
		epmap[ep.ServiceID] = append(epmap[ep.ServiceID], alleps[i])
	}

	// get all the endpoints for that service
	result, ok := epmap[serviceID]
	if ok {
		delete(epmap, serviceID)
	}

	if children {
		// get the tenant id and service path to the service in order to find
		// the service's children.
		tenantID, svcPath, err := f.getServicePath(ctx, serviceID)
		if err != nil {
			return nil, err
		}

		for id, eps := range epmap {
			// add the endpoints with the matching tenant id and share the same
			// path prefix.
			tid, spth, err := f.getServicePath(ctx, id)
			if err != nil {
				return nil, err
			}
			if tid == tenantID && strings.HasPrefix(spth, svcPath+"/") {
				result = append(result, eps...)
			}
		}
	}

	return result, nil
}

// GetServicePublicEndpoints returns all the endpoints for a service and its
// children if enabled.
func (f *Facade) GetServicePublicEndpoints(ctx datastore.Context, serviceID string, children bool) ([]service.PublicEndpoint, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServicePublicEndpoints"))

	pubeps := []service.PublicEndpoint{}
	if children {
		allPeps, err := f.GetAllPublicEndpoints(ctx)
		if err != nil {
			return pubeps, err
		}
		for _, pep := range allPeps {
			if pep.ServiceID == serviceID {
				pubeps = append(pubeps, pep)
			} else {
				// Determine if serviceID is a parent
				detail, err := f.GetServiceDetailsAncestry(ctx, pep.ServiceID)
				if err != nil {
					return pubeps, err
				}
				for detail != nil {
					if detail.ID == serviceID {
						pubeps = append(pubeps, pep)
						break
					}
					detail = detail.Parent
				}
			}
		}
	} else {
		svc, err := f.serviceStore.Get(ctx, serviceID)

		if err != nil {
			return nil, err
		}
		pubeps = f.getServicePublicEndpoints(*svc)
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

// CountDescendantStates returns the count of descendants of a service in terms
// of their Launch (auto/manual) and their DesiredState. This is primarily for
// use by the UI, so that it can know how many descendants a start/stop action
// will affect.
func (f *Facade) CountDescendantStates(ctx datastore.Context, serviceID string) (map[string]map[int]int, error) {
	result := make(map[string]map[int]int)
	f.walkServices(ctx, serviceID, true, func(svc *service.Service) error {
		if svc.ID == serviceID {
			// Ignore the parent service
			return nil
		}
		if svc.Startup == "" {
			// Ignore folder services
			return nil
		}
		m, ok := result[svc.Launch]
		if !ok {
			m = make(map[int]int)
			result[svc.Launch] = m
		}
		m[svc.DesiredState]++
		return nil
	}, "descendantStatus")
	return result, nil
}

// ResolveServicePath resolves a service path (e.g., "infrastructure/mariadb")
// to zero or more service details with their ancestry populated.
func (f *Facade) ResolveServicePath(ctx datastore.Context, svcPath string) ([]service.ServiceDetails, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.ResolveServicePath"))
	var (
		parent  string
		current string
		result  = []service.ServiceDetails{}
	)
	plog := plog.WithFields(log.Fields{
		"svcpath": svcPath,
	})

	// Empty paths match nothing
	if isEmptyPath(svcPath) {
		plog.Debug("Empty path produced empty result")
		return result, nil
	}

	// Clean up trailing slashes and lowercase the requested path
	svcPath = strings.TrimRight(svcPath, "/")
	svcPath = strings.ToLower(svcPath)

	// First pass: get all services that match either ID exactly or name by substring.
	// If it's a single-segment query with a leading slash, it indicates that
	// prefix matching should be used instead of substring matching.
	parent, current = path.Split(svcPath)
	prefix := parent == "/"
	details, err := f.serviceStore.GetServiceDetailsByIDOrName(ctx, current, prefix)
	if err != nil {
		return nil, err
	}
	plog.WithFields(log.Fields{
		"svcPath": svcPath,
		"current": current,
		"prefix":  prefix,
		"matches": len(details),
	}).Debug("Found possible service matches")

	// Populate the ancestry for all of the found services, so we can check
	// their parents
	for _, detail := range details {
		d, err := f.GetServiceDetailsAncestry(ctx, detail.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, *d)
	}

	// Now walk up the path, filtering parents as we go
	level := 1
	for !isEmptyPath(parent) {
		// Split the path to get the segment at this level
		parent, current = path.Split(strings.TrimRight(parent, "/"))

		// Technically this won't ever be needed, as it gets lowered in cli/cmd
		// before it gets here, but for the sake of local clarity...
		current = strings.ToLower(current)

		filtered := make([]service.ServiceDetails, 0)

		// Walk up parents to the current level and check their names to filter
		// the list of potentials
		for _, d := range result {
			p := &d
			for i := 0; i < level; i++ {
				p = p.Parent
				if p == nil {
					break
				}
			}

			// If the parent name at this level matches OR this is the last
			// segment and it matches the deployment ID, it's considered
			// a match
			if (p != nil && strings.ToLower(p.Name) == current) || (isEmptyPath(parent) && strings.ToLower(d.DeploymentID) == current) {
				filtered = append(filtered, d)
			}
		}
		result = filtered
		level++
	}
	plog.WithField("results", len(result)).Debug("Filtered service matches by parent")

	return result, nil
}

func isEmptyPath(p string) bool {
	return p == "" || p == "/"
}
